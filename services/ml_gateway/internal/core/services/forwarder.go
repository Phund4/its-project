package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"ml-gateway/internal/adapters/metrics"
	"ml-gateway/internal/config"
)

// Forwarder пересылает проверенный JSON события дороги в analytics.
type Forwarder struct {
	cfg    config.Config
	client *http.Client
}

// NewForwarder собирает Forwarder с переданным HTTP-клиентом (таймауты из конфига).
func NewForwarder(cfg config.Config, client *http.Client) *Forwarder {
	return &Forwarder{cfg: cfg, client: client}
}

// AnalyticsURL возвращает полный URL ingest или пустую строку, если analytics не настроен.
func (f *Forwarder) AnalyticsURL() string {
	if f.cfg.AnalyticsBaseURL == "" {
		return ""
	}
	return f.cfg.AnalyticsBaseURL + f.cfg.AnalyticsIngestPath
}

// Forward отправляет body без изменений в POST analytics ingest.
func (f *Forwarder) Forward(ctx context.Context, body []byte) error {
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
