package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	analyticsclient "data-ingestion/internal/adapters/analytics"
	coordinatorclient "data-ingestion/internal/adapters/coordinator"
	kafkaadapter "data-ingestion/internal/adapters/kafka"
	mlclient "data-ingestion/internal/adapters/ml"
	s3store "data-ingestion/internal/adapters/s3"
	"data-ingestion/internal/adapters/telemetry"
	"data-ingestion/internal/adapters/telemetrygrpc"
	"data-ingestion/internal/adapters/telemetryhttp"
	"data-ingestion/internal/config"
)

// Deps адаптеры и конфигурация после инициализации.
type Deps struct {
	// Config загруженный YAML и дефолты.
	Config *config.Root

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

	// TelemetryHTTP сервер HTTP входа телеметрии.
	TelemetryHTTP *telemetryhttp.Server

	// TelemetryHTTPListenAddr адрес :port для HTTP телеметрии.
	TelemetryHTTPListenAddr string

	// Coordinator API клиент для динамических назначений источников.
	Coordinator *coordinatorclient.Client
}

// InitializeDependencies загружает конфиг и при необходимости S3, ML, клиент analytics.
func InitializeDependencies(ctx context.Context) (*Deps, error) {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	base := config.CoordinatorBaseURLFromEnv()
	if base == "" {
		return nil, fmt.Errorf("set COORDINATOR_BASE_URL")
	}
	if config.CoordinatorZoneIDFromEnv() == "" || config.CoordinatorClusterIDFromEnv() == "" || config.CoordinatorInstanceIDFromEnv() == "" {
		return nil, fmt.Errorf("set COORDINATOR_ZONE_ID, COORDINATOR_CLUSTER_ID, COORDINATOR_INSTANCE_ID")
	}

	deps := &Deps{Config: cfg}
	deps.Coordinator = coordinatorclient.New(base, 10*time.Second)

	return deps, nil
}

func InitVideoPipeline(ctx context.Context, deps *Deps) error {
	ak := os.Getenv("AWS_ACCESS_KEY_ID")
	sk := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if ak == "" || sk == "" {
		return ErrMissingAWSCredentials
	}
	store, err := s3store.New(ctx, deps.Config.S3.Endpoint, deps.Config.S3.Region, deps.Config.S3.Bucket, ak, sk)
	if err != nil {
		return fmt.Errorf("s3 client: %w", err)
	}
	if deps.Config.Ingest.CreateBucketIfMissing {
		if err := store.EnsureBucket(ctx); err != nil {
			return fmt.Errorf("ensure bucket: %w", err)
		}
		slog.Info("bucket ok", "bucket", deps.Config.S3.Bucket)
	}
	deps.Store = store
	deps.ML = mlclient.New(
		deps.Config.ML.BaseURL,
		deps.Config.ML.ProcessPath,
		time.Duration(deps.Config.ML.TimeoutSeconds)*time.Second,
	)
	return nil
}

func InitTelemetryPipeline(deps *Deps) error {
	kb := config.KafkaBootstrapFromEnv()
	if kb != "" {
		topic := config.KafkaTopicTelemetryFromEnv()
		kp, err := kafkaadapter.NewTelemetryProducer(kb, topic)
		if err != nil {
			return fmt.Errorf("kafka telemetry producer: %w", err)
		}
		deps.TelemetryPublisher = kp
		slog.Info("telemetry to kafka", "topic", topic)
	} else {
		url := config.AnalyticsIngestURLFromEnv()
		if url == "" {
			return fmt.Errorf("set KAFKA_BOOTSTRAP_SERVERS or ANALYTICS_INGEST_URL for telemetry pipeline")
		}
		deps.TelemetryPublisher = analyticsclient.New(url)
	}
	deps.TelemetryGRPC = &telemetrygrpc.Server{Publisher: deps.TelemetryPublisher}
	deps.TelemetryListenAddr = config.TelemetryListenAddrFromEnv()
	deps.TelemetryHTTP = &telemetryhttp.Server{Publisher: deps.TelemetryPublisher}
	deps.TelemetryHTTPListenAddr = config.TelemetryHTTPListenAddrFromEnv()
	return nil
}
