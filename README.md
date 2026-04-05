# Платформа данных и видеоаналитики для транспортного контура

- [high-level-design.md](high-level-design.md) — архитектура и роли сервисов  
- [validation-scaling.md](validation-scaling.md) — валидация и масштабирование  
- [ml-testing-design.md](ml-testing-design.md) — методология испытаний готовых ML-моделей  

## Потоки данных (раздельно)

- **Видео с камер** → [`services/data_ingestion`](services/data_ingestion/README.md) (S3, вызов ML) → при необходимости `ml_experiments` → `ml_gateway` → `analytics`. Нужны RTSP/MinIO/ML только для этого контура.
- **Телеметрия** (отдельная разработка): **не требует** запуска `data_ingestion`, RTSP и пайплайна кадров. Обычно достаточно своего сервиса/консьюмера и общей инфраструктуры (**Kafka**, **ClickHouse** и т.д. из [`infra/`](infra/README.md)) без видео.

## Репозиторий

| Каталог | Назначение |
|---------|------------|
| [infra/](infra/) | Docker Compose: Kafka, Elasticsearch, ClickHouse, MinIO, Prometheus, Grafana |
| [ml_experiments/](ml_experiments/) | Бенчмарк моделей на видео, ML API |
| [services/](services/) | **`data_ingestion`** (видео), `ml_gateway`, `analytics` и др. |

Краткий запуск: [infra/README.md](infra/README.md), [ml_experiments/README.md](ml_experiments/README.md).
