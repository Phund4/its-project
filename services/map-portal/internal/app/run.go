package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	mapv1 "traffic-analytics/api/map/v1"

	"map-portal/internal/config"
	"map-portal/internal/httpapi"
	webassets "map-portal/web"
)

// Run поднимает HTTP: статика и REST, данные — gRPC в analytics.
func Run(rootCtx context.Context) error {
	cfg := config.Load()

	conn, err := grpc.NewClient(cfg.AnalyticsGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("analytics grpc dial: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Warn("analytics grpc close", "err", err)
		}
	}()
	mapCli := mapv1.NewMapPortalClient(conn)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	httpapi.Register(mux, mapCli)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		b, err := webassets.Files.ReadFile("index.html")
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		BaseContext:       func(net.Listener) context.Context { return rootCtx },
		ReadHeaderTimeout: httpReadHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("map_portal starting",
			"listen", cfg.ListenAddr,
			"analytics_grpc", cfg.AnalyticsGRPCAddr,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http listen: %w", err)
	case <-rootCtx.Done():
	}

	slog.Info("map_portal shutdown signal received")
	shCtx, cancel := context.WithTimeout(context.Background(), httpServerShutdown)
	defer cancel()
	if err := srv.Shutdown(shCtx); err != nil {
		slog.Warn("graceful shutdown", "err", err)
	}
	slog.Info("map_portal stopped")
	return nil
}
