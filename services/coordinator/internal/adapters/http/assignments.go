package httpserver

import (
	"net/http"
	"strings"
)

func (s *Server) handleAssignments(w http.ResponseWriter, r *http.Request) {
	zoneID := strings.TrimSpace(r.URL.Query().Get(assignmentsZoneIDKey))
	clusterID := strings.TrimSpace(r.URL.Query().Get(assignmentsClusterIDKey))
	instanceID := strings.TrimSpace(r.URL.Query().Get(assignmentsInstanceIDKey))
	dataClass := strings.TrimSpace(r.URL.Query().Get(assignmentsDataClassKey))
	writeJSON(w, http.StatusOK, map[string]any{itemsKey: s.service.Assignments(zoneID, clusterID, instanceID, dataClass)})
}
