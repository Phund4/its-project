package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client отправляет JSON в analytics /v1/ingest.
type Client struct {
	// ingestURL полный URL эндпоинта ingest.
	ingestURL string

	// http клиент с таймаутом.
	http *http.Client
}

// New создаёт клиент; ingestURL — полный URL, например http://127.0.0.1:8093/v1/ingest (задаётся через ANALYTICS_INGEST_URL).
func New(ingestURL string) *Client {
	return &Client{
		ingestURL: ingestURL,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// IngestBody тело для analytics (без поля ml).
type IngestBody struct {
	// SegmentID сегмент дороги или синтетический для телеметрии.
	SegmentID string `json:"segment_id"`

	// CameraID идентификатор камеры или vehicle_id для автобуса.
	CameraID string `json:"camera_id"`

	// ObservedAt время наблюдения RFC3339.
	ObservedAt string `json:"observed_at"`

	// S3Key ключ кадра в S3 при видео-контуре.
	S3Key string `json:"s3_key,omitempty"`

	// Telemetry произвольный JSON телеметрии.
	Telemetry json.RawMessage `json:"telemetry"`
}

// PublishIngestJSON выполняет POST сырого JSON (как PostIngest после marshal).
func (c *Client) PublishIngestJSON(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ingestURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("post ingest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("analytics ingest: status %s", resp.Status)
	}
	return nil
}

// PostIngest сериализует body и выполняет POST.
func (c *Client) PostIngest(ctx context.Context, body IngestBody) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal ingest: %w", err)
	}
	return c.PublishIngestJSON(ctx, b)
}
