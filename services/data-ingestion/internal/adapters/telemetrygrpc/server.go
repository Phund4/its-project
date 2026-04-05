package telemetrygrpc

import (
	"context"
	"encoding/json"
	"log/slog"

	busv1 "data-ingestion/api/bus/v1"
	"data-ingestion/internal/adapters/analytics"
	"data-ingestion/internal/adapters/metrics"
)

// Server реализует BusTelemetryService: пересылает в analytics.
type Server struct {
	// UnimplementedBusTelemetryServiceServer заглушки gRPC для совместимости.
	busv1.UnimplementedBusTelemetryServiceServer

	// Analytics HTTP-клиент в analytics ingest.
	Analytics *analytics.Client
}

// SendBusTelemetry маппит proto → JSON telemetry и POST в analytics.
func (s *Server) SendBusTelemetry(ctx context.Context, in *busv1.BusTelemetry) (*busv1.SendBusTelemetryResponse, error) {
	if in == nil {
		return &busv1.SendBusTelemetryResponse{}, nil
	}
	seg := in.GetSegmentId()
	if seg == "" {
		seg = "unknown-segment"
	}
	cam := in.GetVehicleId()
	if cam == "" {
		cam = "unknown-vehicle"
	}
	at := in.GetObservedAtRfc3339()
	if at == "" {
		at = "1970-01-01T00:00:00Z"
	}
	tel := map[string]any{
		"vehicle_id":          in.GetVehicleId(),
		"route_id":            in.GetRouteId(),
		"lat":                 in.GetLat(),
		"lon":                 in.GetLon(),
		"speed_kmh":           in.GetSpeedKmh(),
		"heading_deg":         in.GetHeadingDeg(),
		"observed_at_rfc3339": in.GetObservedAtRfc3339(),
		"municipality_id":     in.GetMunicipalityId(),
	}
	raw, err := json.Marshal(tel)
	if err != nil {
		slog.Warn("telemetry marshal", "err", err)
		return &busv1.SendBusTelemetryResponse{}, nil
	}
	body := analytics.IngestBody{
		SegmentID:  seg,
		CameraID:   cam,
		ObservedAt: at,
		S3Key:      "",
		Telemetry:  raw,
	}
	if err := s.Analytics.PostIngest(ctx, body); err != nil {
		metrics.OperationErrors.WithLabelValues("telemetry_forward_analytics").Inc()
		slog.Warn("forward telemetry to analytics", "err", err)
		return nil, err
	}
	metrics.TelemetryForwarded.Inc()
	return &busv1.SendBusTelemetryResponse{}, nil
}
