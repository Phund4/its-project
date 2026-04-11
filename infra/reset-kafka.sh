#!/usr/bin/env bash
# Резервный сброс только при повреждённом томе KRaft (обычно не нужен).
# Стек по умолчанию: KRaft без ZooKeeper, персистентность в traffic-infra_kafka-kraft-data.
set -euo pipefail

cd "$(dirname "$0")"

echo "[1/3] Stop compose..."
docker compose down --remove-orphans

echo "[2/3] Remove Kafka KRaft volume..."
docker volume rm traffic-infra_kafka-kraft-data || true

echo "[3/3] Start infra again..."
docker compose up -d

echo "Done. Kafka KRaft volume recreated."
