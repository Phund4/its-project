#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

echo "[1/3] Stop compose and remove orphans..."
docker compose down --remove-orphans

echo "[2/3] Remove Kafka/ZooKeeper volumes..."
docker volume rm traffic-infra_kafka-data traffic-infra_zookeeper-data || true

echo "[3/3] Start infra again..."
docker compose up -d

echo "Done. Kafka and ZooKeeper are recreated from clean state."
