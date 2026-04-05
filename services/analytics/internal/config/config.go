// Package config читает настройки analytics из переменных окружения (после опционального .env).
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config заполняется из окружения функцией Load.
type Config struct {
	ListenAddr                string
	ClickHouseAddr            string
	ClickHouseDatabase        string
	ClickHouseUser            string
	ClickHousePassword        string
	IncidentsTable            string
	CongestionTable           string
	CrashAlertThreshold       float64
	CongestionPersistInterval time.Duration
}

// Load читает переменные окружения и возвращает Config с дефолтами.
func Load() Config {
	c := Config{
		ListenAddr:                ":8093",
		ClickHouseAddr:            "127.0.0.1:9000",
		ClickHouseDatabase:        "default",
		ClickHouseUser:            "default",
		IncidentsTable:            "road_incidents",
		CongestionTable:           "road_congestion",
		CrashAlertThreshold:       0.5,
		CongestionPersistInterval: 2 * time.Second,
	}
	if v := strings.TrimSpace(os.Getenv("LISTEN_ADDR")); v != "" {
		c.ListenAddr = v
	}
	if v := strings.TrimSpace(os.Getenv("CLICKHOUSE_ADDR")); v != "" {
		c.ClickHouseAddr = v
	}
	if v := strings.TrimSpace(os.Getenv("CLICKHOUSE_DATABASE")); v != "" {
		c.ClickHouseDatabase = v
	}
	if v := strings.TrimSpace(os.Getenv("CLICKHOUSE_USER")); v != "" {
		c.ClickHouseUser = v
	}
	c.ClickHousePassword = os.Getenv("CLICKHOUSE_PASSWORD")
	if v := strings.TrimSpace(os.Getenv("CLICKHOUSE_INCIDENTS_TABLE")); v != "" {
		c.IncidentsTable = v
	}
	if v := strings.TrimSpace(os.Getenv("CLICKHOUSE_CONGESTION_TABLE")); v != "" {
		c.CongestionTable = v
	}
	if v := strings.TrimSpace(os.Getenv("CRASH_ALERT_THRESHOLD")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.CrashAlertThreshold = f
		}
	}
	if v := strings.TrimSpace(os.Getenv("CONGESTION_PERSIST_INTERVAL_SEC")); v != "" {
		if sec, err := strconv.ParseFloat(v, 64); err == nil && sec >= 0 {
			c.CongestionPersistInterval = time.Duration(sec * float64(time.Second))
		}
	}
	return c
}
