# Эксперименты с ML и выбор модели

В каталоге **`ml-experiments`** только офлайн-эксперименты: прогон бенчмарков, сравнение моделей и выбор победителя.

## Назначение

- Запуск экспериментов на ваших видео.
- Выбор лучших чекпойнтов для треков accident и congestion.
- Сохранение результатов в `.data` в корне репозитория, чтобы рантайм мог их подхватить.

## Быстрый запуск

Что нужно заранее:

- чекпойнты в `.data/ml-experiments/artifacts/`:
  - `.data/ml-experiments/artifacts/accident/{baseline-cnn,resnet18}/best.pt`
  - `.data/ml-experiments/artifacts/congestion/{tiny-cnn,linear-resnet}/best.pt`
- один или несколько файлов `*.mp4` в `.data/videos/`
- для основного протокола оценки (без `data/train`):
  - **валидация accident:** `ml-experiments/data/CCTV-accidents/val/` с подпапками классов (`Accident`, `Non Accident` и т.п., как в `BinaryFolderImageDataset`);
  - **тест accident:** `…/data/accident/test/` — такая же структура;
  - **тест congestion:** после `prepare_video_ground_truth.py` — `…/congestion/video-gt/labels/test.csv` и `images/test/`.

```bash
cd ml-experiments
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

bash scripts/run_pipeline.sh
```

Пайплайн пишет в `.data/ml-experiments/`:

- `benchmark/video_gt_results.json`
- `benchmark/video_gt_report.md`
- `winners.json` (выбранные чекпойнты; пути **относительно корня репозитория** — их подхватывает `services/ml-serving`, если не заданы `ACCIDENT_CKPT` / `CONGESTION_CKPT`)

## Сервис инференса

Раздачу моделей вынесли в **`services/ml-serving`**.

Запуск:

```bash
cd services/ml-serving
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn api.main:app --host 0.0.0.0 --port 8000 --no-access-log
```
