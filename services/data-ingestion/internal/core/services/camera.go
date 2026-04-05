package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"data-ingestion/internal/adapters/capture"
	"data-ingestion/internal/adapters/metrics"
	"data-ingestion/internal/adapters/ml"
	"data-ingestion/internal/adapters/s3"
	"data-ingestion/internal/config"
	"data-ingestion/internal/core/domain"
)

// RunCamera в цикле подключается к RTSP, читает кадры, заливает PNG в S3 и вызывает ML process.
func RunCamera(
	ctx context.Context,
	cam config.Camera,
	store *s3store.Client,
	mlc *mlclient.Client,
	s3Prefix string,
	ffmpegPath string,
	targetFPS float64,
) {
	log := slog.With("segment", cam.SegmentID, "camera", cam.CameraID)
	prefix := strings.Trim(s3Prefix, "/")
	var frameNo atomic.Uint64
	var lastUpstreamLog time.Time
	backoff := time.Duration(reconnectBackoffSec) * time.Second

	for ctx.Err() == nil {
		subCtx, cancel := context.WithCancel(ctx)
		pipe, err := capture.FFmpegPipe(subCtx, ffmpegPath, cam.RTSPURL, targetFPS)
		if err != nil {
			metrics.OperationErrors.WithLabelValues("ffmpeg_start").Inc()
			logSourceIssueThrottled(&lastUpstreamLog, log, "ffmpeg start (source not ready or invalid URL)", "err", err)
			cancel()
			sleepBackoff(ctx, backoff)
			continue
		}
		sc := capture.NewScanner(pipe)
		for {
			frame, err := sc.ReadFrameCtx(subCtx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					break
				}
				if errors.Is(err, io.EOF) {
					break
				}
				metrics.OperationErrors.WithLabelValues("frame_read").Inc()
				logSourceIssueThrottled(&lastUpstreamLog, log, "frame read (stream interrupted or paused)", "err", err)
				break
			}
			n := frameNo.Add(1)
			now := time.Now().UTC()
			day := now.Format("2006-01-02")
			ts := now.UnixNano()
			key := fmt.Sprintf("%s/%s/%s/frame_%d.png", prefix, day, cam.CameraID, ts)

			pngBytes, err := s3store.JPEGBytesToPNG(frame)
			if err != nil {
				metrics.OperationErrors.WithLabelValues("s3_put").Inc()
				log.Error("jpeg to png", "err", err)
				continue
			}
			if err := store.PutPNG(ctx, key, pngBytes); err != nil {
				metrics.OperationErrors.WithLabelValues("s3_put").Inc()
				log.Error("s3 put", "key", key, "err", err)
			}

			meta := domain.ProcessMeta{
				SegmentID:  cam.SegmentID,
				CameraID:   cam.CameraID,
				S3Key:      key,
				ObservedAt: now.Format(time.RFC3339Nano),
			}
			if err := mlc.PostProcess(ctx, frame, "frame.jpg", meta); err != nil {
				metrics.OperationErrors.WithLabelValues("ml_process").Inc()
				log.Warn("ml process", "err", err)
			}

			if n%frameLogEveryN == 0 {
				log.Info("frames", "count", n, "last_key", key)
			}
		}
		_ = pipe.Close()
		cancel()
		if ctx.Err() != nil {
			return
		}
		logSourceIssueThrottled(&lastUpstreamLog, log, "source stopped delivering frames, reconnecting", "rtsp_url", cam.RTSPURL)
		sleepBackoff(ctx, backoff)
	}
}
