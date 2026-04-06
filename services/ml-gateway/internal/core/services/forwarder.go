package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	kafkago "github.com/segmentio/kafka-go"

	"ml-gateway/internal/adapters/metrics"
	"ml-gateway/internal/config"
)

// Forwarder пересылает JSON события дороги в analytics по HTTP или в Kafka.
type Forwarder struct {
	// cfg базовый URL и путь ingest.
	cfg config.Config

	// client HTTP с таймаутом.
	client *http.Client

	// kafkaWriter не nil — писать в Kafka (топик из writer), иначе HTTP.
	kafkaWriter *kafkago.Writer
}

// NewForwarder создаёт Forwarder; kafkaWriter может быть nil.
func NewForwarder(cfg config.Config, client *http.Client, kafkaWriter *kafkago.Writer) *Forwarder {
	return &Forwarder{cfg: cfg, client: client, kafkaWriter: kafkaWriter}
}

// HasDestination true если настроен HTTP ingest или Kafka.
func (f *Forwarder) HasDestination() bool {
	return f.AnalyticsURL() != "" || f.kafkaWriter != nil
}

// AnalyticsURL возвращает полный URL ingest или пустую строку.
func (f *Forwarder) AnalyticsURL() string {
	if f.cfg.AnalyticsBaseURL == "" {
		return ""
	}
	return f.cfg.AnalyticsBaseURL + f.cfg.AnalyticsIngestPath
}

// Forward отправляет body в Kafka или POST analytics ingest.
func (f *Forwarder) Forward(ctx context.Context, body []byte) error {
	if f.kafkaWriter != nil {
		return f.forwardKafka(ctx, body)
	}
	return f.forwardHTTP(ctx, body)
}

func (f *Forwarder) forwardKafka(ctx context.Context, body []byte) error {
	key := hashKey(body)
	err := f.kafkaWriter.WriteMessages(ctx, kafkago.Message{
		Key:   key,
		Value: body,
	})
	if err != nil {
		metrics.OperationErrors.WithLabelValues("kafka_write").Inc()
		return fmt.Errorf("kafka write: %w", err)
	}
	return nil
}

// hashKey стабильный ключ для партиционирования (segment_id+camera_id из JSON).
func hashKey(body []byte) []byte {
	seg := jsonStringField(body, "segment_id")
	cam := jsonStringField(body, "camera_id")
	if seg == "" && cam == "" {
		return nil
	}
	return []byte(seg + "\x00" + cam)
}

func jsonStringField(raw []byte, field string) string {
	pat := `"` + field + `":`
	i := bytes.Index(raw, []byte(pat))
	if i < 0 {
		return ""
	}
	rest := raw[i+len(pat):]
	rest = bytes.TrimLeft(rest, " \t\n\r")
	if len(rest) == 0 {
		return ""
	}
	if rest[0] == '"' {
		rest = rest[1:]
		j := bytes.IndexByte(rest, '"')
		if j >= 0 {
			return string(rest[:j])
		}
	}
	return ""
}

func (f *Forwarder) forwardHTTP(ctx context.Context, body []byte) error {
	url := f.AnalyticsURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		metrics.OperationErrors.WithLabelValues("forward_analytics").Inc()
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		metrics.OperationErrors.WithLabelValues("forward_analytics").Inc()
		return fmt.Errorf("analytics unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		metrics.OperationErrors.WithLabelValues("analytics_response").Inc()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("analytics HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}
