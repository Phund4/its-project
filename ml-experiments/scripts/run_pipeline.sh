#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
PY="${PY:-$ROOT/.venv/bin/python}"
if [[ ! -x "$PY" ]]; then
  echo "Create venv first: python3 -m venv .venv && .venv/bin/pip install -r requirements.txt" >&2
  exit 1
fi

VID="${ROOT}/../.data/videos"
if [[ ! -d "${VID}" ]]; then
  echo "Create ${VID} and add .mp4 files" >&2
  exit 1
fi
if ! compgen -G "${VID}/*.mp4" > /dev/null; then
  echo "Add at least one .mp4 under ${VID}" >&2
  exit 1
fi

OUT_ROOT="${ROOT}/../.data/ml-experiments"
DATA_DIR="${OUT_ROOT}/data"
BENCH_DIR="${OUT_ROOT}/benchmark"
WINNERS_JSON="${OUT_ROOT}/winners.json"

"$PY" scripts/rebuild_data_from_videos.py --videos-dir "${VID}" --data-dir "${DATA_DIR}" --fps 3
"$PY" scripts/prepare_video_ground_truth.py --data-dir "${DATA_DIR}"
# Валидация accident: DATA_DIR/CCTV-accidents/val/{Accident,Non Accident}/…
# Тест accident: DATA_DIR/accident/test/{Accident,Non Accident}/…
# Тест congestion: prepare_video_ground_truth → congestion/video-gt/labels/test.csv
"$PY" scripts/evaluate_video_gt.py \
  --cctv-accidents-root "${DATA_DIR}/CCTV-accidents" \
  --accident-test-root "${DATA_DIR}/accident/test" \
  --accident-data "${DATA_DIR}/accident/video-gt/images" \
  --congestion-data "${DATA_DIR}/congestion/video-gt" \
  --output "${BENCH_DIR}/video_gt_results.json" \
  --winners-output "${WINNERS_JSON}"
MPL="${PYTHONMPLCONFIGDIR:-${ROOT}/.mplconfig}"
mkdir -p "${MPL}"
PYTHONMPLCONFIGDIR="${MPL}" "$PY" scripts/report_video_gt.py \
  --input "${BENCH_DIR}/video_gt_results.json" \
  --out-dir "${BENCH_DIR}"

echo "Report: ${BENCH_DIR}/video_gt_report.md"
echo "Winners: ${WINNERS_JSON}"
echo "Runtime service: cd services/ml-serving && uvicorn api.main:app --host 0.0.0.0 --port 8000 --no-access-log"
