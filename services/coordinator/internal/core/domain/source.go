package domain

import "time"

// Replica — инстанс data-ingestion в пуле зоны (порядок в zone_workers = tie-break при равной загрузке).
type Replica struct {
	ClusterID  string `yaml:"cluster_id" json:"cluster_id"`
	InstanceID string `yaml:"instance_id" json:"instance_id"`
	// URL необязательный: куда смотреть / подключаться (метрики, health и т.п.), только для людей и API каталога.
	URL string `yaml:"url,omitempty" json:"url,omitempty"`
}

// IngestionInstance строка из ingestion_instances.yaml с зоной (для GET /v1/ingestion_instances).
type IngestionInstance struct {
	ZoneID     string `json:"zone_id"`
	ClusterID  string `json:"cluster_id"`
	InstanceID string `json:"instance_id"`
	URL        string `json:"url,omitempty"`
}

// Source описывает один входной источник данных зоны.
type Source struct {
	SourceID  string `yaml:"source_id" json:"source_id"`
	DataClass string `yaml:"data_class" json:"data_class"` // см. константы DataClass* в data_class.go
	ZoneID    string `yaml:"zone_id" json:"zone_id"`
	// Поля ниже только для road_segment_video.
	SegmentID string `yaml:"segment_id,omitempty" json:"segment_id,omitempty"`
	CameraID  string `yaml:"camera_id,omitempty" json:"camera_id,omitempty"`
	RTSPURL   string `yaml:"rtsp_url,omitempty" json:"rtsp_url,omitempty"`
	Enabled   bool   `yaml:"enabled" json:"enabled"`
}

// WorkerHeartbeat статус живого ingestion-инстанса.
type WorkerHeartbeat struct {
	ZoneID      string    `json:"zone_id"`
	ClusterID   string    `json:"cluster_id"`
	InstanceID  string    `json:"instance_id"`
	Load        float64   `json:"load"`
	ObservedAt  time.Time `json:"observed_at"`
	Assignments int       `json:"assignments"`
}
