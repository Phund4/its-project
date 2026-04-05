// Program map_portal — веб-страница с интерактивной картой (Leaflet + OSM).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"map-portal/internal/app"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(rootCtx); err != nil {
		slog.Error("run", "err", err)
		os.Exit(1)
	}
}
