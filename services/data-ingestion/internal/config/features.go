package config

import (
	"os"
	"strings"
)

// Features legacy-флаги.
// Режимы теперь определяются назначениями из coordinator, а не env-переменными.
type Features struct {
	// CamerasEnabled контур RTSP → ffmpeg → S3 → ML.
	CamerasEnabled bool

	// S3Enabled загрузка кадров в объектное хранилище.
	S3Enabled bool

	// MLEnabled вызовы HTTP ML по кадрам.
	MLEnabled bool

	// TelemetryGRPC приём BusTelemetry по gRPC и форвард в analytics.
	TelemetryGRPC bool
}

func envBool(key string, defaultVal bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultVal
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultVal
	}
}

// FeaturesFromEnv читает флаги: по умолчанию камеры+S3+ML включены, телеметрия gRPC выключена.
func FeaturesFromEnv() Features {
	return Features{
		CamerasEnabled: true,
		S3Enabled:      true,
		MLEnabled:      true,
		TelemetryGRPC:  true,
	}
}
