# PIIS: Логическая схема системы

Ниже общая диаграмма сервисов проекта и протоколов взаимодействия.

## Источники данных

- **Потоковые** — синтетические видеопотоки и телеметрия ТС (`video-source-sim`, `bus-telemetry-generator`), далее обработка в `data-ingestion` и шина Kafka.
- **Статическая инфраструктура** — справочные данные по населённым пунктам и остановкам (агрегированная информация о «инфраструктуре города» без привязки к конкретному поставщику или формату хранения в этом документе). Они подмешиваются к рантайму при работе карты и сопоставления сегментов/объектов; детали схемы БД и обновлений см. в репозитории сервисов.

## Взаимодействие с системой

- **Развёртывание и сопровождение** — подъём стека из каталога `infra` одной командой `docker compose` (см. **`infra/README.md`**): инфраструктура (Kafka, ClickHouse, MinIO, Prometheus, Grafana и прикладные сервисы) в общей сети Docker.
- **Наблюдение за работой** — **Grafana** (`http://localhost:3000`): дашборды по Kafka, бизнес-метрикам analytics, доступности сервисов (`up`), CPU/RAM контейнеров (cAdvisor). **Prometheus** — сырьё для запросов и алертов; health критичных сервисов дублируется **blackbox-exporter**.
- **Операционный и UI-слой** — **map-portal** отдаёт HTTP JSON поверх gRPC к **analytics** (карта, справочники в духе муниципалитетов/остановок/ТС — см. README сервиса); это точка входа для сценариев «посмотреть состояние на карте» без прямого доступа к ClickHouse.
- **ML-контур** — `data-ingestion` вызывает **ml-serving** по HTTP; при настройке **`ML_GATEWAY_URL`** результаты уходят в **ml-gateway** → Kafka → **analytics**; ручная проверка: `GET /health` у `ml-serving` и `coordinator`.
- **Координация источников** — инстансы `data-ingestion` получают назначения от **coordinator** (heartbeat, `GET assignments`), состояние — в **PostgreSQL**.

```mermaid
flowchart LR
  subgraph Producers["Источники данных"]
    VideoSim[video-source-sim]
    BusGen[bus-telemetry-generator]
    StaticInfra[стат. справочник инфраструктуры]
  end

  subgraph Services["Сервисы"]
    DI[data-ingestion]
    COORD[coordinator]
    MLS[ml-serving]
    MLG[ml-gateway]
    AN[analytics]
    MP[map-portal]
  end

  subgraph Infra["Инфраструктура"]
    MTX[MediaMTX]
    MinIO[MinIO S3]
    KFK[Kafka]
    CH[ClickHouse]
    PG[PostgreSQL]
    Prom[Prometheus]
    Graf[Grafana]
  end

  VideoSim -->|RTSP| MTX
  DI -->|HTTP heartbeat assignments| COORD
  COORD -->|HTTP assignments| DI
  COORD -->|state read/write| PG
  MTX -->|RTSP| DI
  DI -->|HTTP S3 API| MinIO
  DI -->|HTTP multipart v1 process| MLS
  MLS -->|HTTP JSON v1 road-events| MLG
  MLG -->|Kafka produce video topic| KFK
  KFK -->|Kafka consume video telemetry topics| AN
  AN -->|TCP 9000 native| CH

  BusGen -->|gRPC bus telemetry| DI
  DI -->|Kafka produce telemetry topic| KFK

  MP -->|gRPC map portal| AN
  AN -->|HTTP UI JSON API| MP
  StaticInfra -.->|справочник| AN
  StaticInfra -.->|справочник| MP

  Prom -->|HTTP scrape metrics| DI
  Prom -->|HTTP scrape metrics| MLG
  Prom -->|HTTP scrape metrics| AN
  Prom -->|HTTP scrape metrics| MP
  Prom -->|HTTP scrape metrics| MLS
  Prom -->|HTTP scrape metrics| COORD
  Graf -->|PromQL datasource| Prom
```

## coordinator и data-ingestion (детально)

Два инстанса `data-ingestion` в одном кластере. Список инстансов — в `ingestion_instances.yaml` у coordinator; `sources.yaml` — только каталог источников. Назначение — по загрузке среди живых инстансов. В production-сценарии coordinator работает в active-active и читает общее состояние из PostgreSQL.

```mermaid
flowchart TB
  subgraph COORD[coordinator]
    SRC[sources.yaml + ingestion_instances.yaml]
    API[GET assignments POST heartbeat]
    SRC --> API
  end
  PG[(PostgreSQL)]

  subgraph DI1[data-ingestion ingest-a1]
    V1[video RTSP ffmpeg S3 ML]
    T1[gRPC HTTP telemetry]
  end

  subgraph DI2[data-ingestion ingest-a2]
    V2[video RTSP ffmpeg S3 ML]
    T2[gRPC HTTP telemetry]
  end

  MTX[MediaMTX RTSP]
  GEN[bus-telemetry-generator]

  DI1 -->|POST heartbeat| API
  DI2 -->|POST heartbeat| API
  API -->|state read/write| PG
  API -->|GET road_segment_video| DI1
  API -->|GET road_segment_video| DI2
  API -->|GET vehicle_bus_telemetry| DI1
  API -->|GET vehicle_bus_telemetry| DI2

  MTX -->|RTSP по назначению| V1
  MTX -->|RTSP по назначению| V2
  GEN -->|gRPC или HTTP| T1
  GEN -->|gRPC или HTTP| T2

  V1 --> MinIO[(MinIO)]
  V2 --> MinIO
  V1 --> MLS[ml-serving]
  V2 --> MLS
```

## Ключевые протоколы

- `RTSP` — видеопотоки от симулятора через `MediaMTX` в `data-ingestion`.
- `HTTP` — вызовы ML (`data-ingestion -> ml-serving`) и API-взаимодействия.
- `Kafka` — асинхронная передача событий видео/телеметрии в `analytics`.
- `gRPC` — телеметрия автобусов (`bus.v1`) и API карты (`map.v1`).
- `S3 API` — сохранение кадров в `MinIO`.
- `Prometheus scrape` — сбор метрик, визуализация через `Grafana`.

## Мониторинг и health

- Основные дашборды в Grafana: **`Сервисы`** (Kafka, ошибки, `up`, CPU/RAM контейнеров), **`Ingest, gateway, analytics`** (метрики пайплайна видео/аналитики).
- Health-check `coordinator`/`ml-serving` собирается через `blackbox-exporter`.
- CPU/RAM по docker-сервисам собирается через `cadvisor`.
