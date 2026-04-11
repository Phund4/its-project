# data_ingestion (видео-поток → S3 → ML)

Сервис для **видео-контура** и/или **gRPC-телеметрии автобуса**. Что именно запускать (видео, телеметрия или оба контура) определяет **coordinator** через назначения источников.

**Телеметрия (без ML):** два входа:
- gRPC unary `bus.v1.BusTelemetryService/SendBusTelemetry` (порт `TELEMETRY_GRPC_LISTEN_ADDR`, по умолчанию `:50051`);
- HTTP `POST /v1/telemetry` (порт `TELEMETRY_HTTP_LISTEN_ADDR`, по умолчанию `:8094`).

Оба входа маппят данные в JSON с полем **`telemetry`** (не `ml`) и пересылают в analytics/Kafka.  
Каталог в **coordinator** задаёт **`data_class`**: для `vehicle_bus_telemetry` фильтра по списку городов нет — районы и идентификаторы в полезной нагрузке.

Генератор [`data-generators/telemetry-data`](../../data-generators/telemetry-data): переменные **`BUS_TELEMETRY_MUNICIPALITY_ID`**, **`BUS_TELEMETRY_ALL_CITIES`** и координаты вдоль маршрутов из сида ClickHouse (см. код генератора).

**Камеры:** сегменты, `camera_id` и **`rtsp_url`** задаются назначениями в **coordinator** (`data_class=road_segment_video`). Если процесс **ingestion** крутится в той же Docker-сети, что и MediaMTX, в `rtsp_url` укажите хост **`mediamtx`** вместо `localhost` (например `rtsp://mediamtx:8554/cam-01`).

**Телеметрия CAN/GPS и т.п.** этим сервисом не обрабатывается и для её разработки **запускать видео-поток, MinIO, этот сервис и RTSP не требуется** — заводите отдельный конвейер (например Kafka → свой consumer → хранилище). См. [корневой README](../../README.md).

Потоки для примера в `config.cameras.yaml`: **`infra`** профиль **`ingest`** (`video-source-sim`). Локальный файл в `rtsp_url` допускается для отладки.

Ключи S3: `{prefix}/{YYYY-MM-DD}/{camera_id}/frame_{unixnano}.png`

## Требования

- Go **1.22+**
- **Видео:** **`ffmpeg`** в `PATH`, **`AWS_ACCESS_KEY_ID`** / **`AWS_SECRET_ACCESS_KEY`** (MinIO: `minioadmin` / `minioadmin`)
- Опционально **`.env`** (`ENV_FILE`).

## Конфигурация

| Файл | Назначение |
|------|------------|
| `config.telemetry.yaml` | Только gRPC-телеметрия, `cameras: []` |
| `config.cameras.yaml` | Видео: S3/ML/ingest/metrics и список камер |

| Переменная | Назначение |
|------------|------------|
| `CONFIG_PATH` | Путь к YAML (дефолт в коде: `config.telemetry.yaml`) |
| `ML_BASE_URL` | переопределение `ml.base_url` |
| `ML_PROCESS_PATH` | `ml.process_path` |
| `S3_ENDPOINT` | `s3.endpoint` |
| `METRICS_LISTEN_ADDR` | HTTP `/metrics` (по умолчанию `:9091`) |
| `ANALYTICS_INGEST_URL` | полный URL `POST /v1/ingest` при телеметрии |
| `TELEMETRY_GRPC_LISTEN_ADDR` | gRPC (по умолчанию `:50051`) |
| `TELEMETRY_HTTP_LISTEN_ADDR` | HTTP вход телеметрии (по умолчанию `:8094`) |
| `COORDINATOR_BASE_URL` | URL coordinator (`http://127.0.0.1:8098`) для назначений источников (обязательно в видео и telemetry gRPC режимах) |
| `COORDINATOR_ZONE_ID` | зона назначения (например `zone-a`) |
| `COORDINATOR_CLUSTER_ID` | кластер ingestion (например `cluster-1`) |
| `COORDINATOR_INSTANCE_ID` | инстанс ingestion (например `ingest-a1`) |

Опционально генератор в Docker ([`infra`](../../infra/README.md), профиль **`telemetry`**) шлёт на **`host.docker.internal:50051`**; при запуске генератора с хоста достаточно `127.0.0.1:50051`.

## Метрики

- **`data_ingestion_operation_errors_total{stage}`** — `ffmpeg_start`, `frame_read`, `s3_put`, `ml_process`, `telemetry_forward_analytics`
- **`data_ingestion_bus_telemetry_forwarded_total`** — успешная пересылка unary RPC в analytics

Вывод **ffmpeg** в консоль отключён; пока RTSP недоступен, в лог не чаще чем раз в **~45 с на камеру** пишется краткое предупреждение (только если включён контур камер).

Метрики на `http://127.0.0.1:9091/metrics` при локальном запуске. **Prometheus в compose** скрейпит хост через `host.docker.internal:9091` (см. [prometheus.yml](../../infra/prometheus/prometheus.yml)).

## Запуск

`data-ingestion` запускается единым процессом `go run ./cmd/ingest`.
Дальше coordinator назначает источники по полю **`data_class`** в `sources.yaml`:
- **`road_segment_video`** — включает video pipeline (RTSP → S3 → ML);
- **`vehicle_bus_telemetry`** — включает приём gRPC/HTTP телеметрии и форвард в analytics/Kafka.

Список камер: `GET /v1/assignments?...&data_class=road_segment_video`. Телеметрия ТС: `data_class=vehicle_bus_telemetry` (при назначении инстансу включается контур без фильтра по городам из YAML).

Нужны `COORDINATOR_BASE_URL` + `COORDINATOR_ZONE_ID` + `COORDINATOR_CLUSTER_ID` + `COORDINATOR_INSTANCE_ID`. При старте сервис может работать в standby (без активных назначений) и ждать команд от coordinator.

Дальше по цепочке: MinIO → **`ml-experiments`** → **`ml-gateway`** при необходимости (см. [ml-gateway](../ml-gateway/README.md)).

## Остановка

`Ctrl+C` — `SIGINT`/`SIGTERM`, завершение воркеров и metrics-сервера с graceful shutdown.
