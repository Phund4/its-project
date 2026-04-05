package app

import (
	"context"
	"log/slog"
	"sync"
)

// Run инициализирует зависимости, /metrics, опционально воркеры камер и gRPC телеметрии.
func Run(rootCtx context.Context) error {
	deps, err := InitializeDependencies(rootCtx)
	if err != nil {
		return err
	}

	logArgs := []any{
		"config", deps.Config.ConfigFile,
		"cameras_enabled", deps.Features.CamerasEnabled,
		"telemetry_grpc", deps.Features.TelemetryGRPC,
		"metrics", deps.Config.Metrics.ListenAddr,
	}
	if deps.Features.CamerasEnabled {
		logArgs = append(logArgs, "rtsp_sources", len(deps.Config.Cameras))
	}
	if deps.Features.TelemetryGRPC {
		logArgs = append(logArgs, "telemetry_grpc_listen", deps.TelemetryListenAddr)
	}
	slog.Info("data_ingestion starting", logArgs...)

	var wg sync.WaitGroup
	if deps.Features.CamerasEnabled {
		StartCameraWorkers(rootCtx, deps, &wg)
	}

	srvDone := make(chan struct{})
	go func() {
		defer close(srvDone)
		if err := RunMetricsServer(rootCtx, deps.Config.Metrics.ListenAddr); err != nil {
			slog.Error("metrics server", "err", err)
		}
	}()

	grpcDone := make(chan struct{})
	if deps.Features.TelemetryGRPC && deps.TelemetryGRPC != nil {
		go func() {
			defer close(grpcDone)
			if err := RunTelemetryGRPCServer(rootCtx, deps.TelemetryListenAddr, deps.TelemetryGRPC); err != nil {
				slog.Error("telemetry grpc", "err", err)
			}
		}()
	} else {
		close(grpcDone)
	}

	<-rootCtx.Done()
	WaitWorkers(&wg)
	<-srvDone
	<-grpcDone
	slog.Info("data_ingestion stopped")
	return nil
}
