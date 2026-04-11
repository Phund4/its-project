#!/usr/bin/env python3
"""Report/charts for video_gt_results.json."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

import matplotlib.pyplot as plt


def bar(names: list[str], vals: list[float], title: str, ylabel: str, out: Path) -> None:
    fig = plt.figure(figsize=(7, 4))
    ax = fig.add_subplot(111)
    ax.bar(names, vals)
    ax.set_title(title)
    ax.set_ylabel(ylabel)
    fig.tight_layout()
    fig.savefig(out, dpi=150)
    plt.close(fig)


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--input", type=Path, required=True)
    p.add_argument("--out-dir", type=Path, default=Path("benchmark"))
    args = p.parse_args()

    d = json.loads(args.input.read_text(encoding="utf-8"))
    args.out_dir.mkdir(parents=True, exist_ok=True)

    acc_track = d["tracks"]["accident"]
    if isinstance(acc_track.get("test"), dict) and "models" in acc_track["test"]:
        acc = acc_track["test"]["models"]
        acc_val = acc_track.get("validation", {}).get("models") if isinstance(acc_track.get("validation"), dict) else None
    else:
        acc = acc_track["models"]
        acc_val = None
    cong = d["tracks"]["congestion"]["models"]
    acc_n = list(acc.keys())
    cong_n = list(cong.keys())

    has_binary_metrics = "precision_crash" in next(iter(acc.values()))
    if not has_binary_metrics:
        bar(
            acc_n,
            [acc[k]["false_crash_rate"] for k in acc_n],
            "Accident: false crash rate (GT = all normal)",
            "rate",
            args.out_dir / "video_gt_false_crash.png",
        )
    else:
        bar(
            acc_n,
            [acc[k]["f1_crash"] for k in acc_n],
            "Accident: F1 (crash class)",
            "F1",
            args.out_dir / "video_gt_f1_crash.png",
        )
    bar(
        acc_n,
        [acc[k]["mean_crash_probability"] for k in acc_n],
        "Accident: mean P(crash) (ideal ~0 on normal video)",
        "probability",
        args.out_dir / "video_gt_mean_crash_prob.png",
    )
    bar(
        acc_n,
        [acc[k]["f1_normal"] for k in acc_n],
        "Accident: F1 for normal class",
        "F1",
        args.out_dir / "video_gt_f1_normal.png",
    )
    bar(acc_n, [acc[k]["fps"] for k in acc_n], "Accident: FPS", "FPS", args.out_dir / "video_gt_accident_fps.png")
    bar(cong_n, [cong[k]["mae"] for k in cong_n], "Congestion: MAE vs 0", "MAE", args.out_dir / "video_gt_cong_mae.png")
    bar(cong_n, [cong[k]["fps"] for k in cong_n], "Congestion: FPS", "FPS", args.out_dir / "video_gt_cong_fps.png")

    lines = [
        "# Video ground-truth benchmark",
        "",
        "## How to read this",
        "",
        d.get("interpretation", ""),
        "",
        "Assumptions:",
        f"- accident: {d['assumptions']['accident']}",
        f"- congestion: {d['assumptions']['congestion']}",
        "",
        "## Winners",
        f"- accident: `{d['tracks']['accident']['winner']}`",
        f"- congestion: `{d['tracks']['congestion']['winner']}`",
        "",
        "## Accident (test split)",
        "",
        d.get("interpretation", ""),
        "",
    ]
    if has_binary_metrics:
        lines.extend(
            [
                "| model | accuracy | precision_crash | recall_crash | f1_crash | f1_normal | mean_P_crash | p95_P_crash | fps |",
                "|---|---:|---:|---:|---:|---:|---:|---:|---:|",
            ]
        )
        for k in acc_n:
            m = acc[k]
            lines.append(
                f"| {k} | {m['accuracy']:.4f} | {m['precision_crash']:.4f} | {m['recall_crash']:.4f} | "
                f"{m['f1_crash']:.4f} | {m['f1_normal']:.4f} | {m['mean_crash_probability']:.4f} | "
                f"{m['p95_crash_probability']:.4f} | {m['fps']:.2f} |"
            )
    else:
        lines.extend(
            [
                "При двух классах: **accuracy = 1 − false_crash_rate** (доля кадров, где предсказан класс normal).",
                "",
                "| model | accuracy | f1_normal | false_crash_rate | mean_P_crash | p95_P_crash | fps |",
                "|---|---:|---:|---:|---:|---:|---:|",
            ]
        )
        for k in acc_n:
            m = acc[k]
            lines.append(
                f"| {k} | {m['accuracy']:.4f} | {m['f1_normal']:.4f} | {m['false_crash_rate']:.4f} | "
                f"{m['mean_crash_probability']:.4f} | {m['p95_crash_probability']:.4f} | {m['fps']:.2f} |"
            )
        lines.extend(["", "_f1_crash_omit: on all-normal GT, F1 for the crash class is not meaningful."])

    if acc_val:
        lines.extend(
            [
                "",
                "## Accident (validation / CCTV-accidents/val)",
                "",
                "| model | accuracy | precision_crash | recall_crash | f1_crash | f1_normal | fps |",
                "|---|---:|---:|---:|---:|---:|---:|",
            ]
        )
        for k in acc_n:
            m = acc_val[k]
            lines.append(
                f"| {k} | {m['accuracy']:.4f} | {m['precision_crash']:.4f} | {m['recall_crash']:.4f} | "
                f"{m['f1_crash']:.4f} | {m['f1_normal']:.4f} | {m['fps']:.2f} |"
            )

    lines.extend(
        [
            "",
            "## Congestion (test)",
            "",
            "| model | mae | rmse | fps |",
            "|---|---:|---:|---:|",
        ]
    )
    for k in cong_n:
        m = cong[k]
        lines.append(f"| {k} | {m['mae']:.4f} | {m['rmse']:.4f} | {m['fps']:.2f} |")
    lines.extend(
        [
            "",
            "## Charts",
            f"- `{'video_gt_f1_crash.png' if has_binary_metrics else 'video_gt_false_crash.png'}`",
            "- `video_gt_mean_crash_prob.png`",
            "- `video_gt_f1_normal.png`",
            "- `video_gt_accident_fps.png`",
            "- `video_gt_cong_mae.png`",
            "- `video_gt_cong_fps.png`",
        ]
    )

    out_md = args.out_dir / "video_gt_report.md"
    out_md.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"wrote {out_md}")


if __name__ == "__main__":
    main()
