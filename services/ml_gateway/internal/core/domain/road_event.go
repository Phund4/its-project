package domain

import "encoding/json"

// RoadEvent соответствует JSON, который ML шлёт в ml_gateway / analytics ingest.
type RoadEvent struct {
	// SegmentID сегмент дороги.
	SegmentID string `json:"segment_id"`

	// CameraID идентификатор камеры.
	CameraID string `json:"camera_id"`

	// ObservedAt время наблюдения RFC3339.
	ObservedAt string `json:"observed_at"`

	// S3Key ключ кадра в S3.
	S3Key string `json:"s3_key"`

	// ML сырой JSON ответа ML.
	ML json.RawMessage `json:"ml"`
}
