// Program analytics — HTTP-приём событий дороги, метрики Prometheus и запись в ClickHouse.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"traffic-analytics/internal/app"
)

// main настраивает логирование, контекст ОС и передаёт управление app.Run.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(rootCtx); err != nil {
		slog.Error("run", "err", err)
		os.Exit(1)
	}
}
