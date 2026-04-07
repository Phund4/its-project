package telemetrygrpc

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	busv1 "data-ingestion/api/bus/v1"
	"data-ingestion/internal/adapters/analytics"
	"data-ingestion/internal/adapters/metrics"
	"data-ingestion/internal/adapters/telemetry"
)

// Server реализует BusTelemetryService: пересылает в analytics.
type Server struct {
	// UnimplementedBusTelemetryServiceServer заглушки gRPC для совместимости.
	busv1.UnimplementedBusTelemetryServiceServer

	// Publisher HTTP POST или Kafka в топик телеметрии.
	Publisher telemetry.Publisher

	// AllowedMunicipalities ограничивает приём по списку муниципалитетов; nil или пустая карта — любая телеметрия (как в каталоге vehicle_bus_telemetry).
	AllowedMunicipalities map[string]struct{}
}

// SendBusTelemetry маппит proto → JSON telemetry и доставляет в analytics (HTTP или Kafka).
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
	municipalityID := strings.TrimSpace(in.GetMunicipalityId())
	if len(s.AllowedMunicipalities) > 0 {
		if municipalityID == "" {
			slog.Warn("telemetry dropped: empty municipality_id, coordinator filter is enabled")
			return &busv1.SendBusTelemetryResponse{}, nil
		}
		if _, ok := s.AllowedMunicipalities[municipalityID]; !ok {
			slog.Warn("telemetry dropped: municipality not assigned", "municipality_id", municipalityID)
			return &busv1.SendBusTelemetryResponse{}, nil
		}
	}
	tel := map[string]any{
		"vehicle_id":          in.GetVehicleId(),
		"route_id":            in.GetRouteId(),
		"lat":                 in.GetLat(),
		"lon":                 in.GetLon(),
		"speed_kmh":           in.GetSpeedKmh(),
		"heading_deg":         in.GetHeadingDeg(),
		"observed_at_rfc3339": in.GetObservedAtRfc3339(),
		"municipality_id":     municipalityID,
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
	payload, err := json.Marshal(body)
	if err != nil {
		slog.Warn("telemetry marshal ingest body", "err", err)
		return &busv1.SendBusTelemetryResponse{}, nil
	}
	if err := s.Publisher.PublishIngestJSON(ctx, payload); err != nil {
		metrics.OperationErrors.WithLabelValues("telemetry_forward_analytics").Inc()
		slog.Warn("forward telemetry to analytics", "err", err)
		return nil, err
	}
	metrics.TelemetryForwarded.Inc()
	return &busv1.SendBusTelemetryResponse{}, nil
}
