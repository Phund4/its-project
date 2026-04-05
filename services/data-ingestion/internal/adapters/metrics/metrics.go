// Package metrics — Prometheus для data_ingestion.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var OperationErrors = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "data_ingestion_operation_errors_total",
		Help: "Errors during ingest pipeline by stage.",
	},
	[]string{"stage"},
)

// TelemetryForwarded — успешно переслано в analytics (unary SendBusTelemetry).
var TelemetryForwarded = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "data_ingestion_bus_telemetry_forwarded_total",
		Help: "Bus telemetry unary RPCs successfully forwarded to analytics ingest.",
	},
)
