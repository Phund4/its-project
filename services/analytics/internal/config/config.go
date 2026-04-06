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
	// ListenAddr адрес HTTP (ingest, metrics, health).
	ListenAddr string

	// ClickHouseAddr host:port native-протокола ClickHouse.
	ClickHouseAddr string

	// ClickHouseDatabase БД по умолчанию для таблиц analytics (incidents/congestion).
	ClickHouseDatabase string

	// ClickHouseUser имя пользователя CH.
	ClickHouseUser string

	// ClickHousePassword пароль CH (может быть пустым в dev).
	ClickHousePassword string

	// IncidentsTable имя таблицы инцидентов.
	IncidentsTable string

	// CongestionTable имя таблицы загруженности.
	CongestionTable string

	// CrashAlertThreshold порог crash_probability для алерта и записи инцидента.
	CrashAlertThreshold float64

	// CongestionPersistInterval минимальный интервал записи congestion на пару (segment, camera).
	CongestionPersistInterval time.Duration

	// MapGRPCListenAddr адрес gRPC map.v1.MapPortal для map_portal.
	MapGRPCListenAddr string

	// InfraSimDatabase БД со справочниками карты (municipalities, bus_stops).
	InfraSimDatabase string

	// MunicipalityActivityTTL окно «активности» города для приёма телеметрии в память карты.
	MunicipalityActivityTTL time.Duration

	// KafkaBootstrap серверы брокера (через запятую); пусто — консьюмер Kafka не запускается.
	KafkaBootstrap string

	// KafkaConsumerGroup группа для чтения топиков ingest.
	KafkaConsumerGroup string

	// KafkaTopicVideo топик событий ML/видео (ml-gateway → analytics).
	KafkaTopicVideo string

	// KafkaTopicTelemetry топик телеметрии ТС (data-ingestion → analytics).
	KafkaTopicTelemetry string
}

// Load читает переменные окружения и возвращает Config с дефолтами (предварительно подгружает .env).
func Load() Config {
	_ = tryLoadEnvFile()
	c := Config{
		ListenAddr:                ":8093",
		ClickHouseAddr:            "127.0.0.1:9000",
		ClickHouseDatabase:        "default",
		ClickHouseUser:            "default",
		IncidentsTable:            "road_incidents",
		CongestionTable:           "road_congestion",
		CrashAlertThreshold:       0.5,
		CongestionPersistInterval: 2 * time.Second,
		MapGRPCListenAddr:         ":8097",
		InfraSimDatabase:          "its_infra_sim",
		MunicipalityActivityTTL:   45 * time.Second,
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
	if v := strings.TrimSpace(os.Getenv("MAP_GRPC_LISTEN_ADDR")); v != "" {
		c.MapGRPCListenAddr = v
	}
	if v := strings.TrimSpace(os.Getenv("INFRA_SIM_DATABASE")); v != "" {
		c.InfraSimDatabase = v
	}
	if v := strings.TrimSpace(os.Getenv("MUNICIPALITY_ACTIVITY_TTL_SEC")); v != "" {
		if sec, err := strconv.ParseFloat(v, 64); err == nil && sec > 0 {
			c.MunicipalityActivityTTL = time.Duration(sec * float64(time.Second))
		}
	}
	c.KafkaBootstrap = strings.TrimSpace(os.Getenv("KAFKA_BOOTSTRAP_SERVERS"))
	c.KafkaConsumerGroup = strings.TrimSpace(os.Getenv("KAFKA_CONSUMER_GROUP"))
	if c.KafkaConsumerGroup == "" {
		c.KafkaConsumerGroup = "analytics-ingest"
	}
	c.KafkaTopicVideo = strings.TrimSpace(os.Getenv("KAFKA_TOPIC_VIDEO"))
	if c.KafkaTopicVideo == "" {
		c.KafkaTopicVideo = "its.video.ingest"
	}
	c.KafkaTopicTelemetry = strings.TrimSpace(os.Getenv("KAFKA_TOPIC_TELEMETRY"))
	if c.KafkaTopicTelemetry == "" {
		c.KafkaTopicTelemetry = "its.telemetry.ingest"
	}
	return c
}
