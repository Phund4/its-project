package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	mapv1 "traffic-analytics/api/map/v1"
)

// MunicipalityJSON элемент ответа GET /api/v1/municipalities.
type MunicipalityJSON struct {
	// MunicipalityID код города в справочнике (msk, spb, …).
	MunicipalityID string `json:"municipality_id"`

	// NameRU отображаемое название.
	NameRU string `json:"name_ru"`

	// CenterLat широта центра карты WGS84.
	CenterLat float64 `json:"center_lat"`

	// CenterLon долгота центра карты WGS84.
	CenterLon float64 `json:"center_lon"`

	// DefaultZoom масштаб Leaflet по умолчанию.
	DefaultZoom uint8 `json:"default_zoom"`
}

// StopJSON элемент ответа GET /api/v1/stops.
type StopJSON struct {
	// StopID UUID остановки в строковом виде.
	StopID string `json:"stop_id"`

	// StopCode код в НСИ / внутренний.
	StopCode string `json:"stop_code"`

	// Name полное наименование.
	Name string `json:"name"`

	// NameShort краткое имя для подписи на карте.
	NameShort string `json:"name_short"`

	// Lat широта WGS84.
	Lat float64 `json:"lat"`

	// Lon долгота WGS84.
	Lon float64 `json:"lon"`
}

// BusJSON элемент ответа GET /api/v1/buses.
type BusJSON struct {
	// VehicleID идентификатор ТС.
	VehicleID string `json:"vehicle_id"`

	// RouteID номер/ид маршрута.
	RouteID string `json:"route_id"`

	// Lat широта WGS84.
	Lat float64 `json:"lat"`

	// Lon долгота WGS84.
	Lon float64 `json:"lon"`

	// SpeedKmh скорость км/ч.
	SpeedKmh float64 `json:"speed_kmh"`

	// HeadingDeg курс в градусах.
	HeadingDeg float64 `json:"heading_deg"`

	// ObservedAtRfc3339 момент наблюдения по телеметрии.
	ObservedAtRfc3339 string `json:"observed_at_rfc3339"`
}

// Handlers REST-прокси к analytics по gRPC.
type Handlers struct {
	// Map клиент gRPC map.v1.MapPortal.
	Map mapv1.MapPortalClient
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handlers) GetMunicipalities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp, err := h.Map.ListMunicipalities(ctx, &mapv1.ListMunicipalitiesRequest{})
	if err != nil {
		slog.Error("municipalities", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "analytics"})
		return
	}
	out := make([]MunicipalityJSON, 0, len(resp.Items))
	for _, m := range resp.Items {
		out = append(out, MunicipalityJSON{
			MunicipalityID: m.GetMunicipalityId(),
			NameRU:         m.GetNameRu(),
			CenterLat:      m.GetCenterLat(),
			CenterLon:      m.GetCenterLon(),
			DefaultZoom:    uint8(m.GetDefaultZoom()),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) GetStops(w http.ResponseWriter, r *http.Request) {
	mid := strings.TrimSpace(r.URL.Query().Get("municipality_id"))
	if mid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "municipality_id required"})
		return
	}
	resp, err := h.Map.ListStops(r.Context(), &mapv1.ListStopsRequest{MunicipalityId: mid})
	if err != nil {
		slog.Error("stops", "municipality_id", mid, "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "analytics"})
		return
	}
	out := make([]StopJSON, 0, len(resp.Items))
	for _, s := range resp.Items {
		out = append(out, StopJSON{
			StopID:    s.GetStopId(),
			StopCode:  s.GetStopCode(),
			Name:      s.GetName(),
			NameShort: s.GetNameShort(),
			Lat:       s.GetLat(),
			Lon:       s.GetLon(),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) GetBuses(w http.ResponseWriter, r *http.Request) {
	mid := strings.TrimSpace(r.URL.Query().Get("municipality_id"))
	if mid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "municipality_id required"})
		return
	}
	resp, err := h.Map.ListBuses(r.Context(), &mapv1.ListBusesRequest{MunicipalityId: mid})
	if err != nil {
		slog.Error("buses", "municipality_id", mid, "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "analytics"})
		return
	}
	out := make([]BusJSON, 0, len(resp.Items))
	for _, b := range resp.Items {
		out = append(out, BusJSON{
			VehicleID:         b.GetVehicleId(),
			RouteID:           b.GetRouteId(),
			Lat:               b.GetLat(),
			Lon:               b.GetLon(),
			SpeedKmh:          b.GetSpeedKmh(),
			HeadingDeg:        b.GetHeadingDeg(),
			ObservedAtRfc3339: b.GetObservedAtRfc3339(),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// Register монтирует маршруты на mux.
func Register(mux *http.ServeMux, cli mapv1.MapPortalClient) {
	h := &Handlers{Map: cli}
	mux.HandleFunc("GET /api/v1/municipalities", h.GetMunicipalities)
	mux.HandleFunc("GET /api/v1/stops", h.GetStops)
	mux.HandleFunc("GET /api/v1/buses", h.GetBuses)
}
