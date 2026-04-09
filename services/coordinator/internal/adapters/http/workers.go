package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"traffic-coordinator/internal/core/domain"
)

func (s *Server) handleWorkerStatusUpsert(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var status domain.WorkerStatusSnapshot
	if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(status.ZoneID) == "" || strings.TrimSpace(status.ClusterID) == "" || strings.TrimSpace(status.InstanceID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "zone_id, cluster_id, instance_id are required"})
		return
	}
	s.service.UpsertWorkerStatus(status)
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleWorkers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.service.ListWorkerStatuses()})
}

func (s *Server) handleIngestionInstances(w http.ResponseWriter, r *http.Request) {
	zoneID := strings.TrimSpace(r.URL.Query().Get("zone_id"))
	writeJSON(w, http.StatusOK, map[string]any{"items": s.service.IngestionInstances(zoneID)})
}
