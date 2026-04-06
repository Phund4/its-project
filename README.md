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

## Ключевые протоколы

- `RTSP` — видеопотоки от симулятора через `MediaMTX` в `data-ingestion`.
- `HTTP` — вызовы ML (`data-ingestion -> ml-serving`) и API-взаимодействия.
- `Kafka` — асинхронная передача событий видео/телеметрии в `analytics`.
- `gRPC` — телеметрия автобусов (`bus.v1`) и API карты (`map.v1`).
- `S3 API` — сохранение кадров в `MinIO`.
- `Prometheus scrape` — сбор метрик, визуализация через `Grafana`.
