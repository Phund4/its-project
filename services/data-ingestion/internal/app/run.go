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
	// Bootstrap-heartbeat: без него coordinator не считает инстанс "живым",
	// и при холодном старте может вернуть пустые назначения.
	if err := deps.Coordinator.SendHeartbeat(rootCtx, zoneID, clusterID, instanceID, 0); err != nil {
		slog.Warn("coordinator bootstrap heartbeat failed", "err", err)
	}

	cameras, err := deps.Coordinator.FetchCameraAssignments(rootCtx, zoneID, clusterID, instanceID)
	if err != nil {
		slog.Warn("coordinator camera assignments unavailable, starting in standby", "err", err)
		cameras = nil
	}
	hasBusTelemetry, err := deps.Coordinator.HasVehicleBusTelemetryAssignment(rootCtx, zoneID, clusterID, instanceID)
	if err != nil {
		slog.Warn("coordinator telemetry assignments unavailable, starting in standby", "err", err)
		hasBusTelemetry = false
	}
	if len(cameras) == 0 && !hasBusTelemetry {
		slog.Info("no assignments yet, data-ingestion is running in standby", "zone", zoneID, "cluster", clusterID, "instance", instanceID)
	}
	if len(cameras) > 0 {
		if err := InitVideoPipeline(rootCtx, deps); err != nil {
			return err
		}
		logArgs = append(logArgs, "rtsp_sources", len(cameras))
		slog.Info("coordinator camera assignments applied", "assigned_sources", len(cameras), "zone", zoneID, "cluster", clusterID, "instance", instanceID)
	}
	if hasBusTelemetry {
		if err := InitTelemetryPipeline(deps); err != nil {
			return err
		}
		deps.TelemetryGRPC.AllowedMunicipalities = nil
		deps.TelemetryHTTP.AllowedMunicipalities = nil
		logArgs = append(
			logArgs,
			"telemetry_grpc", true,
			"telemetry_grpc_listen", deps.TelemetryListenAddr,
			"telemetry_http", true,
			"telemetry_http_listen", deps.TelemetryHTTPListenAddr,
			"data_class", config.DataClassVehicleBusTelemetry,
		)
		slog.Info("coordinator telemetry assignments applied", "zone", zoneID, "cluster", clusterID, "instance", instanceID)
	}
	assignmentCount := len(cameras)
	if hasBusTelemetry {
		assignmentCount++
	}
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
