// Program gateway — HTTP-шлюз: приём событий от ML и пересылка в analytics.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"ml-gateway/internal/app"
)

// main настраивает лог и контекст сигналов; жизненный цикл в app.Run.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(rootCtx); err != nil {
		slog.Error("run", "err", err)
		os.Exit(1)
	}
}
