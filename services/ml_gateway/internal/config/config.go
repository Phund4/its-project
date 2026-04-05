// Package config загружает настройки ml_gateway из переменных окружения.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config — параметры после Load.
type Config struct {
	ListenAddr          string
	AnalyticsBaseURL    string
	AnalyticsIngestPath string
	AnalyticsTimeout    time.Duration
}

// Load читает конфигурацию из переменных окружения.
func Load() Config {
	c := Config{
		ListenAddr:          ":8092",
		AnalyticsIngestPath: "/v1/ingest",
		AnalyticsTimeout:    10 * time.Second,
	}
	if v := strings.TrimSpace(os.Getenv("LISTEN_ADDR")); v != "" {
		c.ListenAddr = v
	}
	c.AnalyticsBaseURL = strings.TrimSpace(strings.TrimRight(os.Getenv("ANALYTICS_BASE_URL"), "/"))
	path := strings.TrimSpace(os.Getenv("ANALYTICS_INGEST_PATH"))
	if path != "" {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		c.AnalyticsIngestPath = path
	}
	if s := strings.TrimSpace(os.Getenv("ANALYTICS_TIMEOUT")); s != "" {
		if sec, err := strconv.Atoi(s); err == nil && sec > 0 {
			c.AnalyticsTimeout = time.Duration(sec) * time.Second
		}
	}
	return c
}
