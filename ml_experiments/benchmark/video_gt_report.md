# Video ground-truth benchmark

## How to read this

При разметке «на видео везде normal» accuracy — это доля кадров, где argmax == индекс класса normal из чекпойнта. Для двух классов это же число, что 1 − false_crash_rate (доля предсказаний «crash»). Если модель на каждом кадре выбирает «crash» и softmax это подтверждает (высокая mean_P_crash), accuracy=0 — ожидаемый и корректный результат; это не перевёрнутая метрика.

Assumptions:
- all frames labeled normal (no crash in source videos)
- target is 0.0 for every frame (closer model output to 0 is better)

## Winners
- accident: `baseline_cnn`
- congestion: `tiny_cnn`

## Accident (GT = normal on every frame)

При двух классах: **accuracy = 1 − false_crash_rate** (доля кадров, где предсказан класс normal).

| model | accuracy | f1_normal | false_crash_rate | mean_P_crash | p95_P_crash | fps |
|---|---:|---:|---:|---:|---:|---:|
| baseline_cnn | 0.0000 | 0.0000 | 1.0000 | 0.9240 | 0.9989 | 57.80 |
| resnet18 | 0.0000 | 0.0000 | 1.0000 | 0.9963 | 1.0000 | 9.24 |

_f1_crash_omit: on all-normal GT, F1 for the crash class is not meaningful.

## Congestion (target 0)

| model | mae | rmse | fps |
|---|---:|---:|---:|
| tiny_cnn | 0.1800 | 0.1834 | 56.12 |
| linear_resnet | 0.1976 | 0.1995 | 8.76 |

## Charts
- `video_gt_false_crash.png`
- `video_gt_mean_crash_prob.png`
- `video_gt_f1_normal.png`
- `video_gt_accident_fps.png`
- `video_gt_cong_mae.png`
- `video_gt_cong_fps.png`
