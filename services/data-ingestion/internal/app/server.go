package app

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RunMetricsServer поднимает только HTTP /metrics; блокируется до отмены ctx, затем делает Shutdown.
func RunMetricsServer(ctx context.Context, listenAddr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: metricsReadHeaderTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("data_ingestion shutdown signal, stopping metrics server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), metricsShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("metrics server shutdown", "err", err)
	}
	return nil
}
