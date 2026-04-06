package app

import (
	"sync"
	"time"

	"traffic-coordinator/internal/core/domain"
)

// App хранит in-memory каталог источников и heartbeat воркеров ingestion.
type App struct {
	mu               sync.RWMutex
	sources          []domain.Source
	heartbeats       map[string]domain.WorkerHeartbeat
	heartbeatTimeout time.Duration
}

func New(sources []domain.Source, heartbeatTimeout time.Duration) *App {
	if heartbeatTimeout <= 0 {
		heartbeatTimeout = 30 * time.Second
	}
	return &App{
		sources:          sources,
		heartbeats:       make(map[string]domain.WorkerHeartbeat),
		heartbeatTimeout: heartbeatTimeout,
	}
}

func (a *App) Sources(zoneID string) []domain.Source {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := make([]domain.Source, 0, len(a.sources))
	for _, s := range a.sources {
		if zoneID != "" && s.ZoneID != zoneID {
			continue
		}
		if !s.Enabled {
			continue
		}
		out = append(out, s)
	}
	return out
}

func (a *App) Assignments(zoneID, clusterID, instanceID, sourceKind string) []domain.Source {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := make([]domain.Source, 0, len(a.sources))
	for _, s := range a.sources {
		if !s.Enabled {
			continue
		}
		if zoneID != "" && s.ZoneID != zoneID {
			continue
		}
		if sourceKind != "" && s.SourceKind != sourceKind {
			continue
		}
		activeClusterID, activeInstanceID := a.activeOwner(s)
		if clusterID != "" && activeClusterID != "" && activeClusterID != clusterID {
			continue
		}
		if instanceID != "" && activeInstanceID != "" && activeInstanceID != instanceID {
			continue
		}
		out = append(out, s)
	}
	return out
}

func (a *App) UpsertHeartbeat(hb domain.WorkerHeartbeat) {
	// Время heartbeat фиксируется на стороне coordinator, чтобы не зависеть от часов клиента.
	hb.ObservedAt = time.Now().UTC()
	key := workerKey(hb.ZoneID, hb.ClusterID, hb.InstanceID)
	a.mu.Lock()
	a.heartbeats[key] = hb
	a.mu.Unlock()
}

func (a *App) Heartbeats() []domain.WorkerHeartbeat {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]domain.WorkerHeartbeat, 0, len(a.heartbeats))
	for _, hb := range a.heartbeats {
		out = append(out, hb)
	}
	return out
}

func workerKey(zoneID, clusterID, instanceID string) string {
	return zoneID + "|" + clusterID + "|" + instanceID
}

func (a *App) isAlive(zoneID, clusterID, instanceID string) bool {
	if clusterID == "" || instanceID == "" {
		return false
	}
	hb, ok := a.heartbeats[workerKey(zoneID, clusterID, instanceID)]
	if !ok {
		return false
	}
	return time.Since(hb.ObservedAt) <= a.heartbeatTimeout
}

func (a *App) activeOwner(s domain.Source) (string, string) {
	primaryAlive := a.isAlive(s.ZoneID, s.ClusterID, s.InstanceID)
	reserveDefined := s.ReserveClusterID != "" && s.ReserveInstanceID != ""
	if !reserveDefined {
		return s.ClusterID, s.InstanceID
	}
	reserveAlive := a.isAlive(s.ZoneID, s.ReserveClusterID, s.ReserveInstanceID)

	// Приоритет primary. Если primary "протух" по heartbeat, переключаем на reserve.
	if primaryAlive || !reserveAlive {
		return s.ClusterID, s.InstanceID
	}
	return s.ReserveClusterID, s.ReserveInstanceID
}
