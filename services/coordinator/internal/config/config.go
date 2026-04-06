package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"traffic-coordinator/internal/core/domain"
)

type Root struct {
	ListenAddr          string
	HeartbeatTimeoutSec int
	Sources             []domain.Source
}

type fileRoot struct {
	Sources []domain.Source `yaml:"sources"`
}

func LoadFromEnv() (*Root, error) {
	listen := strings.TrimSpace(os.Getenv("LISTEN_ADDR"))
	if listen == "" {
		listen = ":8098"
	}
	heartbeatTimeoutSec := 30
	if v := strings.TrimSpace(os.Getenv("HEARTBEAT_TIMEOUT_SEC")); v != "" {
		var parsed int
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil && parsed > 0 {
			heartbeatTimeoutSec = parsed
		}
	}

	path := strings.TrimSpace(os.Getenv("SOURCES_CONFIG_PATH"))
	if path == "" {
		path = "sources.yaml"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sources config: %w", err)
	}
	var fr fileRoot
	if err := yaml.Unmarshal(b, &fr); err != nil {
		return nil, fmt.Errorf("parse sources config: %w", err)
	}
	for i := range fr.Sources {
		if strings.TrimSpace(fr.Sources[i].SourceKind) == "" {
			fr.Sources[i].SourceKind = "camera"
		}
		fr.Sources[i].SourceKind = strings.ToLower(strings.TrimSpace(fr.Sources[i].SourceKind))
		if fr.Sources[i].SourceKind != "camera" && fr.Sources[i].SourceKind != "telemetry" {
			return nil, fmt.Errorf("sources[%d]: source_kind must be camera or telemetry", i)
		}
		// Источники по умолчанию активны; отключение можно добавить отдельным признаком при необходимости.
		fr.Sources[i].Enabled = true
		if fr.Sources[i].SourceID == "" {
			if fr.Sources[i].CameraID != "" {
				fr.Sources[i].SourceID = fr.Sources[i].CameraID
			} else {
				fr.Sources[i].SourceID = fr.Sources[i].MunicipalityID
			}
		}
		if fr.Sources[i].ZoneID == "" {
			return nil, fmt.Errorf("sources[%d]: zone_id is required", i)
		}
		if fr.Sources[i].ClusterID == "" || fr.Sources[i].InstanceID == "" {
			return nil, fmt.Errorf("sources[%d]: cluster_id and instance_id are required", i)
		}
		if (fr.Sources[i].ReserveClusterID == "") != (fr.Sources[i].ReserveInstanceID == "") {
			return nil, fmt.Errorf("sources[%d]: reserve_cluster_id and reserve_instance_id must be set together", i)
		}
		if fr.Sources[i].SourceKind == "camera" {
			if fr.Sources[i].SegmentID == "" || fr.Sources[i].CameraID == "" || fr.Sources[i].RTSPURL == "" {
				return nil, fmt.Errorf("sources[%d]: camera source requires segment_id, camera_id, rtsp_url", i)
			}
		}
		if fr.Sources[i].SourceKind == "telemetry" && fr.Sources[i].MunicipalityID == "" {
			return nil, fmt.Errorf("sources[%d]: telemetry source requires municipality_id", i)
		}
	}
	return &Root{
		ListenAddr:          listen,
		HeartbeatTimeoutSec: heartbeatTimeoutSec,
		Sources:             fr.Sources,
	}, nil
}
