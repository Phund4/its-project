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

"$PY" scripts/rebuild_data_from_videos.py --videos-dir "${VID}" --data-dir "${ROOT}/data" --fps 3
"$PY" scripts/prepare_video_ground_truth.py --frames-dir "${ROOT}/data/videos/frames"
"$PY" scripts/evaluate_video_gt.py --output "${ROOT}/benchmark/video_gt_results.json"
MPL="${PYTHONMPLCONFIGDIR:-${ROOT}/.mplconfig}"
mkdir -p "${MPL}"
PYTHONMPLCONFIGDIR="${MPL}" "$PY" scripts/report_video_gt.py --input "${ROOT}/benchmark/video_gt_results.json" --out-dir "${ROOT}/benchmark"

echo "Report: ${ROOT}/benchmark/video_gt_report.md"
echo "API: uvicorn api.main:app --host 0.0.0.0 --port 8000 --no-access-log"
