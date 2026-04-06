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

### Профиль `ingest` (MediaMTX + видео-симулятор)

Поднять только RTSP и публикацию тестовых потоков:

```bash
docker compose --profile ingest up -d mediamtx video-source-sim
```

**Остановить** эти контейнеры, не трогая остальной стек (Kafka, ClickHouse и т.д.):

```bash
docker compose --profile ingest stop mediamtx video-source-sim
```

Снова запустить:

```bash
docker compose --profile ingest start mediamtx video-source-sim
```

Удалить контейнеры профиля `ingest` (данные в томах основного стека не затрагиваются):

```bash
docker compose --profile ingest rm -sf mediamtx video-source-sim
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

  **Имитационный справочник** (отдельная БД `its_infra_sim`, не `default`): таблицы `municipalities`, `bus_stops`, `bus_stop_routes`. После `docker compose up -d clickhouse` выполните `infra/clickhouse/bootstrap.sh` (нужен `clickhouse-client` на хосте или Docker compose). Скрипты: [`clickhouse/001_schema.sql`](clickhouse/001_schema.sql), [`clickhouse/002_seed.sql`](clickhouse/002_seed.sql). Проверка: `clickhouse-client -q "SELECT count() FROM its_infra_sim.municipalities"` и `SELECT count() FROM its_infra_sim.bus_stops`.

- **MinIO (S3)** — API с хоста: `http://localhost:9050`; консоль: http://localhost:9051. Учётные данные: **`minioadmin` / `minioadmin`**. Внутри сети: endpoint `http://minio:9000`.

- **Prometheus** — http://localhost:9090. Конфиг [`prometheus/prometheus.yml`](prometheus/prometheus.yml) опрашивает **процессы на хосте** (`data_ingestion` :9091, `ml_gateway` :8092, `analytics` :8093) через **`host.docker.internal`** (у сервиса `prometheus` задан `extra_hosts`). Это только мониторинг: сами Go-сервисы **не** зависят от Docker. Нет процесса — таргет будет DOWN. Метрики: `data_ingestion_*`, `ml_gateway_*`, `analytics_*` и т.д.

- **Grafana** — http://localhost:3000, логин по умолчанию **`admin` / `admin`**. Провижининг из [`grafana/provisioning/`](grafana/provisioning/) (папка дашбордов **Traffic**); созданные вручную дашборды и источники сохраняются в томе **`grafana-data`**.

- **MediaMTX** (профиль `ingest`) — `rtsp://localhost:8554`. Запуск: `docker compose --profile ingest up -d --build mediamtx video-source-sim`.

- **video-source-sim** (профиль `ingest`) — читает `../.data/videos/*.mp4`; при отсутствии файлов — синтетические потоки.

- **bus-telemetry-generator** (профиль **`telemetry`**) — контейнер-генератор: раз в **5 с** шлёт gRPC на **`host.docker.internal:50051`** (на хосте должны быть запущены **`data_ingestion`** с `TELEMETRY_GRPC_ENABLED=true` и **`analytics`**). См. [`services/data-ingestion/README.md`](../services/data-ingestion/README.md).

Профили **`ingest`** (MediaMTX + видео-симулятор) и **`telemetry`** (только генератор автобуса) заданы отдельно. Примеры из каталога `infra`:

```bash
docker compose --profile ingest up -d mediamtx video-source-sim
docker compose --profile telemetry up -d bus-telemetry-generator
docker compose --profile ingest --profile telemetry up -d
```

## Приложения вне compose

**analytics**, **data-ingestion**, **ml-gateway**, **map-portal** (карта HTTP **8096**, к analytics по gRPC **8097** — см. `MAP_GRPC_LISTEN_ADDR` / `ANALYTICS_GRPC_ADDR`), **ml-experiments** запускаются вручную из консоли (см. `.env` в каталогах сервисов): [`services/analytics`](../services/analytics/README.md), [`services/data-ingestion`](../services/data-ingestion/README.md), [`services/ml-gateway`](../services/ml-gateway/README.md), [`services/map-portal`](../services/map-portal/README.md). Для видео: ClickHouse → **analytics** → **ml-gateway** → **ml-experiments** → **data-ingestion**. Для телеметрии автобуса: **analytics** + **data-ingestion** (режим gRPC, `ANALYTICS_INGEST_URL=http://127.0.0.1:8093/v1/ingest`), затем при необходимости **`docker compose --profile telemetry up -d bus-telemetry-generator`**.

Тома данных Compose (список имён): `zookeeper-data`, `kafka-data`, `es-data`, `clickhouse-data`, `minio-data`, `prometheus-data`, `grafana-data`.
