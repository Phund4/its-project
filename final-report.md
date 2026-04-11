### Финальная интеграция ML-сервиса
**Тема**: инференс, связка компонентов, мониторинг, контейнеризация

**Автор**: Попов Александр Иванович  
**Группа**: БВТ2203

Распределённая ИТС: приём видео и телеметрии, ML-инференс, аналитика в ClickHouse, карта, мониторинг.

---

### Шаг 1. Разработка инференс-модуля

Реализован отдельный микросервис **`services/ml-serving`** (FastAPI, Uvicorn):

- **`POST /v1/process`** — приём кадра (multipart), инференс моделей **ДТП (классификация)** и **загруженности (регрессия)**; при настроенном **`ML_GATEWAY_URL`** — передача результата в шлюз (интеграция с Kafka-пайплайном), иначе — JSON-ответ клиенту.
- **`GET /health`** — статус загрузки моделей, пути к чекпойнтам, флаг использования **`winners.json`**.
- Логика инференса вынесена в **`inference_core.py`** (совместимость с офлайн-пайплайном **`ml-experiments`**).

Выбор финальной модели после этапа офлайн-экспериментов (`ml-experiments`):

- явные переменные **`ACCIDENT_CKPT`**, **`CONGESTION_CKPT`**, либо автоматическое чтение из **`.data/ml-experiments/winners.json`** (пайплайн **`ml-experiments/scripts/run_pipeline.sh`**);

Упаковка для рантайма: **`services/ml-serving/Dockerfile`**, переменная **`REPO_ROOT`** для путей к артефактам внутри контейнера.

---

### Шаг 2. Интеграция компонентов

**Цепочка данных:**

1. Источники: `video-source-sim` (RTSP через MediaMTX, профиль `ingest`), `bus-telemetry-generator` (gRPC, профиль `telemetry`).
2. **data-ingestion** — RTSP, S3 (MinIO), вызовы **ml-serving**, телеметрия в Kafka.
3. **coordinator** — назначение источников инстансам, состояние в **PostgreSQL**.
4. **ml-gateway** — приём событий от ML, продюс в Kafka (topic видео).
5. **analytics** — Kafka consumer, запись в **ClickHouse**, бизнес-метрики Prometheus.
6. **map-portal** — gRPC к analytics, UI/API карты.

**Документация по архитектуре:** полная логическая схема (включая coordinator/data-ingestion), протоколы и мониторинг — **`readme.md`** (Mermaid); краткий запуск, управление compose и «куда смотреть» — **`README.md`**. Высокоуровневое описание — **`high-level-design.md`**. При расхождениях итоговой системы с этим планом правки отражаются в этих файлах и в **`infra/`**.

---

### Шаг 3. Внедрение базового мониторинга

| Механизм | Реализация |
|----------|------------|
| Логирование запросов и событий | Логирование в приложениях (например, **`ml-serving`**, предупреждения при отсутствии **`ML_GATEWAY_URL`**); у сервисов на Go — логи в stdout контейнеров. |
| Метрики доступности | **`GET /health`** у сервисов; **blackbox-exporter** опрашивает HTTP health **coordinator** и **ml-serving** (**`infra/prometheus/prometheus.yml`**). |
| Нагрузка на систему | **cAdvisor** — CPU/RAM контейнеров; в Grafana дашборд **«Сервисы»** — ряды по имени контейнера (`name`), т.к. лейблы Compose в метриках cAdvisor на Docker Desktop часто недоступны. |
| Экспорт метрик | **`/metrics`** (формат Prometheus): **data-ingestion**, **ml-gateway**, **analytics**, **map-portal**, **coordinator**, **ml-serving**; сбор — **`infra/prometheus/prometheus.yml`**, TSDB в томе, визуализация — **Grafana** (`infra/grafana/provisioning/`). |

Дашборды по Kafka, трафику и аналитике — JSON в **`infra/grafana/provisioning/dashboards/json/`**.

---

### Шаг 4. Контейнеризация и оркестрация

**Dockerfile по компонентам:**

| Компонент | Путь |
|-----------|------|
| coordinator | `services/coordinator/Dockerfile` |
| data-ingestion | `services/data-ingestion/Dockerfile` |
| ml-serving | `services/ml-serving/Dockerfile` |
| ml-gateway | `services/ml-gateway/Dockerfile` |
| analytics | `services/analytics/Dockerfile` |
| map-portal | `services/map-portal/Dockerfile` |
| video-source-sim | `infra/video-source-sim/Dockerfile` |
| bus-telemetry-generator | `data-generators/telemetry-data/Dockerfile` |

**Оркестрация:** единый **`infra/docker-compose.yml`**: сеть **`traffic-its`**, именованные тома (PostgreSQL, Kafka KRaft, ClickHouse, MinIO, Prometheus, Grafana и т.д.), переменные окружения, healthcheck'и, init топиков Kafka, профили **`ingest`** и **`telemetry`** по необходимости.

Запуск стека из каталога `infra`:

```bash
docker compose up --build -d
```

Минимальные требования: **Docker** с **Compose V2** (`docker compose`). Подробности — **`infra/README.md`**; быстрый старт и таблицы — **`README.md`**, детальная схема — **`readme.md`**.

*Примечание:* **Elasticsearch/Kibana** в compose закомментированы; стек собирается без них.

---

### Шаг 5. Демонстрация и финальная документация

**Тестовые данные:**

- видео для пайплайна: **`.data/videos/*.mp4`** (см. **`ml-experiments/README.md`**);
- полный прогон экспериментов — структуры в **`ml-experiments/data/`** (валидация/тест ДТП, ground truth для заторов и т.д., по README).

**Демонстрация:** запуск `cd infra && docker compose up --build -d`; проверка ML — `curl http://localhost:8000/health` (порт **ml-serving** при пробросе из compose); UI — **Grafana** http://localhost:3000; остальные порты — в **`infra/README.md`**. Опционально: скриншоты дашбордов, схемы из **`readme.md`** / **`README.md`**, сопроводительное видео.

---

### Ссылки на документацию

| Содержание | Файл |
|------------|------|
| Запуск, управление compose, Grafana/Kibana/Prometheus (кратко), общая диаграмма | `README.md` |
| Полная логическая схема (вторая диаграмма coordinator), протоколы, мониторинг | `readme.md` |
| Установка и запуск (Docker) | `infra/README.md` |
| Архитектура (высокий уровень) | `high-level-design.md` |
| ML-эксперименты и выбор моделей | `ml-experiments/README.md` |
| Инференс-сервис | `services/ml-serving/README.md` |
