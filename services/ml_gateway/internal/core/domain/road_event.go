package domain

import "encoding/json"

// RoadEvent соответствует JSON, который ML шлёт в ml_gateway / analytics ingest.
type RoadEvent struct {
	SegmentID  string          `json:"segment_id"`
	CameraID   string          `json:"camera_id"`
	ObservedAt string          `json:"observed_at"`
	S3Key      string          `json:"s3_key"`
	ML         json.RawMessage `json:"ml"`
}
