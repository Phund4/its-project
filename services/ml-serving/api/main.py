"""ML serving API: incident classification and congestion score."""

from __future__ import annotations

import io
import json
import logging
import os
import sys
import threading
import time
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
# ROOT = services/ml-serving; репозиторий — на два уровня выше (или REPO_ROOT из окружения для Docker).
_repo_root = os.environ.get("REPO_ROOT", "").strip()
REPO_ROOT = Path(_repo_root).resolve() if _repo_root else ROOT.parent.parent

try:
    from dotenv import load_dotenv
except ImportError:
    pass
else:
    _env_file = os.environ.get("ENV_FILE", str(ROOT / ".env"))
    load_dotenv(_env_file)

import httpx
import torch
import torch.nn as nn
from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from fastapi.responses import JSONResponse, Response
from prometheus_client import CONTENT_TYPE_LATEST, generate_latest
from PIL import Image

sys.path.insert(0, str(ROOT))
from inference_core import load_checkpoint_auto, make_transform

app = FastAPI(title="ITS ML Serving", version="0.1")
_log = logging.getLogger("ml-serving")

_acc_model: nn.Module | None = None
_acc_img: int = 128
_acc_tf = None
_crash_idx: int = 0
_cong_model: nn.Module | None = None
_cong_img: int = 128
_cong_tf = None

_cong_lock = threading.Lock()
_cong_cache: dict[str, tuple[float, dict]] = {}

_acc_ckpt_used: str = ""
_cong_ckpt_used: str = ""
_winners_json_path: str = ""
_acc_from_winners: bool = False
_cong_from_winners: bool = False


def _congestion_interval_sec() -> float:
    v = os.environ.get("CONGESTION_INTERVAL_SEC", "2").strip()
    try:
        x = float(v)
        return x if x > 0 else 2.0
    except ValueError:
        return 2.0


def _resolve_checkpoint_path(raw: str) -> Path:
    """Путь из env или winners.json: абсолютный или относительно корня репозитория."""
    p = Path(raw.strip())
    if p.is_file():
        return p.resolve()
    q = (REPO_ROOT / raw.strip().lstrip("/")).resolve()
    return q


def _winners_default_path() -> Path:
    return Path(os.environ.get("WINNERS_JSON", str(REPO_ROOT / ".data" / "ml-experiments" / "winners.json")))


def _load_winners_checkpoints() -> tuple[Path | None, Path | None]:
    path = _winners_default_path()
    if not path.is_file():
        return None, None
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None, None
    acc_s = (data.get("accident") or {}).get("checkpoint")
    cong_s = (data.get("congestion") or {}).get("checkpoint")
    acc_p = _resolve_checkpoint_path(acc_s) if isinstance(acc_s, str) and acc_s.strip() else None
    cong_p = _resolve_checkpoint_path(cong_s) if isinstance(cong_s, str) and cong_s.strip() else None
    if acc_p is not None and not acc_p.is_file():
        acc_p = None
    if cong_p is not None and not cong_p.is_file():
        cong_p = None
    return acc_p, cong_p


@app.on_event("startup")
def startup() -> None:
    global _acc_model, _acc_img, _acc_tf, _crash_idx, _cong_model, _cong_img, _cong_tf
    global _acc_ckpt_used, _cong_ckpt_used, _winners_json_path, _acc_from_winners, _cong_from_winners

    _acc_ckpt_used = ""
    _cong_ckpt_used = ""
    _winners_json_path = ""
    _acc_from_winners = False
    _cong_from_winners = False

    default_acc = REPO_ROOT / ".data" / "ml-experiments" / "artifacts" / "accident" / "baseline-cnn" / "best.pt"
    default_cong = REPO_ROOT / ".data" / "ml-experiments" / "artifacts" / "congestion" / "tiny-cnn" / "best.pt"

    w_acc, w_cong = _load_winners_checkpoints()
    wp = _winners_default_path()
    if wp.is_file():
        _winners_json_path = str(wp.resolve())

    acc_env = os.environ.get("ACCIDENT_CKPT", "").strip()
    cong_env = os.environ.get("CONGESTION_CKPT", "").strip()

    if acc_env:
        acc_path = _resolve_checkpoint_path(acc_env)
    elif w_acc is not None:
        acc_path = w_acc
        _acc_from_winners = True
    else:
        acc_path = default_acc

    if cong_env:
        cong_path = _resolve_checkpoint_path(cong_env)
    elif w_cong is not None:
        cong_path = w_cong
        _cong_from_winners = True
    else:
        cong_path = default_cong

    if acc_path.is_file():
        _acc_model, _acc_img, meta = load_checkpoint_auto(acc_path)
        _acc_tf = make_transform(_acc_img)
        _crash_idx = int(meta.get("class_to_idx", {}).get("crash", 0))
        _acc_ckpt_used = str(acc_path.resolve())
    if cong_path.is_file():
        _cong_model, _cong_img, _ = load_checkpoint_auto(cong_path)
        _cong_tf = make_transform(_cong_img)
        _cong_ckpt_used = str(cong_path.resolve())

    if not os.environ.get("ML_GATEWAY_URL", "").strip():
        _log.warning(
            "ML_GATEWAY_URL не задан: POST /v1/process отдаёт только JSON; "
            "ml-gateway и Kafka (its.video.ingest) не получают события. "
            "Задайте переменную окружения или заполните services/ml-serving/.env"
        )


@app.get("/metrics")
def metrics():
    """Минимальный exposition для Prometheus (up/scrape)."""
    return Response(content=generate_latest(), media_type=CONTENT_TYPE_LATEST)


@app.get("/health")
def health():
    return {
        "accident_loaded": _acc_model is not None,
        "congestion_loaded": _cong_model is not None,
        "congestion_interval_sec": _congestion_interval_sec(),
        "ml_gateway_push": bool(os.environ.get("ML_GATEWAY_URL", "").strip()),
        "accident_checkpoint": _acc_ckpt_used or None,
        "congestion_checkpoint": _cong_ckpt_used or None,
        "winners_json": _winners_json_path or None,
        "accident_from_winners_json": _acc_from_winners,
        "congestion_from_winners_json": _cong_from_winners,
        "ml_gateway_configured": bool(os.environ.get("ML_GATEWAY_URL", "").strip()),
    }


def _bytes_to_tensor(data: bytes, tf, device: torch.device) -> torch.Tensor:
    img = Image.open(io.BytesIO(data)).convert("RGB")
    return tf(img).unsqueeze(0).to(device)


def _predict_incident(raw: bytes) -> dict:
    if _acc_model is None or _acc_tf is None:
        raise HTTPException(503, "accident model not loaded; set ACCIDENT_CKPT")
    x = _bytes_to_tensor(raw, _acc_tf, torch.device("cpu"))
    with torch.no_grad():
        logits = _acc_model(x)
        prob = torch.softmax(logits, dim=1)[0, _crash_idx].item()
        pred = int(logits.argmax(1).item())
    has_incident = pred == _crash_idx
    return {
        "crash_probability": prob,
        "predicted_class_index": pred,
        "crash_class_index": _crash_idx,
        "label": "crash" if has_incident else "normal",
        "has_incident": has_incident,
    }


def _predict_congestion(raw: bytes) -> dict:
    if _cong_model is None or _cong_tf is None:
        raise HTTPException(503, "congestion model not loaded; set CONGESTION_CKPT")
    x = _bytes_to_tensor(raw, _cong_tf, torch.device("cpu"))
    with torch.no_grad():
        score = float(_cong_model(x).item())
    return {"congestion_score": score, "note": "proxy [0,1] from lab regressor"}


def _congestion_for_pair(raw: bytes, segment_id: str, camera_id: str) -> dict:
    key = f"{segment_id.strip()}|{camera_id.strip()}"
    interval = _congestion_interval_sec()
    now = time.monotonic()
    with _cong_lock:
        if key in _cong_cache:
            last_t, cached = _cong_cache[key]
            if now - last_t < interval:
                return dict(cached)
        out = _predict_congestion(raw)
        _cong_cache[key] = (now, dict(out))
        return out


def _process_payload(raw: bytes, segment_id: str = "", camera_id: str = "") -> dict:
    incident = _predict_incident(raw)
    seg = (segment_id or "").strip()
    cam = (camera_id or "").strip()
    congestion = _congestion_for_pair(raw, seg, cam) if seg and cam else _predict_congestion(raw)
    return {"incident": incident, "congestion": congestion}


async def _push_to_ml_gateway(ml_payload: dict, segment_id: str, camera_id: str, s3_key: str, observed_at: str) -> None:
    base = os.environ.get("ML_GATEWAY_URL", "").strip().rstrip("/")
    if not base:
        return
    path = os.environ.get("ML_GATEWAY_PATH", "/v1/road-events").strip()
    if not path.startswith("/"):
        path = "/" + path
    timeout = float(os.environ.get("ML_GATEWAY_TIMEOUT", "10"))
    body = {
        "segment_id": segment_id,
        "camera_id": camera_id,
        "observed_at": observed_at,
        "s3_key": s3_key,
        "ml": ml_payload,
    }
    try:
        async with httpx.AsyncClient(timeout=timeout) as client:
            r = await client.post(f"{base}{path}", json=body)
    except httpx.RequestError as e:
        raise HTTPException(502, f"ml_gateway unreachable: {e}") from e
    if r.status_code < 200 or r.status_code >= 300:
        raise HTTPException(502, f"ml_gateway HTTP {r.status_code}: {r.text[:512]}")


@app.post("/v1/process")
async def process(
    image: UploadFile = File(...),
    segment_id: str | None = Form(None),
    camera_id: str | None = Form(None),
    s3_key: str | None = Form(None),
    observed_at: str | None = Form(None),
):
    raw = await image.read()
    gw = os.environ.get("ML_GATEWAY_URL", "").strip()
    seg = (segment_id or "").strip()
    cam = (camera_id or "").strip()
    if gw:
        if not seg or not cam:
            raise HTTPException(400, "segment_id and camera_id are required when ML_GATEWAY_URL is set")
        payload = _process_payload(raw, seg, cam)
        obs = (observed_at or "").strip() or datetime.now(timezone.utc).isoformat()
        key = (s3_key or "").strip()
        await _push_to_ml_gateway(payload, seg, cam, key, obs)
        return Response(status_code=204)
    return JSONResponse(_process_payload(raw, seg, cam))
