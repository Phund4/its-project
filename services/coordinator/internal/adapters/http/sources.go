package httpserver

import (
	"net/http"
	"strings"
)

func (s *Server) handleSources(w http.ResponseWriter, r *http.Request) {
	zoneID := strings.TrimSpace(r.URL.Query().Get(sourcesZoneIDKey))
	if zoneID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{sourcesErrorKey: zoneIDRequiredError})
		return
	}

	sources := s.service.Sources(zoneID)
	writeJSON(w, http.StatusOK, map[string]any{itemsKey: sources})
}
