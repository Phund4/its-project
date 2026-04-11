package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"traffic-coordinator/internal/app"
	"traffic-coordinator/internal/core/domain"
)

func Run(ctx context.Context, listenAddr string, a *app.App) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})

	mux.Handle("GET /metrics", promhttp.Handler())

	mux.HandleFunc("GET /v1/sources", func(w http.ResponseWriter, r *http.Request) {
		zoneID := strings.TrimSpace(r.URL.Query().Get("zone_id"))
		writeJSON(w, http.StatusOK, map[string]any{"items": a.Sources(zoneID)})
	})

	mux.HandleFunc("GET /v1/assignments", func(w http.ResponseWriter, r *http.Request) {
		zoneID := strings.TrimSpace(r.URL.Query().Get("zone_id"))
		clusterID := strings.TrimSpace(r.URL.Query().Get("cluster_id"))
		instanceID := strings.TrimSpace(r.URL.Query().Get("instance_id"))
		dataClass := strings.TrimSpace(r.URL.Query().Get("data_class"))
		writeJSON(w, http.StatusOK, map[string]any{
			"items": a.Assignments(zoneID, clusterID, instanceID, dataClass),
		})
	})

	mux.HandleFunc("POST /v1/workers/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var hb domain.WorkerHeartbeat
		if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		if strings.TrimSpace(hb.ZoneID) == "" || strings.TrimSpace(hb.ClusterID) == "" || strings.TrimSpace(hb.InstanceID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "zone_id, cluster_id, instance_id are required"})
			return
		}
		a.UpsertHeartbeat(hb)
		writeJSON(w, http.StatusNoContent, nil)
	})

	mux.HandleFunc("GET /v1/workers", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": a.Heartbeats()})
	})

	mux.HandleFunc("GET /v1/ingestion_instances", func(w http.ResponseWriter, r *http.Request) {
		zoneID := strings.TrimSpace(r.URL.Query().Get("zone_id"))
		writeJSON(w, http.StatusOK, map[string]any{"items": a.IngestionInstances(zoneID)})
	})

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
	}()
	slog.Info("coordinator starting", "listen", listenAddr)
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
