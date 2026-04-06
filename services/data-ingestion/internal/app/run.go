package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"data-ingestion/internal/config"
)

// Run инициализирует зависимости, /metrics, опционально воркеры камер и gRPC телеметрии.
func Run(rootCtx context.Context) error {
	deps, err := InitializeDependencies(rootCtx)
	if err != nil {
		return err
	}

	logArgs := []any{
		"config", deps.Config.ConfigFile,
		"metrics", deps.Config.Metrics.ListenAddr,
	}

	zoneID := config.CoordinatorZoneIDFromEnv()
	clusterID := config.CoordinatorClusterIDFromEnv()
	instanceID := config.CoordinatorInstanceIDFromEnv()
	if deps.Coordinator == nil {
		return fmt.Errorf("set COORDINATOR_BASE_URL")
	}
	if zoneID == "" || clusterID == "" || instanceID == "" {
		return fmt.Errorf("set COORDINATOR_ZONE_ID, COORDINATOR_CLUSTER_ID, COORDINATOR_INSTANCE_ID")
	}

	cameras, err := deps.Coordinator.FetchCameraAssignments(rootCtx, zoneID, clusterID, instanceID)
	if err != nil {
		return fmt.Errorf("coordinator camera assignments: %w", err)
	}
	telemetryMunicipalities, err := deps.Coordinator.FetchTelemetryMunicipalities(rootCtx, zoneID, clusterID, instanceID)
	if err != nil {
		return fmt.Errorf("coordinator telemetry assignments: %w", err)
	}
	if len(cameras) == 0 && len(telemetryMunicipalities) == 0 {
		return fmt.Errorf("coordinator returned no assignments for zone=%s cluster=%s instance=%s", zoneID, clusterID, instanceID)
	}
	if len(cameras) > 0 {
		if err := InitVideoPipeline(rootCtx, deps); err != nil {
			return err
		}
		logArgs = append(logArgs, "rtsp_sources", len(cameras))
		slog.Info("coordinator camera assignments applied", "assigned_sources", len(cameras), "zone", zoneID, "cluster", clusterID, "instance", instanceID)
	}
	if len(telemetryMunicipalities) > 0 {
		if err := InitTelemetryPipeline(deps); err != nil {
			return err
		}
		allow := make(map[string]struct{}, len(telemetryMunicipalities))
		for _, m := range telemetryMunicipalities {
			allow[m] = struct{}{}
		}
		deps.TelemetryGRPC.AllowedMunicipalities = allow
		deps.TelemetryHTTP.AllowedMunicipalities = allow
		logArgs = append(
			logArgs,
			"telemetry_grpc", true,
			"telemetry_grpc_listen", deps.TelemetryListenAddr,
			"telemetry_http", true,
			"telemetry_http_listen", deps.TelemetryHTTPListenAddr,
			"telemetry_municipalities", len(telemetryMunicipalities),
		)
		slog.Info("coordinator telemetry assignments applied", "assigned_municipalities", len(telemetryMunicipalities), "zone", zoneID, "cluster", clusterID, "instance", instanceID)
	}
	assignmentCount := len(cameras) + len(telemetryMunicipalities)
	if err := deps.Coordinator.SendHeartbeat(rootCtx, zoneID, clusterID, instanceID, assignmentCount); err != nil {
		slog.Warn("coordinator heartbeat failed", "err", err)
	}
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-rootCtx.Done():
				return
			case <-t.C:
				if err := deps.Coordinator.SendHeartbeat(rootCtx, zoneID, clusterID, instanceID, assignmentCount); err != nil {
					slog.Warn("coordinator heartbeat failed", "err", err)
				}
			}
		}
	}()
	slog.Info("data_ingestion starting", logArgs...)

	var wg sync.WaitGroup
	if len(cameras) > 0 {
		StartCameraWorkersWithCameras(rootCtx, deps, cameras, &wg)
	}

	srvDone := make(chan struct{})
	go func() {
		defer close(srvDone)
		if err := RunMetricsServer(rootCtx, deps.Config.Metrics.ListenAddr); err != nil {
			slog.Error("metrics server", "err", err)
		}
	}()

	grpcDone := make(chan struct{})
	if deps.TelemetryGRPC != nil {
		go func() {
			defer close(grpcDone)
			if err := RunTelemetryGRPCServer(rootCtx, deps.TelemetryListenAddr, deps.TelemetryGRPC); err != nil {
				slog.Error("telemetry grpc", "err", err)
			}
		}()
	} else {
		close(grpcDone)
	}

	httpTelemetryDone := make(chan struct{})
	if deps.TelemetryHTTP != nil {
		go func() {
			defer close(httpTelemetryDone)
			if err := RunTelemetryHTTPServer(rootCtx, deps.TelemetryHTTPListenAddr, deps.TelemetryHTTP); err != nil {
				slog.Error("telemetry http", "err", err)
			}
		}()
	} else {
		close(httpTelemetryDone)
	}

	<-rootCtx.Done()
	WaitWorkers(&wg)
	<-srvDone
	<-grpcDone
	<-httpTelemetryDone
	slog.Info("data_ingestion stopped")
	return nil
}
