package services

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"traffic-analytics/internal/adapters/metrics"
	"traffic-analytics/internal/config"
	"traffic-analytics/internal/constants"
	"traffic-analytics/internal/core/domain"
	"traffic-analytics/internal/utils"
)

// IngestService реализует use-case: правила домена, метрики Prometheus и запись в хранилище.
type IngestService struct {
	store        domain.EventStore
	cfg          config.Config
	appCtx       context.Context
	mu           sync.Mutex
	lastCongest  map[string]time.Time
	clickhouseTO time.Duration
}

// NewIngestService собирает сервис приёма; appCtx отменяется при shutdown процесса (сигнал).
func NewIngestService(store domain.EventStore, cfg config.Config, appCtx context.Context) *IngestService {
	return &IngestService{
		store:        store,
		cfg:          cfg,
		appCtx:       appCtx,
		lastCongest:  make(map[string]time.Time),
		clickhouseTO: time.Duration(constants.ClickHouseQueryTimeoutSec) * time.Second,
	}
}

// HandleIngest обрабатывает POST /v1/ingest: JSON, метрики, условные INSERT в ClickHouse.
func (s *IngestService) HandleIngest(w http.ResponseWriter, r *http.Request) {
	reqCtx, reqCancel := utils.WithAppShutdown(r.Context(), s.appCtx)
	defer reqCancel()

	body, err := io.ReadAll(io.LimitReader(r.Body, constants.MaxIngestBodyBytes))
	if err != nil {
		metrics.IngestErrors.WithLabelValues("read_body").Inc()
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	var ev domain.RoadEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		metrics.IngestErrors.WithLabelValues("json_decode").Inc()
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	seg := domain.SanitizeLabel(strings.TrimSpace(ev.SegmentID))
	cam := domain.SanitizeLabel(strings.TrimSpace(ev.CameraID))
	atStr := strings.TrimSpace(ev.ObservedAt)
	s3k := strings.TrimSpace(ev.S3Key)
	if seg == "" || cam == "" || atStr == "" {
		metrics.IngestErrors.WithLabelValues("validate").Inc()
		http.Error(w, "segment_id, camera_id, observed_at required", http.StatusBadRequest)
		return
	}

	var ml domain.MLParsed
	if len(ev.ML) > 0 && string(ev.ML) != "null" {
		if err := json.Unmarshal(ev.ML, &ml); err != nil {
			slog.Warn("ml parse", "err", err)
		}
	}

	crashP := ml.Incident.CrashProbability
	cong := ml.Congestion.CongestionScore
	lbl := strings.TrimSpace(ml.Incident.Label)

	metrics.CongestionScore.WithLabelValues(seg, cam).Set(cong)
	metrics.CrashProbability.WithLabelValues(seg, cam).Set(crashP)

	alert := strings.EqualFold(lbl, "crash") || crashP >= s.cfg.CrashAlertThreshold
	alertVal := 0.0
	if alert {
		alertVal = 1.0
	}
	metrics.CrashAlert.WithLabelValues(seg, cam).Set(alertVal)

	observedAt, err := domain.ParseObservedAt(atStr)
	if err != nil {
		observedAt = time.Now().UTC()
	}

	raw := string(ev.ML)
	if raw == "" {
		raw = "{}"
	}

	chCtx, chCancel := context.WithTimeout(reqCtx, s.clickhouseTO)
	defer chCancel()

	if s.shouldPersistCongestion(seg, cam) {
		if err := s.store.InsertCongestion(chCtx, observedAt, seg, cam, s3k, cong, raw); err != nil {
			slog.Warn("clickhouse congestion insert", "err", err)
		} else {
			s.markCongestionWritten(seg, cam)
			metrics.CongestionRecorded.WithLabelValues(seg, cam).Inc()
		}
	}

	if alert {
		if err := s.store.InsertIncident(chCtx, observedAt, seg, cam, s3k, crashP, lbl, raw); err != nil {
			slog.Warn("clickhouse incident insert", "err", err)
		} else {
			metrics.IncidentsRecorded.WithLabelValues(seg, cam).Inc()
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *IngestService) shouldPersistCongestion(seg, cam string) bool {
	if s.cfg.CongestionPersistInterval <= 0 {
		return true
	}
	now := time.Now()
	k := seg + "\x00" + cam
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.lastCongest[k]; ok && now.Sub(t) < s.cfg.CongestionPersistInterval {
		return false
	}
	return true
}

func (s *IngestService) markCongestionWritten(seg, cam string) {
	k := seg + "\x00" + cam
	s.mu.Lock()
	s.lastCongest[k] = time.Now()
	s.mu.Unlock()
}
