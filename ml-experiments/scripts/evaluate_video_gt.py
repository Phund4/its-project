#!/usr/bin/env python3
"""Evaluate checkpoints on video ground truth (all normal, congestion target 0)."""

from __future__ import annotations

import argparse
import json
import sys
import time
from pathlib import Path

import torch
from PIL import Image
from sklearn.metrics import accuracy_score, f1_score
from torch.utils.data import DataLoader, Dataset

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
from inference_core import (
    artifact_subdir,
    evaluate_congestion_models,
    load_checkpoint,
    make_transform,
    summarize_latency,
)


class AllNormalImageDataset(Dataset):
    def __init__(self, images_root: Path, transform, normal_idx: int):
        base = images_root / "test" / "normal"
        exts = {".png", ".jpg", ".jpeg", ".webp"}
        self.paths = sorted(p for p in base.iterdir() if p.is_file() and p.suffix.lower() in exts)
        self.tf = transform
        self.normal_idx = normal_idx

    def __len__(self) -> int:
        return len(self.paths)

    def __getitem__(self, i: int):
        p = self.paths[i]
        img = Image.open(p).convert("RGB")
        return self.tf(img), torch.tensor(self.normal_idx, dtype=torch.long)


@torch.no_grad()
def eval_accident_video_gt(model: torch.nn.Module, loader: DataLoader, crash_idx: int, normal_idx: int) -> dict:
    ys: list[int] = []
    ps: list[int] = []
    crash_probs: list[float] = []
    lats: list[float] = []
    for x, y in loader:
        t0 = time.perf_counter()
        logits = model(x)
        lats.append((time.perf_counter() - t0) * 1000.0)
        prob_crash = torch.softmax(logits, dim=1)[:, crash_idx].cpu().numpy().tolist()
        pred = logits.argmax(1).cpu().numpy().tolist()
        ys.extend(y.numpy().tolist())
        ps.extend(pred)
        crash_probs.extend(prob_crash)
    n = len(ys)
    false_crash = sum(1 for p in ps if p == crash_idx) / n if n else 0.0
    crash_probs_sorted = sorted(crash_probs)
    p95_prob = crash_probs_sorted[int(0.95 * (n - 1))] if n > 1 else (crash_probs_sorted[0] if crash_probs_sorted else 0.0)
    acc = float(accuracy_score(ys, ps))
    out = {
        "accuracy": acc,
        "f1_crash": float(f1_score(ys, ps, pos_label=crash_idx, zero_division=0)),
        "f1_normal": float(f1_score(ys, ps, pos_label=normal_idx, zero_division=0)),
        "false_crash_rate": float(false_crash),
        "mean_crash_probability": float(sum(crash_probs) / n if n else 0.0),
        "p95_crash_probability": float(p95_prob),
        "samples": n,
    }
    out.update(summarize_latency(lats))
    return out


def evaluate_accident_video_gt(data_dir: Path, artifacts_dir: Path, batch_size: int) -> dict:
    out: dict[str, dict] = {}
    for model_name in ("baseline_cnn", "resnet18"):
        ckpt_path = artifacts_dir / artifact_subdir(model_name) / "best.pt"
        if not ckpt_path.is_file():
            raise SystemExit(f"missing checkpoint: {ckpt_path}")
        model, img_size, meta = load_checkpoint(ckpt_path, "accident")
        tf = make_transform(img_size)
        c2i = meta.get("class_to_idx") or {}
        if "crash" not in c2i or "normal" not in c2i:
            raise SystemExit(f"checkpoint missing class_to_idx: {ckpt_path}")
        crash_idx = int(c2i["crash"])
        normal_idx = int(c2i["normal"])
        test_ds = AllNormalImageDataset(data_dir, tf, normal_idx)
        loader = DataLoader(test_ds, batch_size=batch_size)
        metrics = eval_accident_video_gt(model, loader, crash_idx, normal_idx)
        metrics["checkpoint"] = str(ckpt_path)
        metrics["crash_class_index"] = crash_idx
        metrics["normal_class_index"] = normal_idx
        out[model_name] = metrics
    return out


def pick_accident_winner_video_gt(results: dict[str, dict]) -> str:
    return min(
        results.keys(),
        key=lambda k: (
            results[k]["false_crash_rate"],
            -results[k]["accuracy"],
            -results[k]["f1_normal"],
            -results[k]["fps"],
        ),
    )


def pick_congestion_winner_video_gt(results: dict[str, dict]) -> str:
    return min(results.keys(), key=lambda k: (results[k]["mae"], results[k]["rmse"], -results[k]["fps"]))


def main() -> None:
    repo_root = ROOT.parent
    default_artifacts = repo_root / ".data" / "ml-experiments" / "artifacts"
    legacy_artifacts = ROOT / "artifacts"
    default_out_root = repo_root / ".data" / "ml-experiments"

    p = argparse.ArgumentParser()
    p.add_argument("--accident-data", type=Path, default=ROOT / "data" / "accident" / "video-gt" / "images")
    p.add_argument("--congestion-data", type=Path, default=ROOT / "data" / "congestion" / "video-gt")
    p.add_argument("--accident-artifacts", type=Path, default=default_artifacts / "accident")
    p.add_argument("--congestion-artifacts", type=Path, default=default_artifacts / "congestion")
    p.add_argument("--batch-size", type=int, default=16)
    p.add_argument("--output", type=Path, default=default_out_root / "benchmark" / "video_gt_results.json")
    p.add_argument("--winners-output", type=Path, default=default_out_root / "winners.json")
    args = p.parse_args()

    # Backward-compatible fallback: if new .data artifacts are missing,
    # use legacy ml-experiments/artifacts checkpoints.
    if not args.accident_artifacts.exists() and (legacy_artifacts / "accident").exists():
        args.accident_artifacts = legacy_artifacts / "accident"
    if not args.congestion_artifacts.exists() and (legacy_artifacts / "congestion").exists():
        args.congestion_artifacts = legacy_artifacts / "congestion"

    if not (args.accident_data / "test" / "normal").is_dir():
        raise SystemExit(f"run prepare_video_ground_truth.py first; missing {args.accident_data / 'test' / 'normal'}")
    if not (args.congestion_data / "labels" / "test.csv").is_file():
        raise SystemExit(f"missing {args.congestion_data / 'labels' / 'test.csv'}")

    accident = evaluate_accident_video_gt(args.accident_data, args.accident_artifacts, args.batch_size)
    congestion = evaluate_congestion_models(args.congestion_data, args.congestion_artifacts, args.batch_size)
    acc_winner = pick_accident_winner_video_gt(accident)
    cong_winner = pick_congestion_winner_video_gt(congestion)

    payload = {
        "mode": "video_ground_truth_eval",
        "interpretation": (
            "При разметке «на видео везде normal» accuracy — это доля кадров, где argmax == индекс класса normal из чекпойнта. "
            "Для двух классов это же число, что 1 − false_crash_rate (доля предсказаний «crash»). "
            "Если модель на каждом кадре выбирает «crash» и softmax это подтверждает (высокая mean_P_crash), accuracy=0 — "
            "ожидаемый и корректный результат; это не перевёрнутая метрика."
        ),
        "assumptions": {
            "accident": "all frames labeled normal (no crash in source videos)",
            "congestion": "target is 0.0 for every frame (closer model output to 0 is better)",
        },
        "tracks": {
            "accident": {"models": accident, "winner": acc_winner},
            "congestion": {"models": congestion, "winner": cong_winner},
        },
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(payload, indent=2, ensure_ascii=False), encoding="utf-8")

    repo_root = ROOT.parent

    def rel_ckpt(path_str: str) -> str:
        p = Path(path_str).resolve()
        try:
            return str(p.relative_to(repo_root.resolve()))
        except ValueError:
            return str(p)

    winners_payload = {
        "accident": {
            "winner": acc_winner,
            "checkpoint": rel_ckpt(accident[acc_winner]["checkpoint"]),
        },
        "congestion": {
            "winner": cong_winner,
            "checkpoint": rel_ckpt(congestion[cong_winner]["checkpoint"]),
        },
    }
    args.winners_output.parent.mkdir(parents=True, exist_ok=True)
    args.winners_output.write_text(json.dumps(winners_payload, indent=2, ensure_ascii=False), encoding="utf-8")

    print(f"wrote {args.output}")
    print(f"wrote {args.winners_output}")


if __name__ == "__main__":
    main()
