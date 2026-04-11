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
from sklearn.metrics import accuracy_score, f1_score, precision_score, recall_score
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


class BinaryFolderImageDataset(Dataset):
    def __init__(self, split_root: Path, transform):
        exts = {".png", ".jpg", ".jpeg", ".webp"}
        crash_dirs = ("Accident", "accident", "Crash", "crash")
        normal_dirs = ("Non Accident", "non_accident", "non-accident", "normal", "Normal")

        self.items: list[tuple[Path, int]] = []
        for d in crash_dirs:
            p = split_root / d
            if p.is_dir():
                self.items.extend((f, 0) for f in sorted(p.iterdir()) if f.is_file() and f.suffix.lower() in exts)
                break
        for d in normal_dirs:
            p = split_root / d
            if p.is_dir():
                self.items.extend((f, 1) for f in sorted(p.iterdir()) if f.is_file() and f.suffix.lower() in exts)
                break
        self.tf = transform

    def __len__(self) -> int:
        return len(self.items)

    def __getitem__(self, i: int):
        p, y = self.items[i]
        img = Image.open(p).convert("RGB")
        return self.tf(img), torch.tensor(y, dtype=torch.long)


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


@torch.no_grad()
def eval_accident_binary(model: torch.nn.Module, loader: DataLoader, crash_idx: int, normal_idx: int) -> dict:
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
    crash_probs_sorted = sorted(crash_probs)
    p95_prob = crash_probs_sorted[int(0.95 * (n - 1))] if n > 1 else (crash_probs_sorted[0] if crash_probs_sorted else 0.0)
    out = {
        "accuracy": float(accuracy_score(ys, ps)),
        "precision_crash": float(precision_score(ys, ps, pos_label=crash_idx, zero_division=0)),
        "recall_crash": float(recall_score(ys, ps, pos_label=crash_idx, zero_division=0)),
        "f1_crash": float(f1_score(ys, ps, pos_label=crash_idx, zero_division=0)),
        "f1_normal": float(f1_score(ys, ps, pos_label=normal_idx, zero_division=0)),
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


def evaluate_accident_binary_split(split_root: Path, artifacts_dir: Path, batch_size: int) -> dict:
    """Оценка accident-моделей на папке с подкаталогами Accident / Non Accident (или алиасы)."""
    if not split_root.is_dir():
        raise SystemExit(f"missing accident image split directory: {split_root}")

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
        ds = BinaryFolderImageDataset(split_root, tf)
        if len(ds) == 0:
            raise SystemExit(f"no images found under split: {split_root}")
        loader = DataLoader(ds, batch_size=batch_size)
        metrics = eval_accident_binary(model, loader, crash_idx, normal_idx)
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


def pick_accident_winner_binary(results: dict[str, dict]) -> str:
    return max(
        results.keys(),
        key=lambda k: (
            0.5 * (results[k]["f1_crash"] + results[k]["f1_normal"]),
            results[k]["recall_crash"],
            results[k]["precision_crash"],
            results[k]["accuracy"],
            results[k]["fps"],
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
    p.add_argument(
        "--accident-data",
        type=Path,
        default=ROOT / "data" / "accident" / "video-gt" / "images",
        help="Корень для режима «все кадры normal» (legacy); тест по умолчанию — --accident-test-root.",
    )
    p.add_argument(
        "--cctv-accidents-root",
        type=Path,
        default=ROOT / "data" / "CCTV-accidents",
        help="Валидация accident: подкаталог val/ с классами Accident и Non Accident.",
    )
    p.add_argument(
        "--accident-test-root",
        type=Path,
        default=ROOT / "data" / "accident" / "test",
        help="Тест accident: папки Accident / Non Accident (или алиасы из BinaryFolderImageDataset).",
    )
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

    val_dir = args.cctv_accidents_root / "val"
    has_cctv_val = val_dir.is_dir()
    has_accident_test = args.accident_test_root.is_dir()
    has_video_gt_normal = (args.accident_data / "test" / "normal").is_dir()
    if not (args.congestion_data / "labels" / "test.csv").is_file():
        raise SystemExit(f"missing congestion test split: {args.congestion_data / 'labels' / 'test.csv'}")

    if has_cctv_val ^ has_accident_test:
        raise SystemExit(
            "Задайте оба split для accident или ни одного (тогда — legacy video-gt):\n"
            f"  - валидация: {val_dir}\n"
            f"  - тест: {args.accident_test_root}"
        )

    if has_cctv_val and has_accident_test:
        accident_val = evaluate_accident_binary_split(val_dir, args.accident_artifacts, args.batch_size)
        accident_test = evaluate_accident_binary_split(
            args.accident_test_root, args.accident_artifacts, args.batch_size
        )
        acc_winner = pick_accident_winner_binary(accident_val)
        accident_assumption_val = "validation: ml-experiments/data/CCTV-accidents/val (Accident + Non Accident)"
        accident_assumption_test = "test: ml-experiments/data/accident/test (Accident + Non Accident)"
        accident_assumption_combined = f"{accident_assumption_val}; {accident_assumption_test}"
        mode = "cctv_val_accident_test"
        interpretation = (
            "Победитель accident выбирается по валидации на CCTV-accidents/val. "
            "Итоговые метрики accident — на тесте data/accident/test. "
            "Congestion — только тест (labels/test.csv). Набор data/train не используется."
        )
        accident_track = {
            "validation": {"models": accident_val, "split_path": str(val_dir)},
            "test": {"models": accident_test, "split_path": str(args.accident_test_root)},
            "winner": acc_winner,
        }
        accident_for_winners_json = accident_val
    elif has_video_gt_normal:
        accident_legacy = evaluate_accident_video_gt(args.accident_data, args.accident_artifacts, args.batch_size)
        acc_winner = pick_accident_winner_video_gt(accident_legacy)
        accident_assumption_combined = "all frames labeled normal (no crash in source videos)"
        mode = "video_ground_truth_eval"
        interpretation = (
            "Режим без CCTV-accidents/val и без data/accident/test: только video-gt (все normal). "
            "Для протокола с валидацией и тестом подготовьте data/CCTV-accidents/val и data/accident/test."
        )
        accident_track = {"models": accident_legacy, "winner": acc_winner}
        accident_for_winners_json = accident_legacy
    else:
        raise SystemExit(
            "Нужно одно из:\n"
            f"  (1) валидация + тест accident: {val_dir} и {args.accident_test_root}\n"
            f"  (2) legacy video-gt: {args.accident_data / 'test' / 'normal'}\n"
            "Также нужен congestion test: .../labels/test.csv"
        )

    congestion = evaluate_congestion_models(
        args.congestion_data, args.congestion_artifacts, args.batch_size, split="test"
    )
    cong_winner = pick_congestion_winner_video_gt(congestion)

    assumptions = {
        "accident": accident_assumption_combined,
        "congestion": "test: labels/test.csv (пути в CSV относительно --congestion-data); data/train не используется",
    }

    payload = {
        "mode": mode,
        "interpretation": interpretation,
        "assumptions": assumptions,
        "tracks": {
            "accident": accident_track,
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
            "checkpoint": rel_ckpt(accident_for_winners_json[acc_winner]["checkpoint"]),
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
