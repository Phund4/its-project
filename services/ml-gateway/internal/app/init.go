package app

import (
	"net/http"
	"strings"

	kafkago "github.com/segmentio/kafka-go"

	"ml-gateway/internal/config"
	"ml-gateway/internal/core/services"
)

// Deps конфигурация и сервис пересылки после инициализации.
type Deps struct {
	// Config снимок окружения.
	Config config.Config

	// Forwarder POST тела в analytics ingest.
	Forwarder *services.Forwarder

	// HTTPClient клиент с AnalyticsTimeout.
	HTTPClient *http.Client
}

func splitKafkaBrokers(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// InitializeDependencies создаёт HTTP-клиент и Forwarder из конфигурации.
func InitializeDependencies() *Deps {
	cfg := config.Load()
	client := &http.Client{Timeout: cfg.AnalyticsTimeout}
	var kw *kafkago.Writer
	if cfg.KafkaBootstrap != "" {
		brokers := splitKafkaBrokers(cfg.KafkaBootstrap)
		if len(brokers) > 0 {
			kw = &kafkago.Writer{
				Addr:                   kafkago.TCP(brokers...),
				Topic:                  cfg.KafkaTopicVideo,
				AllowAutoTopicCreation: true,
				RequiredAcks:           kafkago.RequireOne,
			}
		}
	}
	fwd := services.NewForwarder(cfg, client, kw)
	return &Deps{
		Config:     cfg,
		Forwarder:  fwd,
		HTTPClient: client,
	}
}
