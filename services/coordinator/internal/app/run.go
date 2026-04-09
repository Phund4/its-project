package app

import (
	"context"
	"errors"
	"fmt"

	httpserver "traffic-coordinator/internal/adapters/http"
	"traffic-coordinator/internal/config"
)

var (
	ErrRunHTTPServer = errors.New("failed to run HTTP server")
)

func (a *App) Run(ctx context.Context, cfg config.ServerConfig) error {
	server := httpserver.New(a)
	if err := server.Run(ctx, cfg); err != nil {
		return fmt.Errorf("%w: %w", ErrRunHTTPServer, err)
	}
	return nil
}
