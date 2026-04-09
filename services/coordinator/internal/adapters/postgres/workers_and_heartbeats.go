package postgres

import (
	"context"
	"fmt"

	"traffic-coordinator/internal/core/domain"
)

func (r *Repo) UpsertWorkerStatus(ctx context.Context, status domain.WorkerStatusSnapshot) error {
	_, err := r.db.ExecContext(ctx, upsertWorkerStatusQuery, status.ZoneID, status.ClusterID, status.InstanceID, status.Load, status.Assignments, status.ObservedAt)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToExecDB, err)
	}
	return nil
}

func (r *Repo) ListWorkerStatuses(ctx context.Context) ([]domain.WorkerStatusSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, workerStatusesQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToQueryDB, err)
	}
	defer rows.Close()

	out := make([]domain.WorkerStatusSnapshot, 0)
	for rows.Next() {
		var status domain.WorkerStatusSnapshot
		if err := rows.Scan(&status.ZoneID, &status.ClusterID, &status.InstanceID, &status.Load, &status.Assignments, &status.ObservedAt); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToScanDB, err)
		}
		out = append(out, status)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToQueryDB, err)
	}
	return out, nil
}
