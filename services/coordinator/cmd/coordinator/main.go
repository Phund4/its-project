package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpserver "traffic-coordinator/internal/adapters/http"
	"traffic-coordinator/internal/app"
	"traffic-coordinator/internal/config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	a := app.New(cfg.Sources, time.Duration(cfg.HeartbeatTimeoutSec)*time.Second)
	if err := httpserver.Run(ctx, cfg.ListenAddr, a); err != nil {
		slog.Error("run", "err", err)
		os.Exit(1)
	}
}
