// Package app собирает зависимости и жизненный цикл процесса data_ingestion.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"data-ingestion/internal/adapters/ml"
	"data-ingestion/internal/adapters/s3"
	"data-ingestion/internal/config"
	"data-ingestion/internal/constants"
	"data-ingestion/internal/core/services"
	apperrors "data-ingestion/internal/errors"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Run загружает конфиг, поднимает /metrics, S3, ML-клиент и воркеры камер до отмены rootCtx.
func Run(rootCtx context.Context) error {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	config.ApplyEnvOverrides(cfg)
	slog.Info("data_ingestion starting", "cameras", len(cfg.Cameras), "metrics", cfg.Metrics.ListenAddr, "s3", cfg.S3.Bucket)

	ak := os.Getenv("AWS_ACCESS_KEY_ID")
	sk := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if ak == "" || sk == "" {
		return apperrors.ErrMissingAWSCredentials
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	readHdr := time.Duration(constants.MetricsReadHeaderTimeoutSec) * time.Second
	srv := &http.Server{
		Addr:              cfg.Metrics.ListenAddr,
		Handler:           mux,
		BaseContext:       func(net.Listener) context.Context { return rootCtx },
		ReadHeaderTimeout: readHdr,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server", "err", err)
		}
	}()

	ctx := rootCtx
	store, err := s3store.New(ctx, cfg.S3.Endpoint, cfg.S3.Region, cfg.S3.Bucket, ak, sk)
	if err != nil {
		return fmt.Errorf("s3 client: %w", err)
	}
	if cfg.Ingest.CreateBucketIfMissing {
		if err := store.EnsureBucket(ctx); err != nil {
			return fmt.Errorf("ensure bucket: %w", err)
		}
		slog.Info("bucket ok", "bucket", cfg.S3.Bucket)
	}

	mlc := mlclient.New(
		cfg.ML.BaseURL,
		cfg.ML.ProcessPath,
		time.Duration(cfg.ML.TimeoutSeconds)*time.Second,
	)

	var wg sync.WaitGroup
	for _, cam := range cfg.Cameras {
		cam := cam
		wg.Add(1)
		go func() {
			defer wg.Done()
			services.RunCamera(ctx, cam, store, mlc, cfg.S3.Prefix, cfg.Ingest.FFmpegPath, cfg.Ingest.TargetFPS)
		}()
	}

	<-ctx.Done()
	slog.Info("data_ingestion shutdown signal, stopping metrics server")
	shutdownTO := time.Duration(constants.MetricsShutdownTimeoutSec) * time.Second
	shutdownMetrics, cancelMetrics := context.WithTimeout(context.Background(), shutdownTO)
	defer cancelMetrics()
	if err := srv.Shutdown(shutdownMetrics); err != nil {
		slog.Warn("metrics server shutdown", "err", err)
	}
	slog.Info("waiting for camera workers")
	wg.Wait()
	slog.Info("data_ingestion stopped")
	return nil
}
