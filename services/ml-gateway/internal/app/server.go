package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	httpx "ml-gateway/internal/adapters/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RunHTTPServer поднимает /metrics и API до отмены rootCtx или ошибки Listen.
func RunHTTPServer(rootCtx context.Context, deps *Deps) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	httpx.Register(mux, deps.Forwarder, rootCtx)

	srv := &http.Server{
		Addr:              deps.Config.ListenAddr,
		Handler:           mux,
		BaseContext:       func(net.Listener) context.Context { return rootCtx },
		ReadHeaderTimeout: httpReadHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("ml_gateway starting", "listen", deps.Config.ListenAddr, "analytics", deps.Config.AnalyticsBaseURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("%w: %v", ErrHTTPListen, err)
	case <-rootCtx.Done():
	}

	slog.Info("ml_gateway shutdown signal received")
	shCtx, cancel := context.WithTimeout(context.Background(), httpServerShutdown)
	defer cancel()
	if err := srv.Shutdown(shCtx); err != nil {
		slog.Warn("graceful shutdown", "err", err)
	}
	slog.Info("ml_gateway stopped")
	return nil
}
