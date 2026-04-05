# Локальная инфраструктура

Сеть Docker: **`traffic-its`**. Имя проекта Compose: **`traffic-infra`**.

## Запуск и остановка

```bash
cd infra
docker compose pull
docker compose up -d
```

```bash
docker compose down
```

Данные **ClickHouse**, **Kafka**, **Zookeeper**, **Elasticsearch**, **MinIO**, **Prometheus** (TSDB) и **Grafana** (в том числе дашборды и настройки, созданные в UI) хранятся в именованных томах и переживают перезапуск контейнеров. Конфиг Prometheus по-прежнему файл [`prometheus/prometheus.yml`](prometheus/prometheus.yml); после правок: `docker compose restart prometheus` или lifecycle reload.

## Сервисы и подключение

- **Zookeeper** — внутри сети: `zookeeper:2181`.

- **Kafka** — с хоста: `localhost:9092`; внутри сети: `kafka:29092`. Пример: `export KAFKA_BOOTSTRAP_SERVERS=localhost:9092`.

- **Kafka UI** — http://localhost:8080

- **Elasticsearch** — http://localhost:9200 (без логина, dev). Внутри сети: `elasticsearch:9200`.

- **Kibana** — http://localhost:5601

- **ClickHouse** — HTTP с хоста: `http://localhost:8123`; нативный протокол: `localhost:9000`. Внутри сети: `clickhouse:8123`, `clickhouse:9000`. Пользователь **`default`**, пароль пустой (только dev).

  **JDBC** (драйвер `com.clickhouse.jdbc.ClickHouseDriver`, интерфейс HTTP на порту 8123):
  - с хоста: `jdbc:clickhouse://localhost:8123/default`
  - из контейнера в сети `traffic-its`: `jdbc:clickhouse://clickhouse:8123/default`

  Пример `clickhouse-client` с хоста: `clickhouse-client --host localhost --port 9000`.

- **MinIO (S3)** — API с хоста: `http://localhost:9050`; консоль: http://localhost:9051. Учётные данные: **`minioadmin` / `minioadmin`**. Внутри сети: endpoint `http://minio:9000`.

- **Prometheus** — http://localhost:9090. Скрейп с хоста (если процессы запущены): **`data_ingestion`** `:9091`, **`ml_gateway`** `:8092`, **`analytics`** `:8093` (через `host.docker.internal`; на Linux у Prometheus задан `extra_hosts`). Примеры метрик: `data_ingestion_operation_errors_total`, `ml_gateway_operation_errors_total`, `analytics_road_*`, `analytics_clickhouse_errors_total`, `analytics_ingest_errors_total`. Для разработки **телеметрии** метрики `:9091` не обязательны — поднимайте только нужные таргеты и свой сервис с `/metrics`.

- **Grafana** — http://localhost:3000, логин по умолчанию **`admin` / `admin`**. Провижининг из [`grafana/provisioning/`](grafana/provisioning/) (папка дашбордов **Traffic**); созданные вручную дашборды и источники сохраняются в томе **`grafana-data`**.

- **MediaMTX** (профиль `ingest`) — `rtsp://localhost:8554`. Запуск: `docker compose --profile ingest up -d --build mediamtx video-source-sim`.

- **video-source-sim** (профиль `ingest`) — читает `../.data/videos/*.mp4`; при отсутствии файлов — синтетические потоки.

## Приложения вне compose

Вручную (см. `.env` в каталогах сервисов): **analytics** ([`services/analytics`](../services/analytics/README.md)), **ml_gateway** ([`services/ml_gateway`](../services/ml_gateway/README.md)), **ml_experiments**, опционально **`data_ingestion`** — только для **видео** ([`services/data_ingestion`](../services/data_ingestion/README.md)). Для видео-цепочки порядок: ClickHouse → **analytics** → **ml_gateway** → **ml_experiments** → **`data_ingestion`**; `ANALYTICS_BASE_URL=http://127.0.0.1:8093`. Телеметрия может жить отдельно без этих шагов.

Тома данных Compose (список имён): `zookeeper-data`, `kafka-data`, `es-data`, `clickhouse-data`, `minio-data`, `prometheus-data`, `grafana-data`.
