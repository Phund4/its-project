# Бенчмарк по видео-разметке

## Как читать отчёт

Метрики по треку accident считаются на **смешанном тесте** (есть и аварии, и неаварии). Основной критерий выбора модели — F1 по классу «crash» с учётом precision/recall и задержки.

Допущения:

- тестовый split CCTV: оба класса — Accident и Non Accident;
- целевое значение загруженности 0.0 на каждом кадре (чем ближе выход модели к 0, тем лучше).

## Победители

- accident: `resnet18`
- congestion: `tiny_cnn`

## Accident

| модель | accuracy | precision_crash | recall_crash | f1_crash | f1_normal | mean_P_crash | p95_P_crash | fps |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| baseline_cnn | 0.4700 | 0.4700 | 1.0000 | 0.6395 | 0.0000 | 0.5272 | 0.5294 | 63.45 |
| resnet18 | 0.6700 | 0.6750 | 0.5745 | 0.6207 | 0.7080 | 0.4871 | 0.7071 | 10.24 |

## Congestion (цель 0)

| модель | mae | rmse | fps |
|---|---:|---:|---:|
| tiny_cnn | 0.1800 | 0.1834 | 53.89 |
| linear_resnet | 0.1976 | 0.1995 | 8.83 |

## Графики

- `video_gt_f1_crash.png`
- `video_gt_mean_crash_prob.png`
- `video_gt_f1_normal.png`
- `video_gt_accident_fps.png`
- `video_gt_cong_mae.png`
- `video_gt_cong_fps.png`
