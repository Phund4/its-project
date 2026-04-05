// Package app запускает HTTP ml_gateway и Forwarder.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	httpadapter "ml-gateway/internal/adapters/http"
	"ml-gateway/internal/config"
	"ml-gateway/internal/constants"
	"ml-gateway/internal/core/services"
	apperrors "ml-gateway/internal/errors"
	"ml-gateway/internal/utils"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Run поднимает HTTP-сервер до отмены rootCtx или фатальной ошибки Listen.
func Run(rootCtx context.Context) error {
	envPath := os.Getenv("ENV_FILE")
	if envPath == "" {
		envPath = ".env"
	}
	if err := utils.LoadDotEnv(envPath); err != nil {
		slog.Warn("env file", "path", envPath, "err", err)
	}
	cfg := config.Load()

	client := &http.Client{Timeout: cfg.AnalyticsTimeout}
	fwd := services.NewForwarder(cfg, client)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	httpadapter.Register(mux, fwd, rootCtx)

	readHdr := time.Duration(constants.HTTPReadHeaderTimeoutSec) * time.Second
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		BaseContext:       func(net.Listener) context.Context { return rootCtx },
		ReadHeaderTimeout: readHdr,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("ml_gateway starting", "listen", cfg.ListenAddr, "analytics", cfg.AnalyticsBaseURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("%w: %v", apperrors.ErrHTTPListen, err)
	case <-rootCtx.Done():
	}

	slog.Info("ml_gateway shutdown signal received")
	shSec := time.Duration(constants.HTTPServerShutdownSec) * time.Second
	shCtx, cancel := context.WithTimeout(context.Background(), shSec)
	defer cancel()
	if err := srv.Shutdown(shCtx); err != nil {
		slog.Warn("graceful shutdown", "err", err)
	}
	slog.Info("ml_gateway stopped")
	return nil
}
