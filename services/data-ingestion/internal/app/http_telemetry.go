package app

import (
	"context"
	"fmt"

	"data-ingestion/internal/adapters/telemetryhttp"
)

// RunTelemetryHTTPServer поднимает HTTP вход телеметрии; блокируется до отмены ctx.
func RunTelemetryHTTPServer(ctx context.Context, listenAddr string, srv *telemetryhttp.Server) error {
	if srv == nil {
		return nil
	}
	if err := telemetryhttp.Run(ctx, listenAddr, srv); err != nil {
		return fmt.Errorf("telemetry http serve: %w", err)
	}
	return nil
}
