package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	httpserver "traffic-coordinator/internal/adapters/http"
	"traffic-coordinator/internal/app"
	"traffic-coordinator/internal/config"
	memstore "traffic-coordinator/internal/storage/memory"
	pgstore "traffic-coordinator/internal/storage/postgres"
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
	var store app.Store
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		store = memstore.New(cfg.Sources, cfg.ZoneWorkers)
		slog.Info("coordinator store", "backend", "memory")
	} else {
		pgs, err := pgstore.New(ctx, cfg.DatabaseURL)
		if err != nil {
			slog.Error("postgres connect", "err", err)
			os.Exit(1)
		}
		defer pgs.Close()
		store = pgs
		slog.Info("coordinator store", "backend", "postgres")
	}
	a := app.New(store, time.Duration(cfg.HeartbeatTimeoutSec)*time.Second)
	if err := httpserver.Run(ctx, cfg.ListenAddr, a); err != nil {
		slog.Error("run", "err", err)
		os.Exit(1)
	}
}
