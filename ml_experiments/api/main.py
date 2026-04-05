"""Mock ML API: incident classification and congestion score (loads checkpoints)."""

from __future__ import annotations

import io
import os
import sys
import threading
import time
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

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
from PIL import Image

sys.path.insert(0, str(ROOT))
from inference_core import load_checkpoint_auto, make_transform

app = FastAPI(title="ITS Mock ML", version="0.1")

_acc_model: nn.Module | None = None
_acc_img: int = 128
_acc_tf = None
_crash_idx: int = 0
_cong_model: nn.Module | None = None
_cong_img: int = 128
_cong_tf = None

# Per (segment_id, camera_id): last monotonic time and cached congestion dict (model runs at most every CONGESTION_INTERVAL_SEC).
_cong_lock = threading.Lock()
_cong_cache: dict[str, tuple[float, dict]] = {}


def _congestion_interval_sec() -> float:
    """Возвращает интервал (сек) между пересчётами загруженности для пары segment/camera из CONGESTION_INTERVAL_SEC."""
    v = os.environ.get("CONGESTION_INTERVAL_SEC", "2").strip()
    try:
        x = float(v)
        return x if x > 0 else 2.0
    except ValueError:
        return 2.0


@app.on_event("startup")
def startup() -> None:
    """Загружает чекпойнты ДТП и загруженности при старте приложения."""
    global _acc_model, _acc_img, _acc_tf, _crash_idx, _cong_model, _cong_img, _cong_tf
    acc_path = Path(os.environ.get("ACCIDENT_CKPT", ROOT / "artifacts" / "accident" / "baseline_cnn" / "best.pt"))
    cong_path = Path(os.environ.get("CONGESTION_CKPT", ROOT / "artifacts" / "congestion" / "tiny_cnn" / "best.pt"))
    if acc_path.is_file():
        _acc_model, _acc_img, meta = load_checkpoint_auto(acc_path)
        _acc_tf = make_transform(_acc_img)
        _crash_idx = int(meta.get("class_to_idx", {}).get("crash", 0))
    if cong_path.is_file():
        _cong_model, _cong_img, _ = load_checkpoint_auto(cong_path)
        _cong_tf = make_transform(_cong_img)


@app.get("/health")
def health():
    """Отдаёт JSON о загрузке моделей, интервале congestion и наличии ML_GATEWAY_URL."""
    return {
        "accident_loaded": _acc_model is not None,
        "congestion_loaded": _cong_model is not None,
        "congestion_interval_sec": _congestion_interval_sec(),
        "ml_gateway_push": bool(os.environ.get("ML_GATEWAY_URL", "").strip()),
    }


def _bytes_to_tensor(data: bytes, tf, device: torch.device) -> torch.Tensor:
    """Декодирует байты изображения в тензор батча [1,C,H,W] на указанном device."""
    img = Image.open(io.BytesIO(data)).convert("RGB")
    t = tf(img).unsqueeze(0).to(device)
    return t


def _predict_incident(raw: bytes) -> dict:
    """Запускает классификатор инцидента; возвращает вероятности и метку crash/normal."""
    if _acc_model is None or _acc_tf is None:
        raise HTTPException(503, "accident model not loaded; set ACCIDENT_CKPT")
    device = torch.device("cpu")
    x = _bytes_to_tensor(raw, _acc_tf, device)
    with torch.no_grad():
        logits = _acc_model(x)
        prob = torch.softmax(logits, dim=1)[0, _crash_idx].item()
        pred = int(logits.argmax(1).item())
    return {
        "crash_probability": prob,
        "predicted_class_index": pred,
        "crash_class_index": _crash_idx,
        "label": "crash" if pred == _crash_idx else "normal",
    }


def _predict_congestion(raw: bytes) -> dict:
    """Запускает регрессор загруженности и возвращает congestion_score."""
    if _cong_model is None or _cong_tf is None:
        raise HTTPException(503, "congestion model not loaded; set CONGESTION_CKPT")
    device = torch.device("cpu")
    x = _bytes_to_tensor(raw, _cong_tf, device)
    with torch.no_grad():
        score = float(_cong_model(x).item())
    return {"congestion_score": score, "note": "proxy [0,1] from lab regressor"}


def _congestion_for_pair(raw: bytes, segment_id: str, camera_id: str) -> dict:
    """Считает загруженность не чаще раза в CONGESTION_INTERVAL_SEC на пару (segment_id, camera_id); иначе отдаёт кэш."""
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
    """Собирает словарь с блоками incident и congestion для одного кадра."""
    incident = _predict_incident(raw)
    seg = (segment_id or "").strip()
    cam = (camera_id or "").strip()
    if seg and cam:
        congestion = _congestion_for_pair(raw, seg, cam)
    else:
        congestion = _predict_congestion(raw)
    return {"incident": incident, "congestion": congestion}


async def _push_to_ml_gateway(
    ml_payload: dict,
    segment_id: str,
    camera_id: str,
    s3_key: str,
    observed_at: str,
) -> None:
    """POST JSON в ml_gateway; при ошибке сети или статуса бросает HTTPException 502."""
    base = os.environ.get("ML_GATEWAY_URL", "").strip().rstrip("/")
    if not base:
        return
    path = os.environ.get("ML_GATEWAY_PATH", "/v1/road-events").strip()
    if not path.startswith("/"):
        path = "/" + path
    url = f"{base}{path}"
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
            r = await client.post(url, json=body)
    except httpx.RequestError as e:
        raise HTTPException(502, f"ml_gateway unreachable: {e}") from e
    if r.status_code < 200 or r.status_code >= 300:
        raise HTTPException(502, f"ml_gateway HTTP {r.status_code}: {r.text[:512]}")


@app.post("/v1/incident")
async def incident(image: UploadFile = File(...)):
    """HTTP: только классификация инцидента по одному загруженному изображению."""
    raw = await image.read()
    return _predict_incident(raw)


@app.post("/v1/congestion")
async def congestion(image: UploadFile = File(...)):
    """HTTP: только оценка загруженности по одному изображению."""
    raw = await image.read()
    return _predict_congestion(raw)


@app.post("/v1/process")
async def process(
    image: UploadFile = File(...),
    segment_id: str | None = Form(None),
    camera_id: str | None = Form(None),
    s3_key: str | None = Form(None),
    observed_at: str | None = Form(None),
):
    """Если задан ML_GATEWAY_URL — шлёт результат инференса в шлюз и отвечает 204 без тела.

    Иначе возвращает JSON с полями incident и congestion (локальные клиенты).
    """
    raw = await image.read()
    gw = os.environ.get("ML_GATEWAY_URL", "").strip()
    seg = (segment_id or "").strip()
    cam = (camera_id or "").strip()
    if gw:
        if not seg or not cam:
            raise HTTPException(
                400,
                "segment_id and camera_id form fields are required when ML_GATEWAY_URL is set",
            )
        payload = _process_payload(raw, seg, cam)
        obs = (observed_at or "").strip()
        if not obs:
            obs = datetime.now(timezone.utc).isoformat()
        key = (s3_key or "").strip()
        await _push_to_ml_gateway(payload, seg, cam, key, obs)
        return Response(status_code=204)
    payload = _process_payload(raw, seg, cam)
    return JSONResponse(payload)
