package config

import (
	"os"
	"strings"
)

// Config настройки процесса map_portal.
type Config struct {
	// ListenAddr адрес HTTP (карта и JSON API).
	ListenAddr string

	// AnalyticsGRPCAddr host:port gRPC сервиса map.v1.MapPortal в analytics.
	AnalyticsGRPCAddr string
}

// Load читает окружение после опциональной загрузки .env.
func Load() Config {
	_ = tryLoadEnvFile()
	c := Config{
		ListenAddr:        ":8096",
		AnalyticsGRPCAddr: "127.0.0.1:8097",
	}
	if v := strings.TrimSpace(os.Getenv("LISTEN_ADDR")); v != "" {
		c.ListenAddr = v
	}
	if v := strings.TrimSpace(os.Getenv("ANALYTICS_GRPC_ADDR")); v != "" {
		c.AnalyticsGRPCAddr = v
	}
	return c
}
