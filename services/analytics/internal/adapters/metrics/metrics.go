// Package metrics — Prometheus для analytics (Help на английском).
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CongestionScore = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "analytics_road_congestion_score",
			Help: "Last congestion score from ML per road segment and camera.",
		},
		[]string{"segment_id", "camera_id"},
	)
	CrashProbability = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "analytics_road_crash_probability",
			Help: "Last crash probability from ML per road segment and camera.",
		},
		[]string{"segment_id", "camera_id"},
	)
	CrashAlert = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "analytics_road_crash_alert",
			Help: "1 if incident rule fired (crash label or probability threshold), else 0.",
		},
		[]string{"segment_id", "camera_id"},
	)
	IncidentsRecorded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "analytics_road_incidents_recorded_total",
			Help: "Rows successfully inserted into ClickHouse incidents table.",
		},
		[]string{"segment_id", "camera_id"},
	)
	CongestionRecorded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "analytics_road_congestion_recorded_total",
			Help: "Rows successfully inserted into ClickHouse congestion table.",
		},
		[]string{"segment_id", "camera_id"},
	)
	TelemetryIngested = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "analytics_bus_telemetry_ingested_total",
			Help: "Ingest requests that carried a non-empty telemetry payload (not ML).",
		},
		[]string{"segment_id", "camera_id"},
	)
	ClickHouseErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "analytics_clickhouse_errors_total",
			Help: "ClickHouse client failures by operation.",
		},
		[]string{"op"},
	)
	IngestErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "analytics_ingest_errors_total",
			Help: "Ingest handler errors by stage.",
		},
		[]string{"stage"},
	)
)
