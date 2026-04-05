# Бенчмарк на ваших видео и ML API

## Быстрый прогон

Нужны чекпойнты и видео:

- `artifacts/accident/{baseline-cnn,resnet18}/best.pt`
- `artifacts/congestion/{tiny-cnn,linear-resnet}/best.pt`
- один или несколько `*.mp4` в `../.data/videos/` (корень репозитория)

```bash
cd ml-experiments
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

bash scripts/run_pipeline.sh
```

Скрипт: извлекает кады (~3 FPS) → строит разметку «везде normal» и congestion `0` → считает метрики → пишет `benchmark/video_gt_report.md`.

## Как читать отчёт

Числа **0 / низкая P(crash)** на видео без ДТП — ожидаемое поведение *идеальной* модели под вашу съёмку.  
Если `mean_P_crash` высокая и `false_crash_rate` ≈ 1, это обычно **несовпадение домена** (модель училась на других данных), а не ошибка скрипта оценки.

## Mock ML API

```bash
cd ml-experiments
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn api.main:app --host 0.0.0.0 --port 8000 --no-access-log
```

Переменные читаются из **`.env`** в каталоге `ml-experiments` (или из **`ENV_FILE`**). Файл **`.env`** в git не коммитится (после клона создайте его заново: `ACCIDENT_CKPT`, `CONGESTION_CKPT`, при необходимости `ML_GATEWAY_URL` и т.д.). Уже заданные в оболочке переменные **не перезаписываются** значениями из файла.

**Интервал модели загруженности:** при вызове `/v1/process` с полями `segment_id` и `camera_id` инференс **загруженности** выполняется не чаще чем раз в **`CONGESTION_INTERVAL_SEC`** (секунды, по умолчанию **2**) на каждую камеру; классификация **инцидента** считается **на каждый** кадр. Для локального JSON без `segment_id`/`camera_id` загруженность не кэшируется по интервалу. В `/health` отдаётся `congestion_interval_sec`.

Проверка:

```bash
curl -s http://127.0.0.1:8000/health
```

Пример (кадр из вашего видео):

```bash
# Без ML_GATEWAY_URL — ответ JSON с полями incident/congestion.
curl -s -X POST "http://127.0.0.1:8000/v1/process" \
  -F "image=@data/videos/frames/cartraffic01/f_000001.png"

# С ML_GATEWAY_URL — нужны поля контекста; HTTP 204, тело пустое; JSON уходит в ml_gateway.
curl -s -o /dev/null -w "%{http_code}" -X POST "http://127.0.0.1:8000/v1/process" \
  -F "image=@data/videos/frames/cartraffic01/f_000001.png" \
  -F "segment_id=ring-road-5" -F "camera_id=cam-01" \
  -F "s3_key=its-ingest/2026-04-03/cam-01/frame_1.png" \
  -F "observed_at=2026-04-03T10:00:00Z"
```

## RTSP

Инфраструктура: [../infra/README.md](../infra/README.md). Видео кладите в `../.data/videos`.

## Прочее

Kafka для будущих источников: [../infra/README.md](../infra/README.md).
