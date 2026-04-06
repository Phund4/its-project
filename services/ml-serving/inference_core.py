"""PyTorch model builders and checkpoint loading for ml-serving."""

from __future__ import annotations

from pathlib import Path

import torch
import torch.nn as nn
from torchvision import models, transforms


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
