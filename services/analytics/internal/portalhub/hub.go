package portalhub

import (
	"sync"
	"time"
)

// BusSnapshot последняя известная позиция ТС для отдачи на карту.
type BusSnapshot struct {
	// VehicleID идентификатор транспортного средства.
	VehicleID string

	// RouteID маршрут.
	RouteID string

	// Lat широта WGS84.
	Lat float64

	// Lon долгота WGS84.
	Lon float64

	// SpeedKmh скорость км/ч.
	SpeedKmh float64

	// HeadingDeg курс градусы.
	HeadingDeg float64

	// ObservedAtRfc3339 время фиксации телеметрии.
	ObservedAtRfc3339 string
}

// Hub хранит автобусы по municipality_id и TTL активности UI.
type Hub struct {
	// mu защита карт lastActiv и buses.
	mu sync.Mutex

	// ttl как долго после ListStops/ListBuses принимать UpsertBus.
	ttl time.Duration

	// lastActiv время последнего запроса карты по городу.
	lastActiv map[string]time.Time

	// buses municipality_id -> vehicle_id -> снимок.
	buses map[string]map[string]BusSnapshot
}

// New создаёт хаб с заданным TTL активности.
func New(ttl time.Duration) *Hub {
	if ttl <= 0 {
		ttl = 45 * time.Second
	}
	return &Hub{
		ttl:       ttl,
		lastActiv: make(map[string]time.Time),
		buses:     make(map[string]map[string]BusSnapshot),
	}
}

// TouchMunicipality продлевает окно приёма телеметрии для города.
func (h *Hub) TouchMunicipality(municipalityID string) {
	if municipalityID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastActiv[municipalityID] = time.Now()
}

func (h *Hub) isActiveLocked(municipalityID string) bool {
	t, ok := h.lastActiv[municipalityID]
	return ok && time.Since(t) <= h.ttl
}

// UpsertBus записывает позицию только если город активен по TTL.
func (h *Hub) UpsertBus(municipalityID string, snap BusSnapshot) bool {
	if municipalityID == "" || snap.VehicleID == "" {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.isActiveLocked(municipalityID) {
		return false
	}
	if h.buses[municipalityID] == nil {
		h.buses[municipalityID] = make(map[string]BusSnapshot)
	}
	h.buses[municipalityID][snap.VehicleID] = snap
	return true
}

// ListBuses возвращает все снимки по municipality_id.
func (h *Hub) ListBuses(municipalityID string) []BusSnapshot {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.buses[municipalityID]
	if len(m) == 0 {
		return nil
	}
	out := make([]BusSnapshot, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
