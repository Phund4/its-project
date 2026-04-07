# coordinator

Лёгкий сервис-координатор источников данных для `data-ingestion`.

Сейчас покрывает:
- каталог источников по зонам;
- выдачу назначений (`assignments`) для конкретного ingestion-инстанса;
- heartbeat от ingestion-инстансов;
- **пул воркеров по зоне** (`zone_workers`): назначение источника — среди живых инстансов зоны по **минимальной загрузке** (heartbeat: `assignments` + `load`), при равенстве — порядок в списке зоны.

## Конфигурация

- `LISTEN_ADDR` — адрес HTTP (по умолчанию `:8098`)
- `SOURCES_CONFIG_PATH` — путь к YAML только с **источниками** (по умолчанию `sources.yaml`)
- `INGESTION_INSTANCES_PATH` — путь к YAML с **инстансами data-ingestion** по зонам (по умолчанию `ingestion_instances.yaml`)
- `HEARTBEAT_TIMEOUT_SEC` — сколько секунд heartbeat считается свежим (по умолчанию `30`)
- `DATABASE_URL` — строка подключения к PostgreSQL (если пуста, режим in-memory)

Файл `.env` в каталоге `services/coordinator` подхватывается автоматически (можно переопределить через `ENV_FILE`).

## Источники (`sources.yaml`)

Обязательное поле **`data_class`** классифицирует тип данных и подразумеваемый пайплайн:

| `data_class` | Вход | Назначение (data-ingestion) |
|--------------|------|-----------------------------|
| `road_segment_video` | RTSP камеры дорожного участка | S3 + ML |
| `vehicle_bus_telemetry` | телеметрия ТС (gRPC/HTTP) | analytics / Kafka |

Для `road_segment_video` нужны `segment_id`, `camera_id`, `rtsp_url`. Для `vehicle_bus_telemetry` — только `zone_id` (и при желании `source_id`).

## API

- `GET /health`
- `GET /v1/sources?zone_id=zone-a`
- `GET /v1/assignments?zone_id=zone-a&cluster_id=cluster-1&instance_id=ingest-a1&data_class=road_segment_video`
- `GET /v1/assignments?zone_id=zone-a&cluster_id=cluster-1&instance_id=ingest-a1&data_class=vehicle_bus_telemetry`
- `POST /v1/workers/heartbeat`
- `GET /v1/workers`
- `GET /v1/ingestion_instances?zone_id=zone-a` — каталог инстансов из `ingestion_instances.yaml` (в т.ч. поле `url`)

## Назначение и failover

- В **`sources.yaml`** — каталог источников с **`data_class`**. В **`ingestion_instances.yaml`** — **`zone_workers`**: пул `{ cluster_id, instance_id }` на зону.
- Для каждого источника владелец выбирается среди **живых** воркеров зоны с минимальной оценкой загрузки; при отсутствии живых воркеров источник **не выдаётся**.

## Репликация coordinator (active-active)

- Coordinator запускается в 2+ репликах за одним балансировщиком (L4/L7), все реплики подключены к **одной PostgreSQL**.
- В PostgreSQL хранится общее состояние: `sources`, `ingestion_instances`, `worker_heartbeats`.
- Любая реплика принимает heartbeat и пишет в `worker_heartbeats`; любая реплика на `GET /v1/assignments` читает общее состояние и считает назначение детерминированно.
- Для клиента (`data-ingestion`) coordinator выглядит как единая точка (`COORDINATOR_BASE_URL` балансировщика), без «мастера».
- При падении одной реплики остальные продолжают работу без failover-логики на уровне приложения.

## PostgreSQL: миграции и seed

- Миграции и стартовые данные вынесены в `infra/postgres/init`:
  - `001_coordinator_schema.sql`
  - `002_coordinator_seed.sql`
- Эти скрипты применяются контейнером `postgres` при инициализации volume.
- Coordinator **не** выполняет миграции из Go-кода.

## Запуск

```bash
cd services/coordinator
go run ./cmd/coordinator
```
