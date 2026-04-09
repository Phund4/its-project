package config

import (
	"errors"
	"fmt"
	"slices"

	"github.com/kelseyhightower/envconfig"
)

const (
	coordinatorConfigPrefix = "COORDINATOR"

	coordinatorStorePostgres = "postgres"
	coordinatorStoreMemory   = "memory"
)

var coordinatorStoreOptions = []string{
	coordinatorStorePostgres,
	coordinatorStoreMemory,
}

var (
	ErrFailedToLoadCoordinatorConfig     = errors.New("failed to load coordinator config")
	ErrFailedToValidateCoordinatorConfig = errors.New("failed to validate coordinator config")
	ErrWorkerStatusTimeoutSecRequired    = errors.New("worker status timeout sec is required")
	ErrInvalidCoordinatorStore           = errors.New("coordinator store must be one of: postgres, memory")
)

type CoordinatorConfig struct {
	WorkerStatusTimeoutSec int    `envconfig:"WORKER_STATUS_TIMEOUT_SEC" default:"30"`
	Store                  string `envconfig:"STORE" default:"postgres"`
}

func ValidateCoordinatorConfig(cfg CoordinatorConfig) error {
	if cfg.WorkerStatusTimeoutSec <= 0 {
		return ErrWorkerStatusTimeoutSecRequired
	}
	if !slices.Contains(coordinatorStoreOptions, cfg.Store) {
		return ErrInvalidCoordinatorStore
	}
	return nil
}

func loadCoordinatorConfig() (CoordinatorConfig, error) {
	var cfg CoordinatorConfig
	if err := envconfig.Process(coordinatorConfigPrefix, &cfg); err != nil {
		return CoordinatorConfig{}, fmt.Errorf("%w: %w", ErrFailedToLoadCoordinatorConfig, err)
	}

	if err := ValidateCoordinatorConfig(cfg); err != nil {
		return CoordinatorConfig{}, fmt.Errorf("%w: %w", ErrFailedToValidateCoordinatorConfig, err)
	}
	return cfg, nil
}
