package utils

import (
	"log/slog"
	"time"

	"data-ingestion/internal/constants"
)

// LogSourceIssueThrottled пишет предупреждение не чаще одного раза в SourceWaitLogIntervalSec.
func LogSourceIssueThrottled(last *time.Time, log *slog.Logger, msg string, args ...any) {
	interval := time.Duration(constants.SourceWaitLogIntervalSec) * time.Second
	if time.Since(*last) < interval {
		return
	}
	*last = time.Now()
	log.Warn(msg, args...)
}
