#!/bin/sh
set -eu

STREAM_PREFIX="${STREAM_PREFIX:-cam-}"
NUM_STREAMS="${NUM_STREAMS:-4}"

BASE="${RTSP_PUBLISH_BASE:-rtsp://mediamtx:8554}"
VIDEO_DIR="${SIM_VIDEO_DIR:-/videos}"

sorted_mp4() {
  ls -1 "${VIDEO_DIR}"/*.mp4 2>/dev/null | sort
}

publish_stream() {
  NAME="$1"
  SRC_FILE="$2"
  while true; do
    echo "Starting publisher ${SRC_FILE} -> ${BASE}/${NAME}"
    ffmpeg -hide_banner -loglevel warning -re \
      -stream_loop -1 -i "${SRC_FILE}" \
      -an -c:v libx264 -pix_fmt yuv420p -preset veryfast -tune zerolatency \
      -f rtsp -rtsp_transport tcp "${BASE}/${NAME}" || true
    sleep 1
  done
}

VIDEOS="$(sorted_mp4)"
if [ -z "${VIDEOS}" ]; then
  COUNT=0
else
  COUNT="$(printf "%s\n" "${VIDEOS}" | sed "/^$/d" | wc -l | tr -d " ")"
fi

if [ "${COUNT}" = "0" ]; then
  echo "No .mp4 files found under ${VIDEO_DIR}; falling back to synthetic streams."
  FPS="${SIM_FPS:-25}"
  SIZE="${SIM_SIZE:-1280x720}"
  while true; do
    ffmpeg -hide_banner -loglevel warning -re -f lavfi -i "testsrc2=size=${SIZE}:rate=${FPS}" \
      -an -c:v libx264 -pix_fmt yuv420p -preset veryfast -tune zerolatency \
      -f rtsp -rtsp_transport tcp "${BASE}/cam-east-01" || true
    sleep 1
  done &
  while true; do
    ffmpeg -hide_banner -loglevel warning -re -f lavfi -i "smptebars=size=${SIZE}:rate=${FPS}" \
      -an -c:v libx264 -pix_fmt yuv420p -preset veryfast -tune zerolatency \
      -f rtsp -rtsp_transport tcp "${BASE}/cam-west-02" || true
    sleep 1
  done &
else
  k=1
  while [ "$k" -le "$NUM_STREAMS" ]; do
    idx=$(( (k - 1) % COUNT + 1 ))
    SRC="$(printf "%s\n" "${VIDEOS}" | sed -n "${idx}p")"
    NAME="${STREAM_PREFIX}$(printf "%02d" "$k")"
    publish_stream "$NAME" "$SRC" &
    k=$((k + 1))
  done
fi

wait
