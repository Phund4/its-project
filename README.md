# PIIS: Логическая схема системы

Ниже общая диаграмма сервисов проекта и протоколов взаимодействия.

```mermaid
flowchart LR
  %% -------------------- Producers --------------------
  subgraph Producers["Источники данных"]
    VideoSim["video-source-sim"]
    BusGen["bus-telemetry-generator"]
  end

  %% -------------------- Core services --------------------
  subgraph Services["Сервисы (запускаются вручную)"]
    DI["data-ingestion"]
    MLS["ml-serving"]
    MLG["ml-gateway"]
    AN["analytics"]
    MP["map-portal"]
  end

  %% -------------------- Infra --------------------
  subgraph Infra["Инфраструктура (Docker Compose)"]
    MTX["MediaMTX"]
    MinIO["MinIO / S3"]
    KFK["Kafka"]
    CH["ClickHouse"]
    Prom["Prometheus"]
    Graf["Grafana"]
    KUI["Kafka UI"]
  end

  %% -------------------- Video pipeline --------------------
  VideoSim -->|RTSP| MTX
  MTX -->|RTSP| DI
  DI -->|S3 API (HTTP)| MinIO
  DI -->|HTTP multipart /v1/process| MLS
  MLS -->|HTTP JSON /v1/road-events| MLG
  MLG -->|Kafka produce (video topic)| KFK
  KFK -->|Kafka consume (video + telemetry topics)| AN
  AN -->|Native protocol TCP:9000| CH

  %% -------------------- Telemetry pipeline --------------------
  BusGen -->|gRPC bus.v1 SendBusTelemetry| DI
  DI -->|Kafka produce (telemetry topic)| KFK

  %% -------------------- Map pipeline --------------------
  MP -->|gRPC map.v1.MapPortal| AN
  AN -->|HTTP UI + JSON API| MP

  %% -------------------- Observability --------------------
  Prom -->|HTTP scrape /metrics| DI
  Prom -->|HTTP scrape /metrics| MLG
  Prom -->|HTTP scrape /metrics| AN
  Prom -->|HTTP scrape /metrics| MP
  Graf -->|PromQL / datasource| Prom
  KUI -->|HTTP UI| KFK
```

## Ключевые протоколы

- `RTSP` — видеопотоки от симулятора через `MediaMTX` в `data-ingestion`.
- `HTTP` — вызовы ML (`data-ingestion -> ml-serving`) и API-взаимодействия.
- `Kafka` — асинхронная передача событий видео/телеметрии в `analytics`.
- `gRPC` — телеметрия автобусов (`bus.v1`) и API карты (`map.v1`).
- `S3 API` — сохранение кадров в `MinIO`.
- `Prometheus scrape` — сбор метрик, визуализация через `Grafana`.
