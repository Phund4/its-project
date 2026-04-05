// Package app запускает HTTP-сервер analytics и ClickHouse-адаптер.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	chstore "traffic-analytics/internal/adapters/ch"
	httpadapter "traffic-analytics/internal/adapters/http"
	"traffic-analytics/internal/config"
	"traffic-analytics/internal/constants"
	"traffic-analytics/internal/core/services"
	apperrors "traffic-analytics/internal/errors"
	"traffic-analytics/internal/utils"
)

// Run поднимает store, маршруты и блокируется до отмены rootCtx или ошибки Listen.
func Run(rootCtx context.Context) error {
	envPath := os.Getenv("ENV_FILE")
	if envPath == "" {
		envPath = ".env"
	}
	if err := utils.LoadDotEnv(envPath); err != nil {
		slog.Warn("env file", "path", envPath, "err", err)
	}
	cfg := config.Load()

	chAddr := chstore.FirstAddr(cfg.ClickHouseAddr)
	store, err := chstore.New(rootCtx, chAddr, cfg.ClickHouseDatabase, cfg.ClickHouseUser, cfg.ClickHousePassword, cfg.IncidentsTable, cfg.CongestionTable)
	if err != nil {
		return fmt.Errorf("clickhouse: %w", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			slog.Warn("clickhouse close", "err", err)
		}
	}()

	svc := services.NewIngestService(store, cfg, rootCtx)
	mux := http.NewServeMux()
	httpadapter.Register(mux, svc)

	readHdr := time.Duration(constants.HTTPReadHeaderTimeoutSec) * time.Second
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		BaseContext:       func(net.Listener) context.Context { return rootCtx },
		ReadHeaderTimeout: readHdr,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("analytics starting",
			"listen", cfg.ListenAddr,
			"clickhouse", chAddr,
			"congestion_persist_interval", cfg.CongestionPersistInterval.String(),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("%w: %v", apperrors.ErrHTTPListen, err)
	case <-rootCtx.Done():
	}

	slog.Info("analytics shutdown signal received")
	shSec := time.Duration(constants.HTTPServerShutdownSec) * time.Second
	shCtx, cancel := context.WithTimeout(context.Background(), shSec)
	defer cancel()
	if err := srv.Shutdown(shCtx); err != nil {
		slog.Warn("graceful shutdown", "err", err)
	}
	slog.Info("analytics stopped")
	return nil
}
