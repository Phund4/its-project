# data_ingestion (видео-поток → S3 → ML)

Сервис для **видео-контура** и/или **gRPC-телеметрии автобуса**. Видео: читает **поток по URL** (`ffmpeg`, типично **RTSP** с MediaMTX), сэмплирует кадры (`ingest.target_fps`), перекодирует в **PNG**, заливает в **MinIO/S3** и вызывает ML API `POST /v1/process`. При **`ML_GATEWAY_URL`** у `ml_experiments` ответ обычно **204**; события уходят в **`ml_gateway`** → **`analytics`**.

**Телеметрия (без ML):** `TELEMETRY_GRPC_ENABLED=true`, `CAMERAS_ENABLED=false`, `ANALYTICS_INGEST_URL=http://…:8093/v1/ingest`, опционально `TELEMETRY_GRPC_LISTEN_ADDR` (по умолчанию `:50051`). Unary RPC `bus.v1.BusTelemetryService/SendBusTelemetry` маппится в JSON с полем **`telemetry`** (не `ml`) и пересылается в analytics. В сообщении protobuf есть **`municipality_id`** (код города: `msk`, `spb`, `kzn`, `ekb`) — он попадает в JSON; **analytics** кладёт координаты в память для отдачи карте по gRPC. Конфиг: [`config.telemetry.yaml`](config.telemetry.yaml). Если **`CONFIG_PATH`** не задан, по умолчанию подключается **`config.telemetry.yaml`**.

Генератор [`data_generators/bus_telemetry`](../../data_generators/bus_telemetry): переменная **`BUS_TELEMETRY_MUNICIPALITY_ID`** (по умолчанию `msk`) и координаты вокруг центра выбранного города.

**Камеры:** сегменты, `camera_id` и **`rtsp_url`** задаются **только** в [`config.cameras.yaml`](config.cameras.yaml). Для видео: `CONFIG_PATH=./config.cameras.yaml` и `CAMERAS_ENABLED=true`. Если процесс **ingestion** крутится в той же Docker-сети, что и MediaMTX, в `rtsp_url` укажите хост **`mediamtx`** вместо `localhost` (например `rtsp://mediamtx:8554/cam-01`).

**Телеметрия CAN/GPS и т.п.** этим сервисом не обрабатывается и для её разработки **запускать видео-поток, MinIO, этот сервис и RTSP не требуется** — заводите отдельный конвейер (например Kafka → свой consumer → хранилище). См. [корневой README](../../README.md).

Потоки для примера в `config.cameras.yaml`: **`infra`** профиль **`ingest`** (`video_source_sim`). Локальный файл в `rtsp_url` допускается для отладки.

Ключи S3: `{prefix}/{YYYY-MM-DD}/{camera_id}/frame_{unixnano}.png`

## Требования

- Go **1.22+**
- **Видео:** **`ffmpeg`** в `PATH`, **`AWS_ACCESS_KEY_ID`** / **`AWS_SECRET_ACCESS_KEY`** (MinIO: `minioadmin` / `minioadmin`)
- Опционально **`.env`** (`ENV_FILE`). Шаблон телеметрии: [`telemetry.env.example`](telemetry.env.example)

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
| `CAMERAS_ENABLED` / `TELEMETRY_GRPC_ENABLED` | режимы |
| `S3_ENABLED` / `ML_ENABLED` | при `CAMERAS_ENABLED=true` должны быть `true` |
| `ANALYTICS_INGEST_URL` | полный URL `POST /v1/ingest` при телеметрии |
| `TELEMETRY_GRPC_LISTEN_ADDR` | gRPC (по умолчанию `:50051`) |

Опционально генератор в Docker ([`infra`](../../infra/README.md), профиль **`telemetry`**) шлёт на **`host.docker.internal:50051`**; при запуске генератора с хоста достаточно `127.0.0.1:50051`.

## Метрики

- **`data_ingestion_operation_errors_total{stage}`** — `ffmpeg_start`, `frame_read`, `s3_put`, `ml_process`, `telemetry_forward_analytics`
- **`data_ingestion_bus_telemetry_forwarded_total`** — успешная пересылка unary RPC в analytics

Вывод **ffmpeg** в консоль отключён; пока RTSP недоступен, в лог не чаще чем раз в **~45 с на камеру** пишется краткое предупреждение (только если включён контур камер).

Метрики на `http://127.0.0.1:9091/metrics` при локальном запуске. **Prometheus в compose** скрейпит хост через `host.docker.internal:9091` (см. [prometheus.yml](../../infra/prometheus/prometheus.yml)).

## Запуск

**Телеметрия:** задайте `.env` (см. `telemetry.env.example`), затем `go run ./cmd/ingest`.

**Видео:** `CONFIG_PATH=./config.cameras.yaml`, `CAMERAS_ENABLED=true`, ключи S3 в `.env`, MinIO и RTSP (MediaMTX), затем `go run ./cmd/ingest`.

Дальше по цепочке: MinIO → **`ml_experiments`** → **`ml_gateway`** при необходимости (см. [ml_gateway](../ml_gateway/README.md)).

## Остановка

`Ctrl+C` — `SIGINT`/`SIGTERM`, завершение воркеров и metrics-сервера с graceful shutdown.
