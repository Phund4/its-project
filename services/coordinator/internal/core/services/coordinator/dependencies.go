package coordinator

import (
	"traffic-coordinator/internal/core/domain"
)

type DataStorage interface {
	GetSources() ([]domain.Source, error)
	GetIngestionInstances() ([]domain.IngestionInstance, error)
}
