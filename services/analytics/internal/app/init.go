package app

import (
	"context"
	"fmt"

	"traffic-analytics/internal/adapters/clickhouse"
	"traffic-analytics/internal/config"
	"traffic-analytics/internal/core/services"
	"traffic-analytics/internal/portalhub"
)

// Deps инициализированные адаптеры и сервис приложения.
type Deps struct {
	// Config снимок настроек из окружения.
	Config config.Config

	// Store клиент ClickHouse для OLAP и справочников.
	Store *clickhouse.Store

	// Ingest HTTP-обработчик приёма событий.
	Ingest *services.IngestService

	// PortalHub память позиций автобусов для gRPC карты.
	PortalHub *portalhub.Hub

	// CHAddr нормализованный адрес ClickHouse (для логов).
	CHAddr string
}

// InitializeDependencies загружает конфиг (включая .env), подключает ClickHouse и создаёт IngestService.
func InitializeDependencies(ctx context.Context) (*Deps, error) {
	cfg := config.Load()

	chAddr := clickhouse.NormalizeAddr(cfg.ClickHouseAddr)
	store, err := clickhouse.New(ctx, chAddr, cfg.ClickHouseDatabase, cfg.ClickHouseUser, cfg.ClickHousePassword, cfg.IncidentsTable, cfg.CongestionTable)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: %w", err)
	}

	hub := portalhub.New(cfg.MunicipalityActivityTTL)
	ingest := services.NewIngestService(store, cfg, ctx, hub)
	return &Deps{
		Config:    cfg,
		Store:     store,
		Ingest:    ingest,
		PortalHub: hub,
		CHAddr:    chAddr,
	}, nil
}

// Close освобождает ресурсы зависимостей.
func (d *Deps) Close() error {
	if d.Store == nil {
		return nil
	}
	return d.Store.Close()
}
