package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	busv1 "data-ingestion/api/bus/v1"
	"data-ingestion/internal/adapters/telemetrygrpc"

	"google.golang.org/grpc"
)

// RunTelemetryGRPCServer поднимает gRPC BusTelemetryService; блокируется до отмены ctx.
func RunTelemetryGRPCServer(ctx context.Context, listenAddr string, srv *telemetrygrpc.Server) error {
	if srv == nil {
		return nil
	}
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("telemetry grpc listen: %w", err)
	}
	grpcSrv := grpc.NewServer()
	busv1.RegisterBusTelemetryServiceServer(grpcSrv, srv)

	errCh := make(chan error, 1)
	go func() {
		errCh <- grpcSrv.Serve(ln)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("telemetry grpc serve: %w", err)
		}
	case <-ctx.Done():
		slog.Info("telemetry grpc shutting down")
		grpcSrv.GracefulStop()
		if err := <-errCh; err != nil {
			slog.Warn("telemetry grpc stopped", "err", err)
		}
	}
	return nil
}
