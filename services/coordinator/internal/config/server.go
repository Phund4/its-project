package config

import (
	"errors"
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

const (
	serverConfigPrefix = "SERVER"
)

var (
	ErrFailedToLoadServerConfig     = errors.New("failed to load server config")
	ErrFailedToValidateServerConfig = errors.New("failed to validate server config")
	ErrListenAddrRequired           = errors.New("listen addr is required")
)

type ServerConfig struct {
	ListenAddr string `envconfig:"LISTEN_ADDR"`
}

func (c *ServerConfig) Validate() error {
	if c.ListenAddr == "" {
		return ErrListenAddrRequired
	}
	return nil
}

func loadServerConfig() (ServerConfig, error) {
	var s ServerConfig

	if err := envconfig.Process(serverConfigPrefix, &s); err != nil {
		return ServerConfig{}, fmt.Errorf("%w: %w", ErrFailedToLoadServerConfig, err)
	}

	if err := s.Validate(); err != nil {
		return ServerConfig{}, fmt.Errorf("%w: %w", ErrFailedToValidateServerConfig, err)
	}

	return s, nil
}
