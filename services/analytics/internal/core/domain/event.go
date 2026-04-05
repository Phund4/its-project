package domain

import (
	"encoding/json"
	"strings"
	"time"

	"traffic-analytics/internal/constants"
)

// RoadEvent — входящий JSON конверт для /v1/ingest.
type RoadEvent struct {
	SegmentID  string          `json:"segment_id"`
	CameraID   string          `json:"camera_id"`
	ObservedAt string          `json:"observed_at"`
	S3Key      string          `json:"s3_key"`
	ML         json.RawMessage `json:"ml"`
}

// MLParsed — вложенный объект ml для метрик и записи в БД.
type MLParsed struct {
	Incident struct {
		CrashProbability float64 `json:"crash_probability"`
		Label            string  `json:"label"`
	} `json:"incident"`
	Congestion struct {
		CongestionScore float64 `json:"congestion_score"`
	} `json:"congestion"`
}

// SanitizeLabel ограничивает длину строки для меток Prometheus.
func SanitizeLabel(s string) string {
	max := constants.MaxLabelLen
	if len(s) > max {
		return s[:max]
	}
	return s
}

// ParseObservedAt разбирает RFC3339 / RFC3339Nano из полей ingest/ML.
func ParseObservedAt(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	return time.Parse(time.RFC3339, s)
}
