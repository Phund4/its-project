# Video benchmark

## How to read this

Метрики accident считаются на mixed test (есть и аварии, и неаварии). Основная метрика выбора — F1 по классу crash с учётом recall/precision и latency.

Assumptions:
- CCTV test split with both classes: Accident and Non Accident
- target is 0.0 for every frame (closer model output to 0 is better)

## Winners
- accident: `resnet18`
- congestion: `tiny_cnn`

## Accident

| model | accuracy | precision_crash | recall_crash | f1_crash | f1_normal | mean_P_crash | p95_P_crash | fps |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| baseline_cnn | 0.4700 | 0.4700 | 1.0000 | 0.6395 | 0.0000 | 0.5272 | 0.5294 | 63.45 |
| resnet18 | 0.6700 | 0.6750 | 0.5745 | 0.6207 | 0.7080 | 0.4871 | 0.7071 | 10.24 |

## Congestion (target 0)

| model | mae | rmse | fps |
|---|---:|---:|---:|
| tiny_cnn | 0.1800 | 0.1834 | 53.89 |
| linear_resnet | 0.1976 | 0.1995 | 8.83 |

## Charts
- `video_gt_f1_crash.png`
- `video_gt_mean_crash_prob.png`
- `video_gt_f1_normal.png`
- `video_gt_accident_fps.png`
- `video_gt_cong_mae.png`
- `video_gt_cong_fps.png`
