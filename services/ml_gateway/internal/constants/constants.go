// Package constants — константы ml_gateway.
package constants

const (
	// MaxRoadEventBodyBytes — лимит тела POST /v1/road-events.
	MaxRoadEventBodyBytes = 8 << 20
	// HTTPReadHeaderTimeoutSec — таймаут чтения заголовка HTTP.
	HTTPReadHeaderTimeoutSec = 10
	// HTTPServerShutdownSec — таймаут graceful shutdown HTTP.
	HTTPServerShutdownSec = 15
)
