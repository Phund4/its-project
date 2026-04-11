# Шлюз ml-gateway

Тонкий HTTP-слой между **`ml-serving`** (или другим клиентом) и **`analytics`**: принимает `POST /v1/road-events`, проверяет JSON и **синхронно пересылает** то же тело в **`analytics`** (`ANALYTICS_BASE_URL` + `ANALYTICS_INGEST_PATH`). Локального хранения нет, эндпоинта **`GET /v1/road-events`** нет.

При необходимости сервис можно не включать в `infra/docker-compose.yml` — запуск вручную.

## API

- `POST /v1/road-events` — тело JSON: `segment_id`, `camera_id`, `observed_at`, `s3_key`, `ml`. Успех — **`204`**. Если не заданы ни **`KAFKA_BOOTSTRAP_SERVERS`**, ни **`ANALYTICS_BASE_URL`** — **`503`**. Сбой пересылки (Kafka или HTTP) — **`502`**.
- `GET /metrics` — метрики Prometheus.
- `GET /health` — `200 ok`.

## Переменные окружения

Файл **`.env`** в каталоге сервиса (см. корневой `.gitignore`), либо **`ENV_FILE`**.

| Переменная | Назначение |
|------------|------------|
| `LISTEN_ADDR` | по умолчанию `:8092` |
| `ANALYTICS_BASE_URL` | базовый URL analytics, например `http://127.0.0.1:8093` |
| `KAFKA_BOOTSTRAP_SERVERS` | если задано — события пишутся в Kafka (топик из `KAFKA_TOPIC_VIDEO`) вместо HTTP |
| `KAFKA_TOPIC_VIDEO` | топик для видео-событий |
| `ANALYTICS_INGEST_PATH` | по умолчанию `/v1/ingest` |
| `ANALYTICS_TIMEOUT` | таймаут HTTP-клиента в секундах, по умолчанию `10` |

## Запуск

```bash
cd services/ml-gateway
go run ./cmd/gateway
```

Типичный порядок для видео: **ClickHouse** → **analytics** → **ml-gateway** ← **ml-serving** ← **data-ingestion**.

В **`services/ml-serving/.env`** задаётся **`ML_GATEWAY_URL`** на этот сервис (если нужна отправка событий в шлюз после инференса).

Метрика ошибок: **`ml_gateway_operation_errors_total{stage}`** — `post_decode` | `post_validate` | `forward_analytics` | `analytics_response` | `kafka_write` и др.
