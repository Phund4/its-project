package app

import (
	"context"
	"sort"
	"time"

	"traffic-coordinator/internal/core/domain"
)

type Store interface {
	Sources(ctx context.Context, zoneID string) ([]domain.Source, error)
	ZoneWorkers(ctx context.Context, zoneID string) (map[string][]domain.Replica, error)
	UpsertWorkerStatus(ctx context.Context, status domain.WorkerStatusSnapshot) error
	ListWorkerStatuses(ctx context.Context) ([]domain.WorkerStatusSnapshot, error)
}

// App работает поверх общего стора (PostgreSQL или in-memory).
type App struct {
	store               Store
	workerStatusTimeout time.Duration
}

func New(store Store, workerStatusTimeout time.Duration) *App {
	if workerStatusTimeout <= 0 {
		workerStatusTimeout = 30 * time.Second
	}
	return &App{
		store:               store,
		workerStatusTimeout: workerStatusTimeout,
	}
}

func (a *App) Sources(zoneID string) []domain.Source {
	items, err := a.store.Sources(context.Background(), zoneID)
	if err != nil {
		return nil
	}
	return items
}

func (a *App) Assignments(zoneID, clusterID, instanceID, dataClass string) []domain.Source {
	ctx := context.Background()
	candidates, err := a.store.Sources(ctx, zoneID)
	if err != nil {
		return nil
	}
	workersByZone, err := a.store.ZoneWorkers(ctx, zoneID)
	if err != nil {
		return nil
	}
	hbs, err := a.store.ListWorkerStatuses(ctx)
	if err != nil {
		return nil
	}
	workerStatuses := make(map[string]domain.WorkerStatusSnapshot, len(hbs))
	for _, status := range hbs {
		workerStatuses[workerKey(status.ZoneID, status.ClusterID, status.InstanceID)] = status
	}
	filtered := make([]domain.Source, 0, len(candidates))
	for _, s := range candidates {
		if dataClass != "" && s.DataClass != dataClass {
			continue
		}
		filtered = append(filtered, s)
	}
	candidates = filtered
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].SourceID < candidates[j].SourceID
	})

	sim := a.initialSim(zoneID, workersByZone, workerStatuses)
	owners := make(map[string]domain.Replica, len(candidates))

	for _, s := range candidates {
		pool := workersByZone[s.ZoneID]
		alive := a.aliveReplicas(s.ZoneID, pool, workerStatuses)
		if len(alive) == 0 {
			continue
		}
		best := pickLeastLoaded(s.ZoneID, alive, pool, sim)
		owners[s.SourceID] = best
		k := workerKey(s.ZoneID, best.ClusterID, best.InstanceID)
		sim[k]++
	}

	out := make([]domain.Source, 0, len(candidates))
	for _, s := range candidates {
		br, ok := owners[s.SourceID]
		if !ok {
			continue
		}
		if clusterID != "" && br.ClusterID != clusterID {
			continue
		}
		if instanceID != "" && br.InstanceID != instanceID {
			continue
		}
		out = append(out, s)
	}
	return out
}

// initialSim — стартовая «занятость» по worker status (assignments + load), только для живых нод в зоне.
func (a *App) initialSim(zoneID string, workersByZone map[string][]domain.Replica, workerStatuses map[string]domain.WorkerStatusSnapshot) map[string]float64 {
	sim := make(map[string]float64)
	for zid, pool := range workersByZone {
		if zoneID != "" && zid != zoneID {
			continue
		}
		for _, r := range pool {
			k := workerKey(zid, r.ClusterID, r.InstanceID)
			if _, ok := sim[k]; ok {
				continue
			}
			if !a.isAlive(zid, r.ClusterID, r.InstanceID, workerStatuses) {
				continue
			}
			status := workerStatuses[k]
			sim[k] = float64(status.Assignments) + status.Load
		}
	}
	return sim
}

func (a *App) aliveReplicas(zoneID string, pool []domain.Replica, workerStatuses map[string]domain.WorkerStatusSnapshot) []domain.Replica {
	out := make([]domain.Replica, 0, len(pool))
	for _, r := range pool {
		if a.isAlive(zoneID, r.ClusterID, r.InstanceID, workerStatuses) {
			out = append(out, r)
		}
	}
	return out
}

// pickLeastLoaded — реплика с минимальным sim[key]; при равенстве — меньший индекс в пуле зоны (стабильный tie-break).
func pickLeastLoaded(zoneID string, alive []domain.Replica, order []domain.Replica, sim map[string]float64) domain.Replica {
	index := make(map[string]int, len(order))
	for i, r := range order {
		k := r.ClusterID + "|" + r.InstanceID
		if _, ok := index[k]; !ok {
			index[k] = i
		}
	}
	best := alive[0]
	bestK := workerKey(zoneID, best.ClusterID, best.InstanceID)
	bestScore := sim[bestK]
	bestIdx := index[best.ClusterID+"|"+best.InstanceID]

	for _, r := range alive[1:] {
		k := workerKey(zoneID, r.ClusterID, r.InstanceID)
		sc := sim[k]
		idx := index[r.ClusterID+"|"+r.InstanceID]
		if sc < bestScore || (sc == bestScore && idx < bestIdx) {
			best = r
			bestScore = sc
			bestIdx = idx
		}
	}
	return best
}

func (a *App) UpsertWorkerStatus(status domain.WorkerStatusSnapshot) {
	status.ObservedAt = time.Now().UTC()
	_ = a.store.UpsertWorkerStatus(context.Background(), status)
}

func (a *App) ListWorkerStatuses() []domain.WorkerStatusSnapshot {
	out, err := a.store.ListWorkerStatuses(context.Background())
	if err != nil {
		return nil
	}
	return out
}

// IngestionInstances возвращает конфигурированные инстансы из ingestion_instances.yaml (в т.ч. url для наблюдения).
func (a *App) IngestionInstances(zoneID string) []domain.IngestionInstance {
	workersByZone, err := a.store.ZoneWorkers(context.Background(), zoneID)
	if err != nil {
		return nil
	}
	out := make([]domain.IngestionInstance, 0)
	for zid, pool := range workersByZone {
		for _, r := range pool {
			out = append(out, domain.IngestionInstance{
				ZoneID:     zid,
				ClusterID:  r.ClusterID,
				InstanceID: r.InstanceID,
				URL:        r.URL,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ZoneID != out[j].ZoneID {
			return out[i].ZoneID < out[j].ZoneID
		}
		if out[i].ClusterID != out[j].ClusterID {
			return out[i].ClusterID < out[j].ClusterID
		}
		return out[i].InstanceID < out[j].InstanceID
	})
	return out
}

func workerKey(zoneID, clusterID, instanceID string) string {
	return zoneID + "|" + clusterID + "|" + instanceID
}

func (a *App) isAlive(zoneID, clusterID, instanceID string, workerStatuses map[string]domain.WorkerStatusSnapshot) bool {
	if clusterID == "" || instanceID == "" {
		return false
	}
	status, ok := workerStatuses[workerKey(zoneID, clusterID, instanceID)]
	if !ok {
		return false
	}
	return time.Since(status.ObservedAt) <= a.workerStatusTimeout
}
