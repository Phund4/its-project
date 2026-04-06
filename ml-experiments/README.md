# ML experiments and model selection

`ml-experiments` now contains only offline experiments: benchmark runs, model comparison, and winner selection.

## Purpose

- Run experiments on your videos.
- Pick best checkpoints for accident and congestion tracks.
- Save outputs to repository root `.data` so runtime can consume them.

## Quick run

Prerequisites:

- checkpoints under `.data/ml-experiments/artifacts/`:
  - `.data/ml-experiments/artifacts/accident/{baseline-cnn,resnet18}/best.pt`
  - `.data/ml-experiments/artifacts/congestion/{tiny-cnn,linear-resnet}/best.pt`
- one or more `*.mp4` in `.data/videos/`

```bash
cd ml-experiments
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

bash scripts/run_pipeline.sh
```

Pipeline writes to `.data/ml-experiments/`:

- `benchmark/video_gt_results.json`
- `benchmark/video_gt_report.md`
- `winners.json` (selected best checkpoints, пути **относительно корня репозитория** — их подхватывает `services/ml-serving`, если не заданы `ACCIDENT_CKPT` / `CONGESTION_CKPT`)

## Runtime service moved

Model serving was moved to `services/ml-serving`.

Run service:

```bash
cd services/ml-serving
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn api.main:app --host 0.0.0.0 --port 8000 --no-access-log
```

