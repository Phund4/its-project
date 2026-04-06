# coordinator

Лёгкий сервис-координатор источников данных для `data-ingestion`.

Сейчас покрывает:
- каталог источников (camera/telemetry) по зонам;
- выдачу назначений (`assignments`) для конкретного ingestion-инстанса;
- heartbeat от ingestion-инстансов;
- primary/reserve владельца источника с авто-переключением по timeout heartbeat.

## Конфигурация

- `LISTEN_ADDR` — адрес HTTP (по умолчанию `:8098`)
- `SOURCES_CONFIG_PATH` — путь к YAML с источниками (по умолчанию `sources.yaml`)
- `HEARTBEAT_TIMEOUT_SEC` — сколько секунд heartbeat считается свежим (по умолчанию `30`)

## API

- `GET /health`
- `GET /v1/sources?zone_id=zone-a`
- `GET /v1/assignments?zone_id=zone-a&cluster_id=cluster-1&instance_id=ingest-a1&source_kind=camera`
- `GET /v1/assignments?zone_id=zone-a&cluster_id=cluster-1&instance_id=ingest-a1&source_kind=telemetry`
- `POST /v1/workers/heartbeat`
- `GET /v1/workers`

## Режимы назначения

- **Шаг 1 (статический):** `cluster_id/instance_id` в `sources.yaml` закрепляют источник за конкретным `data-ingestion`.
- **Шаг 2 (failover):** задайте `reserve_cluster_id/reserve_instance_id`.
  - пока heartbeat primary свежий — источник у primary;
  - если heartbeat primary истёк и reserve жив — источник автоматически у reserve.

## Запуск

```bash
cd services/coordinator
go run ./cmd/coordinator
```
