// Package constants — константы сервиса analytics.
package constants

const (
	// MaxIngestBodyBytes — лимит тела POST /v1/ingest.
	MaxIngestBodyBytes = 8 << 20
	// MaxLabelLen — максимальная длина метки Prometheus (кардинальность).
	MaxLabelLen = 128
	// HTTPReadHeaderTimeoutSec — таймаут чтения заголовка входящего HTTP.
	HTTPReadHeaderTimeoutSec = 10
	// HTTPServerShutdownSec — таймаут graceful shutdown HTTP.
	HTTPServerShutdownSec = 15
	// ClickHouseQueryTimeoutSec — таймаут одной операции записи в ClickHouse из хендлера.
	ClickHouseQueryTimeoutSec = 8
)
