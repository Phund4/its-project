// Package kafka — консьюмер событий ingest из Kafka (топики видео/ML и телеметрии).
package kafka

import (
	"context"
	"log/slog"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"traffic-analytics/internal/adapters/metrics"
	"traffic-analytics/internal/config"
	"traffic-analytics/internal/core/services"
)

func splitBrokers(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// RunIngestConsumer читает KAFKA_TOPIC_VIDEO и KAFKA_TOPIC_TELEMETRY до отмены ctx.
func RunIngestConsumer(ctx context.Context, ingest *services.IngestService, cfg config.Config) {
	brokers := splitBrokers(cfg.KafkaBootstrap)
	if len(brokers) == 0 {
		return
	}
	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:     brokers,
		GroupID:     cfg.KafkaConsumerGroup,
		GroupTopics: []string{cfg.KafkaTopicVideo, cfg.KafkaTopicTelemetry},
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     2 * time.Second,
	})
	defer func() {
		if err := r.Close(); err != nil {
			slog.Warn("kafka reader close", "err", err)
		}
	}()
	slog.Info("analytics kafka consumer",
		"brokers", brokers,
		"group", cfg.KafkaConsumerGroup,
		"topics", []string{cfg.KafkaTopicVideo, cfg.KafkaTopicTelemetry},
	)
	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			metrics.KafkaConsumeErrors.WithLabelValues("read").Inc()
			slog.Warn("kafka read", "err", err)
			time.Sleep(time.Second)
			continue
		}
		if err := ingest.ProcessIngest(ctx, m.Value); err != nil {
			metrics.KafkaConsumeErrors.WithLabelValues("process").Inc()
			slog.Warn("kafka ingest process", "topic", m.Topic, "err", err)
		} else {
			metrics.KafkaIngestProcessed.WithLabelValues(m.Topic).Inc()
		}
	}
}
