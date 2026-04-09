# data-ingestion и coordinator: новый инстанс

Чтобы процесс `data-ingestion` участвовал в назначениях, его **идентичность** должна совпадать с одной строкой в **`services/coordinator/ingestion_instances.yaml`** в блоке **`zone_workers`** для нужной зоны.

## Идентичность (должна совпадать с YAML)

| Роль | Переменная окружения | Поле в `zone_workers` |
|------|----------------------|-------------------------|
| Зона | `COORDINATOR_ZONE_ID` | ключ зоны, например `zone-a` |
| Кластер | `COORDINATOR_CLUSTER_ID` | `cluster_id` |
| Инстанс | `COORDINATOR_INSTANCE_ID` | `instance_id` (уникально в зоне среди ваших процессов) |

**Правило:** для каждого запущенного процесса ingestion добавьте в `zone_workers.<zone_id>` элемент в **`ingestion_instances.yaml`** (не в `sources.yaml`):

```yaml
- cluster_id: "<тот же COORDINATOR_CLUSTER_ID>"
  instance_id: "<уникальный COORDINATOR_INSTANCE_ID>"
  url: "http://127.0.0.1:9091/metrics"   # опционально: куда смотреть (часто /metrics)
```

Список с `url` также отдаёт coordinator: `GET /v1/ingestion_instances?zone_id=zone-a`.

Иначе coordinator не будет считать этот инстанс кандидатом для назначений, worker status не будет учитываться при балансировке.

## Обязательные переменные (связь с coordinator)

| Переменная | Пример | Заметка |
|------------|--------|---------|
| `COORDINATOR_BASE_URL` | `http://127.0.0.1:8098` | без завершающего `/` |
| `COORDINATOR_ZONE_ID` | `zone-a` | как в `sources` и в ключе `zone_workers` |
| `COORDINATOR_CLUSTER_ID` | `cluster-1` | как в YAML |
| `COORDINATOR_INSTANCE_ID` | `ingest-a1` | у второго процесса — другой, например `ingest-a2` |

Без этих переменных сервис не стартует (нет назначений источников).

## Чеклист: добавили инстанс

1. В **`services/coordinator/ingestion_instances.yaml`** под `zone_workers.<ваша_зона>` добавлена пара `cluster_id` / `instance_id`.
2. Перезапущен **coordinator** (конфиг читается при старте).
3. Для нового процесса **data-ingestion** заданы те же `COORDINATOR_*`, что и в YAML-строке.
4. Если на **одном хосте** два процесса — разведены **порты** (иначе bind error):

   | Переменная | Назначение |
   |------------|------------|
   | `METRICS_LISTEN_ADDR` | Prometheus, по умолчанию `:9091` |
   | `TELEMETRY_GRPC_LISTEN_ADDR` | gRPC телеметрии, по умолчанию `:50051` |
   | `TELEMETRY_HTTP_LISTEN_ADDR` | HTTP телеметрии, по умолчанию `:8094` |

   Пример второго инстанса: `:9092`, `:50052`, `:8095`.

## Как проверить, что инстанс «виден» логике назначений

1. Запустите coordinator и поднимите процесс с нужными `COORDINATOR_*`.
2. Убедитесь, что worker status принимается (в логах ingestion нет постоянных ошибок отправки worker status; при необходимости `GET http://127.0.0.1:8098/v1/workers`).
3. Запросите назначения для **этого** инстанса (подставьте свои значения):

   ```bash
   curl -sS "http://127.0.0.1:8098/v1/assignments?zone_id=zone-a&cluster_id=cluster-1&instance_id=ingest-a1&data_class=road_segment_video"
   ```

   В ответе `items` — только источники, назначенные **данному** `cluster_id` + `instance_id` для указанного **`data_class`** (см. `services/coordinator/README.md`).

## Поведение coordinator

- **Назначения** источников по зоне распределяются между живыми воркерами из `ingestion_instances.yaml` (worker status + загрузка). См. `services/coordinator/README.md`.
- **Идентификаторы** в worker status и в GET `/v1/assignments` должны совпадать с теми, что в YAML.

## См. также

- Полный список env и режимов: `README.md` в этом каталоге.
- Файлы coordinator: `sources.yaml` (источники), `ingestion_instances.yaml` (инстансы), `services/coordinator/README.md`.
- Пример локального `.env`: `.env` (второй инстанс — другой `COORDINATOR_INSTANCE_ID` и порты).
