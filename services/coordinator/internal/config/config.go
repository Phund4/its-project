package config

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"traffic-coordinator/internal/core/domain"
)

type Root struct {
	ListenAddr          string
	HeartbeatTimeoutSec int
	DatabaseURL         string
	Sources             []domain.Source
	ZoneWorkers         map[string][]domain.Replica
}

type sourcesFileRoot struct {
	Sources []domain.Source `yaml:"sources"`
}

type ingestionInstancesFileRoot struct {
	ZoneWorkers map[string][]domain.Replica `yaml:"zone_workers"`
}

func LoadFromEnv() (*Root, error) {
	if err := tryLoadDotEnv(); err != nil {
		return nil, err
	}
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
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))

	sourcesPath := strings.TrimSpace(os.Getenv("SOURCES_CONFIG_PATH"))
	if sourcesPath == "" {
		sourcesPath = "sources.yaml"
	}
	sb, err := os.ReadFile(sourcesPath)
	if err != nil {
		return nil, fmt.Errorf("read sources config %q: %w", sourcesPath, err)
	}
	var sfr sourcesFileRoot
	if err := yaml.Unmarshal(sb, &sfr); err != nil {
		return nil, fmt.Errorf("parse sources config %q: %w", sourcesPath, err)
	}

	instPath := strings.TrimSpace(os.Getenv("INGESTION_INSTANCES_PATH"))
	if instPath == "" {
		instPath = "ingestion_instances.yaml"
	}
	ib, err := os.ReadFile(instPath)
	if err != nil {
		return nil, fmt.Errorf("read ingestion instances %q: %w", instPath, err)
	}
	var ifr ingestionInstancesFileRoot
	if err := yaml.Unmarshal(ib, &ifr); err != nil {
		return nil, fmt.Errorf("parse ingestion instances %q: %w", instPath, err)
	}

	fr := struct {
		Sources     []domain.Source
		ZoneWorkers map[string][]domain.Replica
	}{
		Sources:     sfr.Sources,
		ZoneWorkers: ifr.ZoneWorkers,
	}

	validClasses := domain.ValidDataClasses()
	for i := range fr.Sources {
		dc := strings.ToLower(strings.TrimSpace(fr.Sources[i].DataClass))
		fr.Sources[i].DataClass = dc
		if dc == "" {
			return nil, fmt.Errorf("sources[%d]: data_class is required (one of: %s)", i, strings.Join(validClasses, ", "))
		}
		if !slices.Contains(validClasses, dc) {
			return nil, fmt.Errorf("sources[%d]: data_class %q must be one of: %s", i, dc, strings.Join(validClasses, ", "))
		}
		fr.Sources[i].Enabled = true
		if fr.Sources[i].SourceID == "" {
			switch fr.Sources[i].DataClass {
			case domain.DataClassRoadSegmentVideo:
				if fr.Sources[i].CameraID != "" {
					fr.Sources[i].SourceID = fr.Sources[i].CameraID
				}
			case domain.DataClassVehicleBusTelemetry:
				fr.Sources[i].SourceID = "telemetry-" + fr.Sources[i].ZoneID
			}
		}
		if fr.Sources[i].ZoneID == "" {
			return nil, fmt.Errorf("sources[%d]: zone_id is required", i)
		}
		switch fr.Sources[i].DataClass {
		case domain.DataClassRoadSegmentVideo:
			if fr.Sources[i].SegmentID == "" || fr.Sources[i].CameraID == "" || fr.Sources[i].RTSPURL == "" {
				return nil, fmt.Errorf("sources[%d]: road_segment_video requires segment_id, camera_id, rtsp_url", i)
			}
		case domain.DataClassVehicleBusTelemetry:
			// только зона и source_id (по умолчанию telemetry-<zone>)
		}
	}
	if fr.ZoneWorkers == nil {
		fr.ZoneWorkers = make(map[string][]domain.Replica)
	}
	zoneIDs := make(map[string]struct{})
	for _, s := range fr.Sources {
		zoneIDs[s.ZoneID] = struct{}{}
	}
	for zid := range zoneIDs {
		pool := fr.ZoneWorkers[zid]
		if len(pool) == 0 {
			return nil, fmt.Errorf("zone_workers[%s] must list at least one cluster_id/instance_id", zid)
		}
		for j := range pool {
			r := &fr.ZoneWorkers[zid][j]
			if strings.TrimSpace(r.ClusterID) == "" || strings.TrimSpace(r.InstanceID) == "" {
				return nil, fmt.Errorf("zone_workers[%s][%d]: cluster_id and instance_id are required", zid, j)
			}
			r.URL = strings.TrimSpace(r.URL)
		}
	}
	return &Root{
		ListenAddr:          listen,
		HeartbeatTimeoutSec: heartbeatTimeoutSec,
		DatabaseURL:         databaseURL,
		Sources:             fr.Sources,
		ZoneWorkers:         fr.ZoneWorkers,
	}, nil
}
