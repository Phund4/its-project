package app

import (
	"net/http"

	"ml-gateway/internal/config"
	"ml-gateway/internal/core/services"
)

// Deps конфигурация и сервис пересылки после инициализации.
type Deps struct {
	// Config снимок окружения.
	Config config.Config

	// Forwarder POST тела в analytics ingest.
	Forwarder *services.Forwarder

	// HTTPClient клиент с AnalyticsTimeout.
	HTTPClient *http.Client
}

// InitializeDependencies создаёт HTTP-клиент и Forwarder из конфигурации.
func InitializeDependencies() *Deps {
	cfg := config.Load()
	client := &http.Client{Timeout: cfg.AnalyticsTimeout}
	fwd := services.NewForwarder(cfg, client)
	return &Deps{
		Config:     cfg,
		Forwarder:  fwd,
		HTTPClient: client,
	}
}
