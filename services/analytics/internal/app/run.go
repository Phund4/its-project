package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	"traffic-analytics/internal/adapters/grpcmap"
)

// Run инициализирует зависимости, gRPC для map_portal и HTTP до завершения rootCtx.
func Run(rootCtx context.Context) error {
	deps, err := InitializeDependencies(rootCtx)
	if err != nil {
		return err
	}
	defer func() {
		if err := deps.Close(); err != nil {
			slog.Warn("deps close", "err", err)
		}
	}()

	lis, err := net.Listen("tcp", deps.Config.MapGRPCListenAddr)
	if err != nil {
		return fmt.Errorf("map grpc listen: %w", err)
	}
	grpcSrv := grpc.NewServer()
	grpcmap.Register(grpcSrv, deps.Store, deps.PortalHub, deps.Config)
	go func() {
		slog.Info("analytics map grpc", "listen", deps.Config.MapGRPCListenAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Error("map grpc serve", "err", err)
		}
	}()
	defer grpcSrv.GracefulStop()

	return RunHTTPServer(rootCtx, deps)
}
