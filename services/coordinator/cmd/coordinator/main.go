package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	pgstore "traffic-coordinator/internal/adapters/postgres"
	"traffic-coordinator/internal/app"
	"traffic-coordinator/internal/config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	var store app.Store
	switch cfg.Coordinator.Store {
	case "postgres":
		pgs, err := pgstore.New(
			ctx,
			cfg.Postgres.URL,
			cfg.Postgres.MaxOpenConns,
			cfg.Postgres.MaxIdleConns,
			cfg.Postgres.ConnMaxIdleTime,
		)
		if err != nil {
			slog.Error("postgres connect", "err", err)
			os.Exit(1)
		}
		defer pgs.Close()
		store = pgs
		slog.Info("coordinator store", "backend", "postgres")
	case "memory":
		slog.Error("coordinator store", "err", "memory store is temporarily disabled")
		os.Exit(1)
	default:
		slog.Error("coordinator store", "err", "unknown coordinator store")
		os.Exit(1)
	}

	a := app.New(store, time.Duration(cfg.Coordinator.WorkerStatusTimeoutSec)*time.Second)
	if err := a.Run(ctx, cfg.Server); err != nil {
		slog.Error("run", "err", err)
		os.Exit(1)
	}
}
