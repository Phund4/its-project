# data_ingestion (видео-поток → S3 → ML)

Сервис **только для видео-контура**: читает **поток по URL** (`ffmpeg`, типично **RTSP** с MediaMTX), сэмплирует кадры (`ingest.target_fps`), перекодирует в **PNG**, заливает в **MinIO/S3** и вызывает ML API `POST /v1/process`. При **`ML_GATEWAY_URL`** у `ml_experiments` ответ обычно **204**; события уходят в **`ml_gateway`** → **`analytics`**.

**Телеметрия CAN/GPS и т.п.** этим сервисом не обрабатывается и для её разработки **запускать видео-поток, MinIO, этот сервис и RTSP не требуется** — заводите отдельный конвейер (например Kafka → свой consumer → хранилище). См. [корневой README](../../README.md).

В **`config.example.yaml`** — четыре камеры, **`target_fps: 5`**, **`rtsp://localhost:8554/cam-01` …**. Потоки: **`infra`** профиль **`ingest`** (`video_source_sim`). Локальный файл в `rtsp_url` допускается для отладки.

Ключи S3: `{prefix}/{YYYY-MM-DD}/{camera_id}/frame_{unixnano}.png`

## Требования

- Go **1.22+**, **`ffmpeg`** в `PATH`
- **`AWS_ACCESS_KEY_ID`** / **`AWS_SECRET_ACCESS_KEY`** (MinIO: `minioadmin` / `minioadmin`)
- Опционально **`.env`** (`ENV_FILE`)

## Конфигурация

```bash
cp config.example.yaml config.yaml
```

| Переменная | Назначение |
|------------|------------|
| `ML_BASE_URL` | `ml.base_url` |
| `ML_PROCESS_PATH` | `ml.process_path` |
| `S3_ENDPOINT` | `s3.endpoint` |
| `METRICS_LISTEN_ADDR` | HTTP ` /metrics` (по умолчанию `:9091`) |

## Метрики

- **`data_ingestion_operation_errors_total{stage}`** — `ffmpeg_start`, `frame_read`, `s3_put`, `ml_process`

Вывод **ffmpeg** в консоль отключён; пока RTSP недоступен, в лог не чаще чем раз в **~45 с на камеру** пишется краткое предупреждение. Ошибки **S3** и вызова **ML** логируются как раньше.

Scrape: `host.docker.internal:9091` (см. [infra/README.md](../../infra/README.md)).

## Запуск

```bash
cd services/data_ingestion
cp config.example.yaml config.yaml
go run ./cmd/ingest
```

Дальше по цепочке: MinIO → **`ml_experiments`** → **`ml_gateway`** при необходимости (см. [ml_gateway](../ml_gateway/README.md)).

## Остановка

`Ctrl+C` — `SIGINT`/`SIGTERM`, завершение воркеров и metrics-сервера с graceful shutdown.
