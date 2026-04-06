package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	analyticsclient "data-ingestion/internal/adapters/analytics"
	kafkaadapter "data-ingestion/internal/adapters/kafka"
	mlclient "data-ingestion/internal/adapters/ml"
	s3store "data-ingestion/internal/adapters/s3"
	"data-ingestion/internal/adapters/telemetry"
	"data-ingestion/internal/adapters/telemetrygrpc"
	"data-ingestion/internal/config"
)

// Deps адаптеры и конфигурация после инициализации.
type Deps struct {
	// Config загруженный YAML и дефолты.
	Config *config.Root

	// Features флаги CAMERAS_ENABLED, TELEMETRY_GRPC_ENABLED, ….
	Features config.Features

	// Store клиент S3 при включённых камерах.
	Store *s3store.Client

	// ML клиент HTTP к сервису обработки кадров.
	ML *mlclient.Client

	// TelemetryPublisher HTTP или Kafka для gRPC телеметрии.
	TelemetryPublisher telemetry.Publisher

	// TelemetryGRPC сервер gRPC BusTelemetry при TELEMETRY_GRPC_ENABLED.
	TelemetryGRPC *telemetrygrpc.Server

	// TelemetryListenAddr адрес :port для gRPC телеметрии.
	TelemetryListenAddr string
}

// InitializeDependencies загружает конфиг и при необходимости S3, ML, клиент analytics.
func InitializeDependencies(ctx context.Context) (*Deps, error) {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	feat := config.FeaturesFromEnv()

	deps := &Deps{Config: cfg, Features: feat}

	if feat.TelemetryGRPC {
		kb := config.KafkaBootstrapFromEnv()
		if kb != "" {
			topic := config.KafkaTopicTelemetryFromEnv()
			kp, err := kafkaadapter.NewTelemetryProducer(kb, topic)
			if err != nil {
				return nil, fmt.Errorf("kafka telemetry producer: %w", err)
			}
			deps.TelemetryPublisher = kp
			slog.Info("telemetry to kafka", "topic", topic)
		} else {
			url := config.AnalyticsIngestURLFromEnv()
			if url == "" {
				return nil, fmt.Errorf("при TELEMETRY_GRPC_ENABLED: задайте KAFKA_BOOTSTRAP_SERVERS или ANALYTICS_INGEST_URL")
			}
			deps.TelemetryPublisher = analyticsclient.New(url)
		}
		deps.TelemetryGRPC = &telemetrygrpc.Server{Publisher: deps.TelemetryPublisher}
		deps.TelemetryListenAddr = config.TelemetryListenAddrFromEnv()
	}

	if feat.CamerasEnabled {
		ak := os.Getenv("AWS_ACCESS_KEY_ID")
		sk := os.Getenv("AWS_SECRET_ACCESS_KEY")
		if ak == "" || sk == "" {
			return nil, ErrMissingAWSCredentials
		}
		store, err := s3store.New(ctx, cfg.S3.Endpoint, cfg.S3.Region, cfg.S3.Bucket, ak, sk)
		if err != nil {
			return nil, fmt.Errorf("s3 client: %w", err)
		}
		if cfg.Ingest.CreateBucketIfMissing {
			if err := store.EnsureBucket(ctx); err != nil {
				return nil, fmt.Errorf("ensure bucket: %w", err)
			}
			slog.Info("bucket ok", "bucket", cfg.S3.Bucket)
		}
		deps.Store = store

		mlc := mlclient.New(
			cfg.ML.BaseURL,
			cfg.ML.ProcessPath,
			time.Duration(cfg.ML.TimeoutSeconds)*time.Second,
		)
		deps.ML = mlc
	}

	return deps, nil
}
