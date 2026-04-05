#!/usr/bin/env bash
# Применяет схему и тестовые данные к локальному ClickHouse (из каталога infra: docker compose up -d clickhouse).
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_ROOT="$(cd "$DIR/.." && pwd)"

CH_HOST="${CH_HOST:-127.0.0.1}"
CH_PORT="${CH_PORT:-9000}"
CH_USER="${CH_USER:-default}"
CH_PASSWORD="${CH_PASSWORD:-}"

run_client() {
  local sql_file=$1
  if command -v clickhouse-client >/dev/null 2>&1; then
    local -a args=(--host "$CH_HOST" --port "$CH_PORT" --user "$CH_USER" --multiquery)
    if [[ -n "$CH_PASSWORD" ]]; then
      args+=(--password "$CH_PASSWORD")
    fi
    clickhouse-client "${args[@]}" < "$sql_file"
    return
  fi

  if [[ -f "$INFRA_ROOT/docker-compose.yml" ]] && docker info >/dev/null 2>&1; then
    echo "clickhouse-client не найден в PATH, используем контейнер compose (сервис clickhouse)…"
    (cd "$INFRA_ROOT" && docker compose exec -T clickhouse clickhouse-client \
      --user "$CH_USER" --multiquery) < "$sql_file"
    return
  fi

  echo "Не удалось выполнить SQL: установите clickhouse-client или запустите Docker с сервисом clickhouse в $INFRA_ROOT" >&2
  exit 1
}

echo "Applying $DIR/001_schema.sql …"
run_client "$DIR/001_schema.sql"
echo "Applying $DIR/002_seed.sql …"
run_client "$DIR/002_seed.sql"
echo "Done. Проверка: SELECT count() FROM its_infra_sim.municipalities; SELECT count() FROM its_infra_sim.bus_stops;"
