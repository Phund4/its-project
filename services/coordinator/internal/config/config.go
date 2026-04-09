package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

var (
	ErrLoadServerConfig             = errors.New("failed to load server config")
	ErrLoadCoordinatorServiceConfig = errors.New("failed to load coordinator service config")
	ErrLoadPostgresConfig           = errors.New("failed to load postgres config")
	ErrEnvVarFromFile               = errors.New("failed to load environment variables from file")
)

type Config struct {
	Server      ServerConfig
	Coordinator CoordinatorConfig
	Postgres    PostgresConfig
}

const (
	envFilePathKey = "ENV_FILE"
)

func LoadConfig() (*Config, error) {
	if err := tryLoadDotEnv(); err != nil {
		return nil, err
	}

	serverCfg, err := loadServerConfig()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLoadServerConfig, err)
	}

	coordinatorCfg, err := loadCoordinatorConfig()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLoadCoordinatorServiceConfig, err)
	}

	postgresCfg, err := loadPostgresConfig()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLoadPostgresConfig, err)
	}

	return &Config{
		Server:      serverCfg,
		Coordinator: coordinatorCfg,
		Postgres:    postgresCfg,
	}, nil
}

func tryLoadDotEnv() error {
	envPath := strings.TrimSpace(os.Getenv(envFilePathKey))
	if envPath == "" {
		envPath = ".env"
	}

	if err := godotenv.Load(envPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %w", ErrEnvVarFromFile, err)
	}

	return nil
}
