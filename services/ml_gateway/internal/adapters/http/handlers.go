// Package httpadapter — маршруты ml_gateway.
package httpadapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"ml-gateway/internal/adapters/metrics"
	"ml-gateway/internal/constants"
	"ml-gateway/internal/core/domain"
	"ml-gateway/internal/core/services"
	"ml-gateway/internal/utils"
)

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func roadEventsHandler(fwd *services.Forwarder, appCtx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if fwd.AnalyticsURL() == "" {
			metrics.OperationErrors.WithLabelValues("forward_analytics").Inc()
			http.Error(w, "analytics not configured (set ANALYTICS_BASE_URL)", http.StatusServiceUnavailable)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, constants.MaxRoadEventBodyBytes))
		if err != nil {
			metrics.OperationErrors.WithLabelValues("post_decode").Inc()
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}

		var ev domain.RoadEvent
		if err := json.Unmarshal(body, &ev); err != nil {
			metrics.OperationErrors.WithLabelValues("post_decode").Inc()
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		ev.SegmentID = strings.TrimSpace(ev.SegmentID)
		ev.CameraID = strings.TrimSpace(ev.CameraID)
		ev.ObservedAt = strings.TrimSpace(ev.ObservedAt)
		ev.S3Key = strings.TrimSpace(ev.S3Key)
		if ev.SegmentID == "" || ev.CameraID == "" || ev.ObservedAt == "" {
			metrics.OperationErrors.WithLabelValues("post_validate").Inc()
			http.Error(w, "segment_id, camera_id, observed_at required", http.StatusBadRequest)
			return
		}

		fwdCtx, fwdCancel := utils.WithAppShutdown(r.Context(), appCtx)
		defer fwdCancel()
		if err := fwd.Forward(fwdCtx, body); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// Register вешает POST /v1/road-events, GET /health и монтирует /metrics снаружи (в app.Run).
func Register(mux *http.ServeMux, fwd *services.Forwarder, appCtx context.Context) {
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /v1/road-events", roadEventsHandler(fwd, appCtx))
}
