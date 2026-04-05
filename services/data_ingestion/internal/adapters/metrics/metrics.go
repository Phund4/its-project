// Package metrics регистрирует метрики Prometheus для сервиса data_ingestion.
// Имена и Help-строки метрик — на английском (конвенция Prometheus).
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// OperationErrors — счётчик ошибок пайплайна по стадии.
var OperationErrors = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "data_ingestion_operation_errors_total",
		Help: "Errors during ingest pipeline by stage.",
	},
	[]string{"stage"},
)
