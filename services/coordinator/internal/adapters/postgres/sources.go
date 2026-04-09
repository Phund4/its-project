package postgres

import (
	"context"
	"fmt"

	"traffic-coordinator/internal/core/domain"
)

func (r *Repo) Sources(ctx context.Context, zoneID string) ([]domain.Source, error) {
	query, args := buildSourcesQuery(zoneID)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToQueryDB, err)
	}
	defer rows.Close()

	out := make([]domain.Source, 0)
	for rows.Next() {
		var srow domain.Source
		if err := rows.Scan(
			&srow.SourceID,
			&srow.DataClass,
			&srow.ZoneID,
			&srow.SegmentID,
			&srow.CameraID,
			&srow.RTSPURL,
			&srow.Enabled,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToScanDB, err)
		}
		out = append(out, srow)
	}

	return out, rows.Err()
}
