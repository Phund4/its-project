#!/usr/bin/env python3
"""Rebuild ml_experiments/data from local videos (frames only, no labels)."""

from __future__ import annotations

import argparse
import shutil
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def extract(video: Path, out_dir: Path, fps: float) -> int:
    clip = video.stem
    dst = out_dir / clip
    dst.mkdir(parents=True, exist_ok=True)
    pattern = str(dst / "f_%06d.png")
    cmd = [
        "ffmpeg",
        "-hide_banner",
        "-loglevel",
        "error",
        "-i",
        str(video),
        "-vf",
        f"fps={fps}",
        "-y",
        pattern,
    ]
    subprocess.run(cmd, check=True)
    return len(list(dst.glob("*.png")))


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--videos-dir", type=Path, default=ROOT.parent / ".data" / "videos")
    p.add_argument("--data-dir", type=Path, default=ROOT / "data")
    p.add_argument("--fps", type=float, default=3.0)
    args = p.parse_args()

    videos = sorted(args.videos_dir.glob("*.mp4"))
    if not videos:
        raise SystemExit(f"no mp4 videos found in {args.videos_dir}")

    if args.data_dir.exists():
        shutil.rmtree(args.data_dir)
    frames_root = args.data_dir / "videos" / "frames"
    frames_root.mkdir(parents=True, exist_ok=True)

    summary: list[str] = []
    total = 0
    for video in videos:
        n = extract(video, frames_root, args.fps)
        total += n
        summary.append(f"{video.name}: {n} frames")

    (args.data_dir / "README.txt").write_text(
        "Data rebuilt from local videos only.\n"
        "No ground-truth labels were created automatically.\n"
        "Frames are in data/videos/frames/<clip>/*.png\n"
        + "\n".join(summary)
        + f"\nTotal frames: {total}\n",
        encoding="utf-8",
    )
    print(f"rebuilt {args.data_dir} from {len(videos)} videos, total frames={total}")


if __name__ == "__main__":
    main()
