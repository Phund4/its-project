# Сервис ml-serving

Сервис инференса ML в рантайме (эндпоинт `/v1/process`).

- **`ml-experiments`** — офлайн-эксперименты; пишет `.data/ml-experiments/winners.json`.
- **`services/ml-serving`** — инференс: если **`ACCIDENT_CKPT` / `CONGESTION_CKPT` не заданы**, подставляются чекпойнты из **`WINNERS_JSON`** (по умолчанию `.data/ml-experiments/winners.json`). Явные переменные в `.env` имеют приоритет.

## Запуск

```bash
cd services/ml-serving
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn api.main:app --host 0.0.0.0 --port 8000 --no-access-log
```

Настройки — в **`services/ml-serving/.env`** (или переменные окружения). Запускайте из каталога сервиса или задайте абсолютные пути к чекпойнтам и к `WINNERS_JSON`.

## Проверка

```bash
curl -s http://127.0.0.1:8000/health
```

В ответе: `accident_checkpoint`, `congestion_checkpoint`, `winners_json`, флаги `*_from_winners_json`.
