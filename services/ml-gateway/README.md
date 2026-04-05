# ml_gateway

Тонкий HTTP-слой между **ml-experiments** и **analytics**: принимает `POST /v1/road-events`, валидирует JSON и **синхронно пересылает** то же тело в **`analytics`** (`ANALYTICS_BASE_URL` + `ANALYTICS_INGEST_PATH`). Локального хранения и **нет** `GET /v1/road-events`.

Сервис **не** входит в `infra/docker-compose.yml` — запуск вручную.

## API

- `POST /v1/road-events` — тело JSON: `segment_id`, `camera_id`, `observed_at`, `s3_key`, `ml`. При успешной пересылке в analytics ответ **`204`**. Если **`ANALYTICS_BASE_URL`** не задан — **`503`**. Ошибка analytics — **`502`**.
- `GET /metrics` — Prometheus.
- `GET /health` — `200 ok`.

## Переменные окружения

Файл **`.env`** в каталоге сервиса (см. корневой `.gitignore`), либо **`ENV_FILE`**.

| Переменная | Назначение |
|------------|------------|
| `LISTEN_ADDR` | по умолчанию `:8092` |
| `ANALYTICS_BASE_URL` | базой URL analytics, например `http://127.0.0.1:8093` |
| `ANALYTICS_INGEST_PATH` | по умолчанию `/v1/ingest` |
| `ANALYTICS_TIMEOUT` | секунды HTTP-клиента, по умолчанию `10` |

## Запуск

```bash
cd services/ml-gateway
go run ./cmd/gateway
```

Порядок: **ClickHouse** → **analytics** → **ml-gateway** → **ml-experiments** → **data-ingestion** (только для видео-потока).

В **`ml-experiments/.env`** по-прежнему **`ML_GATEWAY_URL`** на этот сервис.

Метрика ошибок: **`ml_gateway_operation_errors_total{stage}`** — `post_decode` | `post_validate` | `forward_analytics` | `analytics_response`.
