# map-portal

Веб-сервис с интерактивной картой (Leaflet, OpenStreetMap). **Данных в ClickHouse нет** — города, остановки и автобусы запрашиваются по **gRPC у analytics** (`map.v1.MapPortal`), который читает `its_infra_sim` из ClickHouse и держит в памяти последние позиции автобусов после приёма телеметрии на `POST /v1/ingest`.

## Цепочка

1. **analytics** — HTTP `:8093`, gRPC **`MAP_GRPC_LISTEN_ADDR`** (по умолчанию `:8097`), ClickHouse для OLAP и для справочника `INFRA_SIM_DATABASE` (`its_infra_sim`).
2. **map-portal** — HTTP (карта), клиент gRPC к analytics.
3. Телеметрия: **data-ingestion** → analytics ingest → обновление in-memory хаба в analytics (как раньше «пересылка в map-portal», но теперь внутри analytics).

## Запуск

Сначала **analytics** (с доступом к ClickHouse и применённым `infra/clickhouse/bootstrap.sh`).

```bash
cd services/map-portal
go run ./cmd/map-portal
```

Браузер: http://127.0.0.1:8096/

Модуль подключает сгенерированный API из соседнего модуля `traffic-analytics` (`replace` в [`go.mod`](go.mod)).

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `LISTEN_ADDR` | `:8096` | HTTP карты |
| `ANALYTICS_GRPC_ADDR` | `127.0.0.1:8097` | Адрес gRPC **analytics** (map.v1.MapPortal) |

## HTTP (страница и JSON для фронта)

- `GET /` — карта.
- `GET /health` — `ok`.
- `GET /api/v1/municipalities` | `/api/v1/stops?municipality_id=` | `/api/v1/buses?municipality_id=` — прокси к analytics по gRPC.

## OpenStreetMap

Тайлы `tile.openstreetmap.org` — для dev нормально; в проде см. [политику OSM](https://operations.osmfoundation.org/policies/tiles/).

## Регенерация proto (analytics)

После правок [`services/analytics/api/map/v1/map_api.proto`](../analytics/api/map/v1/map_api.proto):

```bash
cd services/analytics
protoc --go_out=. --go_opt=module=traffic-analytics \
  --go-grpc_out=. --go-grpc_opt=module=traffic-analytics \
  api/map/v1/map_api.proto
```
