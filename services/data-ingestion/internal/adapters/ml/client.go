package mlclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"data-ingestion/internal/core/domain"
)

// Client HTTP-клиент к сервису машинного обучения.
type Client struct {
	// base корневой URL ML без завершающего слэша.
	base string

	// processPath путь POST multipart (например /v1/process).
	processPath string

	// cli HTTP с таймаутом из конструктора.
	cli *http.Client
}

// New создаёт клиент с baseURL, путём process и таймаутом HTTP.
func New(baseURL, processPath string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		base:        baseURL,
		processPath: processPath,
		cli:         &http.Client{Timeout: timeout},
	}
}

// PostProcess отправляет JPEG-кадр в ML. При шлюзе с 204 тело ответа пустое.
func (c *Client) PostProcess(ctx context.Context, jpeg []byte, filename string, meta domain.ProcessMeta) error {
	return c.postMultipart(ctx, c.processPath, jpeg, filename, meta)
}

func (c *Client) postMultipart(ctx context.Context, path string, jpeg []byte, filename string, meta domain.ProcessMeta) error {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("image", filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(jpeg); err != nil {
		return err
	}
	if meta.SegmentID != "" {
		_ = w.WriteField("segment_id", meta.SegmentID)
	}
	if meta.CameraID != "" {
		_ = w.WriteField("camera_id", meta.CameraID)
	}
	if meta.S3Key != "" {
		_ = w.WriteField("s3_key", meta.S3Key)
	}
	if meta.ObservedAt != "" {
		_ = w.WriteField("observed_at", meta.ObservedAt)
	}
	if err := w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := c.cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
