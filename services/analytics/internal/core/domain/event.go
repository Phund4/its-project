package domain

import "encoding/json"

// RoadEvent входящий JSON для POST /v1/ingest.
type RoadEvent struct {
	// SegmentID логический сегмент дороги / линии.
	SegmentID string `json:"segment_id"`

	// CameraID идентификатор источника (камера или «виртуальный» для телеметрии).
	CameraID string `json:"camera_id"`

	// ObservedAt время события RFC3339.
	ObservedAt string `json:"observed_at"`

	// S3Key ключ кадра в S3 при видео-контуре.
	S3Key string `json:"s3_key,omitempty"`

	// ML сырой JSON ответа ML (инцидент/загруженность).
	ML json.RawMessage `json:"ml,omitempty"`

	// Telemetry сырой JSON телеметрии ТС (без ML).
	Telemetry json.RawMessage `json:"telemetry,omitempty"`
}

// IncidentBlock поддерево ml.incident.
type IncidentBlock struct {
	// CrashProbability оценка вероятности ДТП [0, 1].
	CrashProbability float64 `json:"crash_probability"`

	// Label класс события (например crash).
	Label string `json:"label"`
}

// CongestionBlock поддерево ml.congestion.
type CongestionBlock struct {
	// CongestionScore степень загруженности [0, 1].
	CongestionScore float64 `json:"congestion_score"`
}

// MLParsed разбор поля ml для метрик и записи в ClickHouse.
type MLParsed struct {
	// Incident блок инцидента.
	Incident IncidentBlock `json:"incident"`

	// Congestion блок загруженности.
	Congestion CongestionBlock `json:"congestion"`
}
