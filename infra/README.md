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

Данные **PostgreSQL**, **ClickHouse**, **Kafka**, **Elasticsearch**, **MinIO**, **Prometheus** (TSDB) и **Grafana** (в том числе дашборды и настройки, созданные в UI) хранятся в именованных томах и переживают перезапуск контейнеров. Конфиг Prometheus — [`prometheus/prometheus.yml`](prometheus/prometheus.yml); после правок: `docker compose restart prometheus` или lifecycle reload.

## Сервисы и подключение

- **Kafka** (KRaft, без ZooKeeper) — с хоста: `localhost:9092`; внутри сети: `kafka:9092`. Пример: `export KAFKA_BOOTSTRAP_SERVERS=localhost:9092`.

- **Elasticsearch** — http://localhost:9200 (без логина, dev). Внутри сети: `elasticsearch:9200`.

- **Logstash** — приём от Filebeat (Beats) на **5044**, HTTP API на **9600**. Пайплайн: [`logstash/pipeline/logstash.conf`](logstash/pipeline/logstash.conf) → индексы **`piis-docker-logs-YYYY.MM.DD`** (префикс не `logstash-*`: в ES 8 шаблоны Elastic для `logstash-*` ожидают **data stream** и дают 400 на обычный `index` из Logstash).

- **Filebeat** — читает JSON-логи контейнеров Docker и отправляет в Logstash (см. [`filebeat/filebeat.yml`](filebeat/filebeat.yml)): **filestream** по путям `/var/lib/docker/containers/*/*-json.log`, `prospector.scanner.fingerprint.enabled: false`, `compression_level: 0` к Logstash. В `docker-compose.yml` задан явный `command` (`filebeat -e --strict.perms=false -c …`). В Kibana — **Data view** **`piis-docker-logs-*`**, время **`@timestamp`**. Запасной обход Logstash: [`filebeat/filebeat.direct-es.yml`](filebeat/filebeat.direct-es.yml).

- **Kibana** — http://localhost:5601 (логи: Discover → data view **`piis-docker-logs-*`**).

- **ClickHouse** — HTTP с хоста: `http://localhost:8123`; нативный протокол: `localhost:9000`. Внутри сети: `clickhouse:8123`, `clickhouse:9000`. Пользователь **`default`**, пароль пустой (только dev).

  **JDBC** (драйвер `com.clickhouse.jdbc.ClickHouseDriver`, интерфейс HTTP на порту 8123):
  - с хоста: `jdbc:clickhouse://localhost:8123/default`
  - из контейнера в сети `traffic-its`: `jdbc:clickhouse://clickhouse:8123/default`

  Пример `clickhouse-client` с хоста: `clickhouse-client --host localhost --port 9000`.

  **Имитационный справочник** (отдельная БД `its_infra_sim`, не `default`): таблицы `municipalities`, `bus_stops`, `bus_stop_routes`. После `docker compose up -d clickhouse` выполните `infra/clickhouse/bootstrap.sh` (нужен `clickhouse-client` на хосте или Docker compose). Скрипты: [`clickhouse/001_schema.sql`](clickhouse/001_schema.sql), [`clickhouse/002_seed.sql`](clickhouse/002_seed.sql). Проверка: `clickhouse-client -q "SELECT count() FROM its_infra_sim.municipalities"` и `SELECT count() FROM its_infra_sim.bus_stops`.

- **MinIO (S3)** — API с хоста: `http://localhost:9050`; консоль: http://localhost:9051. Учётные данные: **`minioadmin` / `minioadmin`**. Внутри сети: endpoint `http://minio:9000`.

- **Prometheus** — http://localhost:9090. Конфиг [`prometheus/prometheus.yml`](prometheus/prometheus.yml): scrape **`/metrics`** у прикладных сервисов в сети `traffic-its` (`data-ingestion`, `ml-gateway`, `analytics`, `map-portal`, `ml-serving`, `coordinator`), экспортёры **Elasticsearch**, **PostgreSQL**, **Kafka**, **ClickHouse**, **`cadvisor`**, **`blackbox-exporter`** (HTTP health `coordinator` / `ml-serving`). Часть таргетов может быть DOWN, если сервис не запущен.

- **Grafana** — http://localhost:3000, логин по умолчанию **`admin` / `admin`**. Провижининг из [`grafana/provisioning/`](grafana/provisioning/) (папка дашбордов **Traffic**); созданные вручную дашборды и источники сохраняются в томе **`grafana-data`**. В дашборде **`Сервисы`** (uid `kafka-services`): Kafka ingest, ошибки сервисов, `UP/DOWN` по `up{job=…}` для приложений, CPU/RAM контейнеров по имени контейнера (`name` в cAdvisor; лейблы Compose в метриках часто недоступны, в т.ч. на Docker Desktop).

- **MediaMTX** (профиль `ingest`) — `rtsp://localhost:8554`. Запуск: `docker compose --profile ingest up -d --build mediamtx video-source-sim`.

- **video-source-sim** (профиль `ingest`) — читает `../.data/videos/*.mp4`; при отсутствии файлов — синтетические потоки.

- **bus-telemetry-generator** (профиль **`telemetry`**) — контейнер-генератор: раз в **5 с** шлёт gRPC на **`host.docker.internal:50051`** (на хосте должны быть запущены **`data_ingestion`** и **`analytics`**). См. [`services/data-ingestion/README.md`](../services/data-ingestion/README.md).

Профили **`ingest`** (MediaMTX + видео-симулятор) и **`telemetry`** (только генератор автобуса) заданы отдельно. Примеры из каталога `infra`:

```bash
docker compose --profile ingest up -d mediamtx video-source-sim
docker compose --profile telemetry up -d bus-telemetry-generator
docker compose --profile ingest --profile telemetry up -d
```

## Приложения вне compose

**analytics**, **data-ingestion**, **ml-gateway**, **ml-serving**, **map-portal** (карта по HTTP **8096**, к analytics по gRPC **8097** — см. `MAP_GRPC_LISTEN_ADDR` / `ANALYTICS_GRPC_ADDR`), а также офлайн-набор **`ml-experiments`**, запускаются вручную (см. `.env` в каталогах сервисов): [`services/analytics`](../services/analytics/README.md), [`services/data-ingestion`](../services/data-ingestion/README.md), [`services/ml-gateway`](../services/ml-gateway/README.md), [`services/ml-serving`](../services/ml-serving/README.md), [`services/map-portal`](../services/map-portal/README.md), [`ml-experiments`](../ml-experiments/README.md). Для видео: **data-ingestion** → **ml-serving** → **ml-gateway** → **analytics** → **ClickHouse** (и метрики). Для телеметрии автобуса: **analytics** + **data-ingestion** (gRPC и при необходимости Kafka), затем по желанию **`docker compose --profile telemetry up -d bus-telemetry-generator`**.

При старте compose автоматически выполняются one-shot инициализаторы:
- `clickhouse-init` — создаёт таблицы `default.road_incidents` и `default.road_congestion`;
- `minio-init` — создаёт бакет `its-frames`.

Тома данных Compose (список имён): `postgres-data`, `zookeeper-data`, `kafka-data`, `es-data`, `clickhouse-data`, `minio-data`, `prometheus-data`, `grafana-data`.
