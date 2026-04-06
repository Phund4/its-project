package kafka

import (
	"context"
	"fmt"
	"strings"

	kafkago "github.com/segmentio/kafka-go"
)

// TelemetryProducer пишет JSON ingest в топик телеметрии.
type TelemetryProducer struct {
	w *kafkago.Writer
}

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

// NewTelemetryProducer создаёт продюсер; bootstrap — KAFKA_BOOTSTRAP_SERVERS.
func NewTelemetryProducer(bootstrap, topic string) (*TelemetryProducer, error) {
	brokers := splitBrokers(bootstrap)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("empty KAFKA_BOOTSTRAP_SERVERS")
	}
	return &TelemetryProducer{
		w: &kafkago.Writer{
			Addr:                   kafkago.TCP(brokers...),
			Topic:                  topic,
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafkago.RequireOne,
		},
	}, nil
}

// PublishIngestJSON реализует telemetry.Publisher.
func (p *TelemetryProducer) PublishIngestJSON(ctx context.Context, payload []byte) error {
	return p.w.WriteMessages(ctx, kafkago.Message{Value: payload})
}
