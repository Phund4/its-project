# analytics

Сервис принимает `POST /v1/ingest` (от **`ml_gateway`** с блоком ML и/или от **`data_ingestion`** с полем **telemetry**), обновляет **метрики Prometheus** и пишет в **ClickHouse** только по **ML**-событиям: загруженность (`road_congestion`), инциденты (`road_incidents`).

Локального списка событий нет — только gauge/counter в Prometheus и OLAP в ClickHouse.

## Запуск

Требуется доступный **ClickHouse** на `CLICKHOUSE_ADDR` (по умолчанию `127.0.0.1:9000`). Удобно поднять через [infra/docker-compose.yml](../../infra/docker-compose.yml); сам **analytics** к Docker не привязан.

```bash
cd services/analytics
go run ./cmd/analytics
```

Настройки в **`.env`** (в git не коммитится). Переопределение файла: **`ENV_FILE`**.

| Переменная | Назначение |
|------------|------------|
| `LISTEN_ADDR` | HTTP, по умолчанию `:8093` |
| `CLICKHOUSE_ADDR` | `host:9000` native, по умолчанию `127.0.0.1:9000` |
| `CLICKHOUSE_DATABASE` | по умолчанию `default` |
| `CLICKHOUSE_USER` / `CLICKHOUSE_PASSWORD` | учётка CH |
| `CLICKHOUSE_INCIDENTS_TABLE` | по умолчанию `road_incidents` |
| `CLICKHOUSE_CONGESTION_TABLE` | по умолчанию `road_congestion` |
| `CRASH_ALERT_THRESHOLD` | порог для `crash_probability` (плюс label `crash`) |
| `CONGESTION_PERSIST_INTERVAL_SEC` | минимальный интервал (сек.) между строками в таблице загруженности **на одну камеру** (по умолчанию `2`; согласуйте с `CONGESTION_INTERVAL_SEC` в ML) |
| `MAP_GRPC_LISTEN_ADDR` | gRPC **map.v1.MapPortal** для **map_portal** (по умолчанию `:8097`) |
| `INFRA_SIM_DATABASE` | БД со справочниками для карты (по умолчанию `its_infra_sim`: `municipalities`, `bus_stops`) |
| `MUNICIPALITY_ACTIVITY_TTL_SEC` | окно активности города для приёма телеметрии в память карты (по умолчанию `45`) |

При приёме **telemetry** на ingest позиции ТС обновляют in-memory хаб для **map_portal** (нужен непустой `municipality_id` в JSON); отдельного HTTP в map_portal больше нет.

## API

- **gRPC** `map.v1.MapPortal` на `MAP_GRPC_LISTEN_ADDR` — `ListMunicipalities`, `ListStops`, `ListBuses` (использует **map_portal**).
- `POST /v1/ingest` — JSON как у бывшего `ml_gateway` road-events (`segment_id`, `camera_id`, `observed_at`, `s3_key`, `ml`).
- `GET /metrics` — Prometheus.
- `GET /health` — проверка процесса (не проверяет CH).

## Запись в ClickHouse

- **`road_congestion`**: не чаще чем раз в `CONGESTION_PERSIST_INTERVAL_SEC` **на пару** `(segment_id, camera_id)` — те же поля; метрика `analytics_road_congestion_score` по-прежнему обновляется на каждый ingest.
- **`road_incidents`** (или `CLICKHOUSE_INCIDENTS_TABLE`): только если **`incident.label` == `crash`** (без учёта регистра) **или** `incident.crash_probability >= CRASH_ALERT_THRESHOLD` — поля `crash_probability`, `incident_label`, `raw_ml` и т.д.

При обновлении с одной объединённой таблицы пересоздайте схему или выполните миграцию вручную: `CREATE TABLE` выполняется только для несуществующих таблиц.
