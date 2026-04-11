#!/usr/bin/env python3
"""Create labeled eval sets from extracted video frames (symlinks + CSV).

Default labels match "no incident" video: all accident class = normal, congestion target = 0.
"""

from __future__ import annotations

import argparse
import csv
import os
import shutil
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument(
        "--data-dir",
        type=Path,
        default=None,
        help="Корень данных (как у rebuild_data_from_videos): frames в data-dir/videos/frames, GT — в data-dir/accident|congestion/...",
    )
    p.add_argument("--frames-dir", type=Path, default=None)
    p.add_argument(
        "--accident-images-root",
        type=Path,
        default=None,
        help="Will contain test/normal/*.png symlinks",
    )
    p.add_argument(
        "--congestion-root",
        type=Path,
        default=None,
        help="images/test/*.png symlinks + labels/test.csv",
    )
    p.add_argument(
        "--congestion-target",
        type=float,
        default=0.0,
        help="Regression target for every frame (0 = no congestion proxy).",
    )
    args = p.parse_args()

    data_dir = args.data_dir
    if data_dir is not None:
        data_dir = data_dir.resolve()
        if args.frames_dir is None:
            args.frames_dir = data_dir / "videos" / "frames"
        if args.accident_images_root is None:
            args.accident_images_root = data_dir / "accident" / "video-gt" / "images"
        if args.congestion_root is None:
            args.congestion_root = data_dir / "congestion" / "video-gt"
    if args.frames_dir is None:
        args.frames_dir = ROOT / "data" / "videos" / "frames"
    if args.accident_images_root is None:
        args.accident_images_root = ROOT / "data" / "accident" / "video-gt" / "images"
    if args.congestion_root is None:
        args.congestion_root = ROOT / "data" / "congestion" / "video-gt"

    frames = sorted(args.frames_dir.rglob("*.png"))
    if not frames:
        raise SystemExit(f"no frames under {args.frames_dir}")

    normal_dir = args.accident_images_root / "test" / "normal"
    cong_img_dir = args.congestion_root / "images" / "test"
    labels_dir = args.congestion_root / "labels"

    if args.accident_images_root.exists():
        shutil.rmtree(args.accident_images_root)
    if args.congestion_root.exists():
        shutil.rmtree(args.congestion_root)

    normal_dir.mkdir(parents=True, exist_ok=True)
    cong_img_dir.mkdir(parents=True, exist_ok=True)
    labels_dir.mkdir(parents=True, exist_ok=True)

    rows: list[tuple[str, float]] = []
    for src in frames:
        clip = src.parent.name
        name = f"{clip}_{src.name}"
        acc_link = normal_dir / name
        cong_link = cong_img_dir / name

        rel_to_acc = os.path.relpath(src.resolve(), acc_link.parent.resolve())
        rel_to_cong = os.path.relpath(src.resolve(), cong_link.parent.resolve())
        os.symlink(rel_to_acc, acc_link)
        os.symlink(rel_to_cong, cong_link)

        rel_csv_path = f"images/test/{name}".replace("\\", "/")
        rows.append((rel_csv_path, args.congestion_target))

    test_csv = labels_dir / "test.csv"
    with test_csv.open("w", newline="", encoding="utf-8") as f:
        w = csv.DictWriter(f, fieldnames=["path", "congestion"])
        w.writeheader()
        for path, score in rows:
            w.writerow({"path": path, "congestion": score})

    readme = args.congestion_root.parent / "video-gt_README.txt"
    readme.write_text(
        "Ground truth for your driving videos (no real incidents in footage).\n"
        "Каталог data/train не создаётся и не используется.\n"
        "- accident (legacy video-gt): только test/normal/ (класс normal).\n"
        "  Для валидации/теста по классам используйте data/CCTV-accidents/val и data/accident/test отдельно.\n"
        f"- congestion: все targets = {args.congestion_target} в labels/test.csv (тестовый split).\n"
        f"- frames linked: {len(frames)}\n",
        encoding="utf-8",
    )
    print(f"wrote accident GT: {normal_dir} ({len(frames)} symlinks)")
    print(f"wrote congestion GT: {test_csv}")


if __name__ == "__main__":
    main()
