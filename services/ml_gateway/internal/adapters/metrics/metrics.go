// Package metrics — Prometheus для ml_gateway.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var OperationErrors = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "ml_gateway_operation_errors_total",
		Help: "Validation and analytics forward errors by stage.",
	},
	[]string{"stage"},
)
