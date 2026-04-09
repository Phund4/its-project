package postgres

import (
	"context"
	"fmt"
	"strings"

	"traffic-coordinator/internal/core/domain"
)

func (r *Repo) ZoneWorkers(ctx context.Context, zoneID string) (map[string][]domain.Replica, error) {
	query, args := buildZoneWorkersQuery(zoneID)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToQueryDB, err)
	}
	defer rows.Close()

	out := map[string][]domain.Replica{}
	for rows.Next() {
		var zid string
		var rep domain.Replica
		if err := rows.Scan(&zid, &rep.ClusterID, &rep.InstanceID, &rep.URL); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToScanDB, err)
		}
		rep.ClusterID = strings.TrimSpace(rep.ClusterID)
		rep.InstanceID = strings.TrimSpace(rep.InstanceID)
		out[zid] = append(out[zid], rep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToQueryDB, err)
	}
	return out, nil
}
