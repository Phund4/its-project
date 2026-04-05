package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	httpx "traffic-analytics/internal/adapters/http"
)

// RunHTTPServer поднимает HTTP-сервер с маршрутами до отмены rootCtx или ошибки Listen.
func RunHTTPServer(rootCtx context.Context, deps *Deps) error {
	mux := http.NewServeMux()
	httpx.Register(mux, deps.Ingest)

	srv := &http.Server{
		Addr:              deps.Config.ListenAddr,
		Handler:           mux,
		BaseContext:       func(net.Listener) context.Context { return rootCtx },
		ReadHeaderTimeout: httpReadHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("analytics starting",
			"listen", deps.Config.ListenAddr,
			"clickhouse", deps.CHAddr,
			"congestion_persist_interval", deps.Config.CongestionPersistInterval.String(),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("%w: %v", ErrHTTPListen, err)
	case <-rootCtx.Done():
	}

	slog.Info("analytics shutdown signal received")
	shCtx, cancel := context.WithTimeout(context.Background(), httpServerShutdown)
	defer cancel()
	if err := srv.Shutdown(shCtx); err != nil {
		slog.Warn("graceful shutdown", "err", err)
	}
	slog.Info("analytics stopped")
	return nil
}
