package utils

import (
	"context"
	"time"
)

// SleepBackoff ждёт d или отмену ctx.
func SleepBackoff(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
