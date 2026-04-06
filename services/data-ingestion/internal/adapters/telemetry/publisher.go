package telemetry

import "context"

// Publisher доставляет JSON тела `POST /v1/ingest` в analytics (HTTP или Kafka).
type Publisher interface {
	PublishIngestJSON(ctx context.Context, payload []byte) error
}
