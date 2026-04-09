package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	postgresConfigPrefix = "POSTGRES"
)

var (
	ErrURLRequired                    = errors.New("url is required")
	ErrMaxOpenConnsRequired           = errors.New("max_open_conns must be greater than 0")
	ErrMaxIdleConnsRequired           = errors.New("max_idle_conns must be greater than 0")
	ErrConnMaxIdleTimeRequired        = errors.New("conn_max_idle_time must be greater than 0")
	ErrFailedToLoadPostgresConfig     = errors.New("failed to load postgres config")
	ErrFailedToValidatePostgresConfig = errors.New("failed to validate postgres config")
)

type PostgresConfig struct {
	URL             string        `envconfig:"URL"`
	MaxOpenConns    int           `envconfig:"MAX_OPEN_CONNS" default:"20"`
	MaxIdleConns    int           `envconfig:"MAX_IDLE_CONNS" default:"20"`
	ConnMaxIdleTime time.Duration `envconfig:"CONN_MAX_IDLE_TIME" default:"5m"`
}

func (c *PostgresConfig) Validate() error {
	if c.URL == "" {
		return ErrURLRequired
	}
	if c.MaxOpenConns <= 0 {
		return ErrMaxOpenConnsRequired
	}
	if c.MaxIdleConns <= 0 {
		return ErrMaxIdleConnsRequired
	}
	if c.ConnMaxIdleTime <= 0 {
		return ErrConnMaxIdleTimeRequired
	}
	return nil
}

func loadPostgresConfig() (PostgresConfig, error) {
	var p PostgresConfig

	if err := envconfig.Process(postgresConfigPrefix, &p); err != nil {
		return PostgresConfig{}, fmt.Errorf("%w: %w", ErrFailedToLoadPostgresConfig, err)
	}

	if err := p.Validate(); err != nil {
		return PostgresConfig{}, fmt.Errorf("%w: %w", ErrFailedToValidatePostgresConfig, err)
	}

	return p, nil
}
