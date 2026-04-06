package domain

import "time"

// Source описывает один входной источник данных зоны.
type Source struct {
	SourceID       string `yaml:"source_id" json:"source_id"`
	SourceKind     string `yaml:"source_kind" json:"source_kind"` // camera | telemetry
	ZoneID         string `yaml:"zone_id" json:"zone_id"`
	SegmentID      string `yaml:"segment_id" json:"segment_id"`
	CameraID       string `yaml:"camera_id" json:"camera_id"`
	RTSPURL        string `yaml:"rtsp_url" json:"rtsp_url"`
	MunicipalityID string `yaml:"municipality_id" json:"municipality_id"`
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	ClusterID      string `yaml:"cluster_id" json:"cluster_id"`
	InstanceID     string `yaml:"instance_id" json:"instance_id"`
	ReserveClusterID  string `yaml:"reserve_cluster_id" json:"reserve_cluster_id"`
	ReserveInstanceID string `yaml:"reserve_instance_id" json:"reserve_instance_id"`
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
