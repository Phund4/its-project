"""Shared PyTorch models and checkpoint loading for API + offline benchmarks."""

from __future__ import annotations

import csv
import statistics
import time
from pathlib import Path

import torch
import torch.nn as nn
from PIL import Image
from sklearn.metrics import mean_absolute_error, root_mean_squared_error
from torch.utils.data import DataLoader, Dataset
from torchvision import models, transforms

ROOT = Path(__file__).resolve().parent


class SmallCNN(nn.Module):
    def __init__(self, num_classes: int = 2):
        super().__init__()
        self.net = nn.Sequential(
            nn.Conv2d(3, 16, 3, padding=1),
            nn.ReLU(inplace=True),
            nn.MaxPool2d(2),
            nn.Conv2d(16, 32, 3, padding=1),
            nn.ReLU(inplace=True),
            nn.MaxPool2d(2),
            nn.Conv2d(32, 64, 3, padding=1),
            nn.ReLU(inplace=True),
            nn.AdaptiveAvgPool2d(1),
        )
        self.fc = nn.Linear(64, num_classes)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.fc(self.net(x).flatten(1))


class TinyRegCNN(nn.Module):
    def __init__(self):
        super().__init__()
        self.net = nn.Sequential(
            nn.Conv2d(3, 16, 3, padding=1),
            nn.ReLU(inplace=True),
            nn.MaxPool2d(2),
            nn.Conv2d(16, 32, 3, padding=1),
            nn.ReLU(inplace=True),
            nn.MaxPool2d(2),
            nn.Conv2d(32, 64, 3, padding=1),
            nn.ReLU(inplace=True),
            nn.AdaptiveAvgPool2d(1),
        )
        self.fc = nn.Sequential(nn.Linear(64, 32), nn.ReLU(inplace=True), nn.Linear(32, 1), nn.Sigmoid())

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.fc(self.net(x).flatten(1)).squeeze(1)


class EmbeddingRegressor(nn.Module):
    def __init__(self):
        super().__init__()
        self.backbone = models.resnet18(weights=None)
        self.backbone.fc = nn.Identity()
        self.head = nn.Linear(512, 1)
        self.act = nn.Sigmoid()

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.act(self.head(self.backbone(x))).squeeze(1)


def build_accident(name: str) -> nn.Module:
    if name == "baseline_cnn":
        return SmallCNN(2)
    if name == "resnet18":
        m = models.resnet18(weights=None)
        m.fc = nn.Linear(m.fc.in_features, 2)
        return m
    raise ValueError(name)


def build_congestion(name: str) -> nn.Module:
    if name == "tiny_cnn":
        return TinyRegCNN()
    if name == "linear_resnet":
        return EmbeddingRegressor()
    raise ValueError(name)


def load_checkpoint(path: Path, track: str) -> tuple[nn.Module, int, dict]:
    ckpt = torch.load(path, map_location="cpu", weights_only=False)
    model_name = str(ckpt["model"])
    img_size = int(ckpt["img"])
    model = build_accident(model_name) if track == "accident" else build_congestion(model_name)
    model.load_state_dict(ckpt["state_dict"])
    model.eval()
    meta = {k: v for k, v in ckpt.items() if k != "state_dict"}
    return model, img_size, meta


def load_checkpoint_auto(path: Path) -> tuple[nn.Module, int, dict]:
    p = str(path.resolve())
    track = "accident" if "accident" in p else "congestion"
    return load_checkpoint(path, track)


def make_transform(img_size: int):
    return transforms.Compose(
        [
            transforms.Resize((img_size, img_size)),
            transforms.ToTensor(),
            transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),
        ]
    )


class CongestionCSV(Dataset):
    def __init__(self, root: Path, split: str, tf):
        self.root = root
        self.tf = tf
        self.rows: list[tuple[str, float]] = []
        with (root / "labels" / f"{split}.csv").open(encoding="utf-8") as f:
            r = csv.DictReader(f)
            for row in r:
                self.rows.append((row["path"], float(row["congestion"])))

    def __len__(self) -> int:
        return len(self.rows)

    def __getitem__(self, idx: int):
        rel, y = self.rows[idx]
        img = Image.open(self.root / rel).convert("RGB")
        return self.tf(img), torch.tensor(y, dtype=torch.float32)


def summarize_latency(samples_ms: list[float]) -> dict:
    samples_ms = sorted(samples_ms)
    if not samples_ms:
        return {"mean_ms": 0.0, "p50_ms": 0.0, "p95_ms": 0.0, "fps": 0.0}
    p50 = samples_ms[len(samples_ms) // 2]
    p95 = samples_ms[int(0.95 * (len(samples_ms) - 1))]
    mean_ms = statistics.mean(samples_ms)
    fps = 1000.0 / mean_ms if mean_ms > 0 else 0.0
    return {"mean_ms": mean_ms, "p50_ms": p50, "p95_ms": p95, "fps": fps}


@torch.no_grad()
def eval_congestion(model: nn.Module, loader: DataLoader) -> dict:
    ys: list[float] = []
    ps: list[float] = []
    lats: list[float] = []
    for x, y in loader:
        t0 = time.perf_counter()
        pred = model(x).cpu().numpy().tolist()
        lats.append((time.perf_counter() - t0) * 1000.0)
        ys.extend(y.numpy().tolist())
        ps.extend(pred)
    out = {
        "mae": float(mean_absolute_error(ys, ps)),
        "rmse": float(root_mean_squared_error(ys, ps)),
        "samples": len(ys),
    }
    out.update(summarize_latency(lats))
    return out


def evaluate_congestion_models(data_dir: Path, artifacts_dir: Path, batch_size: int) -> dict:
    out: dict[str, dict] = {}
    for model_name in ("tiny_cnn", "linear_resnet"):
        ckpt_path = artifacts_dir / model_name / "best.pt"
        if not ckpt_path.is_file():
            raise SystemExit(f"missing checkpoint: {ckpt_path}")
        model, img_size, _ = load_checkpoint(ckpt_path, "congestion")
        tf = make_transform(img_size)
        test_ds = CongestionCSV(data_dir, "test", tf)
        loader = DataLoader(test_ds, batch_size=batch_size)
        metrics = eval_congestion(model, loader)
        metrics["checkpoint"] = str(ckpt_path)
        out[model_name] = metrics
    return out
