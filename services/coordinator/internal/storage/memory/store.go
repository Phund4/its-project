package memory

import (
	"context"
	"sync"

	"traffic-coordinator/internal/core/domain"
)

type Store struct {
	mu          sync.RWMutex
	sources     []domain.Source
	zoneWorkers map[string][]domain.Replica
	heartbeats  map[string]domain.WorkerHeartbeat
}

func New(sources []domain.Source, zoneWorkers map[string][]domain.Replica) *Store {
	if zoneWorkers == nil {
		zoneWorkers = map[string][]domain.Replica{}
	}
	return &Store{
		sources:     append([]domain.Source(nil), sources...),
		zoneWorkers: zoneWorkers,
		heartbeats:  map[string]domain.WorkerHeartbeat{},
	}
}

func (s *Store) Sources(_ context.Context, zoneID string) ([]domain.Source, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Source, 0, len(s.sources))
	for _, src := range s.sources {
		if !src.Enabled {
			continue
		}
		if zoneID != "" && src.ZoneID != zoneID {
			continue
		}
		out = append(out, src)
	}
	return out, nil
}

func (s *Store) ZoneWorkers(_ context.Context, zoneID string) (map[string][]domain.Replica, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string][]domain.Replica{}
	for zid, pool := range s.zoneWorkers {
		if zoneID != "" && zid != zoneID {
			continue
		}
		cp := append([]domain.Replica(nil), pool...)
		out[zid] = cp
	}
	return out, nil
}

func (s *Store) UpsertHeartbeat(_ context.Context, hb domain.WorkerHeartbeat) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hb.ZoneID + "|" + hb.ClusterID + "|" + hb.InstanceID
	s.heartbeats[k] = hb
	return nil
}

func (s *Store) Heartbeats(_ context.Context) ([]domain.WorkerHeartbeat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.WorkerHeartbeat, 0, len(s.heartbeats))
	for _, hb := range s.heartbeats {
		out = append(out, hb)
	}
	return out, nil
}

