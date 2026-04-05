// Program ingest — сервис захвата кадров с RTSP, загрузки в S3 и вызова ML.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"data-ingestion/internal/app"
	apperrors "data-ingestion/internal/errors"
)

// main настраивает логирование, контекст сигналов и делегирует жизненный цикл в app.Run.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(rootCtx); err != nil {
		if errors.Is(err, apperrors.ErrMissingAWSCredentials) {
			slog.Error("set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY (e.g. minioadmin)")
			os.Exit(1)
		}
		slog.Error("run", "err", err)
		os.Exit(1)
	}
}
