// Package httpadapter регистрирует HTTP-маршруты analytics.
package httpadapter

import (
	"net/http"

	"traffic-analytics/internal/core/services"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

// Register монтирует /metrics, GET /health и POST /v1/ingest.
func Register(mux *http.ServeMux, ingest *services.IngestService) {
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /v1/ingest", ingest.HandleIngest)
}
