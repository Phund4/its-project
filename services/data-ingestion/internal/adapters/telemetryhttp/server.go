package telemetryhttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"data-ingestion/internal/adapters/analytics"
	"data-ingestion/internal/adapters/metrics"
	"data-ingestion/internal/adapters/telemetry"
)

// Server принимает телеметрию по HTTP и пересылает в analytics/Kafka.
type Server struct {
	Publisher telemetry.Publisher

	// AllowedMunicipalities ограничивает приём по списку муниципалитетов; nil или пустая карта — любая телеметрия (vehicle_bus_telemetry в coordinator).
	AllowedMunicipalities map[string]struct{}
}

type telemetryInput struct {
	SegmentID     string  `json:"segment_id"`
	VehicleID     string  `json:"vehicle_id"`
	RouteID       string  `json:"route_id"`
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	SpeedKmh      float64 `json:"speed_kmh"`
	HeadingDeg    float64 `json:"heading_deg"`
	ObservedAt    string  `json:"observed_at_rfc3339"`
	MunicipalityID string `json:"municipality_id"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /v1/telemetry", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if s.Publisher == nil {
			http.Error(w, "publisher is not configured", http.StatusServiceUnavailable)
			return
		}
		var in telemetryInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		municipalityID := strings.TrimSpace(in.MunicipalityID)
		if len(s.AllowedMunicipalities) > 0 {
			if municipalityID == "" {
				http.Error(w, "municipality_id is required", http.StatusBadRequest)
				return
			}
			if _, ok := s.AllowedMunicipalities[municipalityID]; !ok {
				http.Error(w, "municipality is not assigned", http.StatusForbidden)
				return
			}
		}
		seg := strings.TrimSpace(in.SegmentID)
		if seg == "" {
			seg = "unknown-segment"
		}
		vehicleID := strings.TrimSpace(in.VehicleID)
		if vehicleID == "" {
			vehicleID = "unknown-vehicle"
		}
		at := strings.TrimSpace(in.ObservedAt)
		if at == "" {
			at = time.Now().UTC().Format(time.RFC3339Nano)
		}

		tel := map[string]any{
			"vehicle_id":          vehicleID,
			"route_id":            strings.TrimSpace(in.RouteID),
			"lat":                 in.Lat,
			"lon":                 in.Lon,
			"speed_kmh":           in.SpeedKmh,
			"heading_deg":         in.HeadingDeg,
			"observed_at_rfc3339": at,
			"municipality_id":     municipalityID,
		}
		raw, err := json.Marshal(tel)
		if err != nil {
			http.Error(w, "marshal telemetry failed", http.StatusInternalServerError)
			return
		}
		body := analytics.IngestBody{
			SegmentID:  seg,
			CameraID:   vehicleID,
			ObservedAt: at,
			Telemetry:  raw,
		}
		payload, err := json.Marshal(body)
		if err != nil {
			http.Error(w, "marshal ingest failed", http.StatusInternalServerError)
			return
		}
		if err := s.Publisher.PublishIngestJSON(r.Context(), payload); err != nil {
			metrics.OperationErrors.WithLabelValues("telemetry_forward_analytics").Inc()
			slog.Warn("forward telemetry (http) to analytics", "err", err)
			http.Error(w, "forward failed", http.StatusBadGateway)
			return
		}
		metrics.TelemetryForwarded.Inc()
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

func Run(ctx context.Context, listenAddr string, srv *Server) error {
	if srv == nil {
		return nil
	}
	httpSrv := &http.Server{
		Addr:              listenAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- httpSrv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shCtx)
	}
	return nil
}
