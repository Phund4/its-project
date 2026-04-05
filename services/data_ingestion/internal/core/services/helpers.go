package services

import (
	"context"
	"log/slog"
	"time"
)

func logSourceIssueThrottled(last *time.Time, log *slog.Logger, msg string, args ...any) {
	interval := time.Duration(sourceWaitLogIntervalSec) * time.Second
	if time.Since(*last) < interval {
		return
	}
	*last = time.Now()
	log.Warn(msg, args...)
}

func sleepBackoff(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
