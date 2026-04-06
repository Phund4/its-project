package config

import (
	"os"
	"strings"
)

// ApplyEnvOverrides перезаписывает поля конфига значениями из переменных окружения.
func ApplyEnvOverrides(c *Root) {
	if v := os.Getenv("ML_BASE_URL"); v != "" {
		c.ML.BaseURL = strings.TrimRight(v, "/")
	}
	if v := os.Getenv("ML_PROCESS_PATH"); v != "" {
		c.ML.ProcessPath = v
	}
	if v := os.Getenv("S3_ENDPOINT"); v != "" {
		c.S3.Endpoint = strings.TrimRight(v, "/")
	}
	if v := os.Getenv("METRICS_LISTEN_ADDR"); v != "" {
		c.Metrics.ListenAddr = v
	}
}

// TelemetryListenAddrFromEnv — адрес gRPC-сервера телеметрии (по умолчанию :50051).
func TelemetryListenAddrFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("TELEMETRY_GRPC_LISTEN_ADDR")); v != "" {
		return v
	}
	return ":50051"
}

// AnalyticsIngestURLFromEnv — полный URL POST /v1/ingest (если телеметрия без Kafka).
func AnalyticsIngestURLFromEnv() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv("ANALYTICS_INGEST_URL")), "/")
}

// KafkaBootstrapFromEnv — брокеры для телеметрии (пусто = только HTTP).
func KafkaBootstrapFromEnv() string {
	return strings.TrimSpace(os.Getenv("KAFKA_BOOTSTRAP_SERVERS"))
}

// KafkaTopicTelemetryFromEnv — топик телеметрии (совпадает с analytics).
func KafkaTopicTelemetryFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("KAFKA_TOPIC_TELEMETRY")); v != "" {
		return v
	}
	return "its.telemetry.ingest"
}
