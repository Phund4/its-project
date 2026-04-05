package domain

import (
	"context"
	"time"
)

// EventStore сохраняет инциденты и срезы загруженности в ClickHouse.
type EventStore interface {
	InsertIncident(ctx context.Context, observedAt time.Time, segmentID, cameraID, s3Key string, crashProb float64, label, rawML string) error
	InsertCongestion(ctx context.Context, observedAt time.Time, segmentID, cameraID, s3Key string, congestionScore float64, rawML string) error
	Close() error
}
