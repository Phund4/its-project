# PIIS: Логическая схема системы

Ниже общая диаграмма сервисов проекта и протоколов взаимодействия.

```mermaid
flowchart LR
  subgraph Producers["Источники данных"]
    VideoSim[video-source-sim]
    BusGen[bus-telemetry-generator]
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
    Prom[Prometheus]
    Graf[Grafana]
  end

  VideoSim -->|RTSP| MTX
  DI -->|HTTP heartbeat assignments| COORD
  COORD -->|HTTP assignments| DI
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

  Prom -->|HTTP scrape metrics| DI
  Prom -->|HTTP scrape metrics| MLG
  Prom -->|HTTP scrape metrics| AN
  Prom -->|HTTP scrape metrics| MP
  Graf -->|PromQL datasource| Prom
```

## coordinator и data-ingestion (детально)

Два инстанса `data-ingestion` в одном кластере, назначения и резерв из `sources.yaml`. Heartbeat обновляет «живость» владельца; при таймауте primary источник переключается на reserve.

```mermaid
flowchart TB
  subgraph COORD[coordinator]
    SRC[sources.yaml primary reserve]
    API[GET assignments POST heartbeat]
    SRC --> API
  end

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
  API -->|GET assignments camera| DI1
  API -->|GET assignments camera| DI2
  API -->|GET assignments telemetry| DI1
  API -->|GET assignments telemetry| DI2

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
