// Package chstore — адаптер ClickHouse.
package chstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"traffic-analytics/internal/adapters/metrics"
)

type Store struct {
	conn            driver.Conn
	database        string
	incidentsTable  string
	congestionTable string
}

func New(ctx context.Context, addr, database, user, password, incidentsTable, congestionTable string) (*Store, error) {
	if incidentsTable == "" {
		incidentsTable = "road_incidents"
	}
	if congestionTable == "" {
		congestionTable = "road_congestion"
	}
	opts := &clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		DialTimeout: 5 * time.Second,
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	s := &Store{
		conn:            conn,
		database:        database,
		incidentsTable:  incidentsTable,
		congestionTable: congestionTable,
	}
	if err := s.initSchema(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.conn.Close()
}

func (s *Store) initSchema(ctx context.Context) error {
	incQ := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.%s
(
    observed_at DateTime64(3, 'UTC'),
    segment_id LowCardinality(String),
    camera_id LowCardinality(String),
    s3_key String,
    crash_probability Float64,
    incident_label LowCardinality(String),
    raw_ml String CODEC(ZSTD(3))
)
ENGINE = MergeTree
ORDER BY (segment_id, camera_id, observed_at)
`, s.database, s.incidentsTable)
	if err := s.conn.Exec(ctx, incQ); err != nil {
		return fmt.Errorf("incidents schema: %w", err)
	}

	congQ := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.%s
(
    observed_at DateTime64(3, 'UTC'),
    segment_id LowCardinality(String),
    camera_id LowCardinality(String),
    s3_key String,
    congestion_score Float64,
    raw_ml String CODEC(ZSTD(3))
)
ENGINE = MergeTree
ORDER BY (segment_id, camera_id, observed_at)
`, s.database, s.congestionTable)
	if err := s.conn.Exec(ctx, congQ); err != nil {
		return fmt.Errorf("congestion schema: %w", err)
	}
	return nil
}

func (s *Store) InsertIncident(ctx context.Context, observedAt time.Time, segmentID, cameraID, s3Key string, crashProb float64, label, rawML string) error {
	q := fmt.Sprintf(`INSERT INTO %s.%s (observed_at, segment_id, camera_id, s3_key, crash_probability, incident_label, raw_ml) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.database, s.incidentsTable)
	batch, err := s.conn.PrepareBatch(ctx, q)
	if err != nil {
		metrics.ClickHouseErrors.WithLabelValues("prepare_incident").Inc()
		return err
	}
	if err := batch.Append(observedAt, segmentID, cameraID, s3Key, crashProb, label, rawML); err != nil {
		metrics.ClickHouseErrors.WithLabelValues("append_incident").Inc()
		return err
	}
	if err := batch.Send(); err != nil {
		metrics.ClickHouseErrors.WithLabelValues("send_incident").Inc()
		return err
	}
	return nil
}

func (s *Store) InsertCongestion(ctx context.Context, observedAt time.Time, segmentID, cameraID, s3Key string, congestionScore float64, rawML string) error {
	q := fmt.Sprintf(`INSERT INTO %s.%s (observed_at, segment_id, camera_id, s3_key, congestion_score, raw_ml) VALUES (?, ?, ?, ?, ?, ?)`,
		s.database, s.congestionTable)
	batch, err := s.conn.PrepareBatch(ctx, q)
	if err != nil {
		metrics.ClickHouseErrors.WithLabelValues("prepare_congestion").Inc()
		return err
	}
	if err := batch.Append(observedAt, segmentID, cameraID, s3Key, congestionScore, rawML); err != nil {
		metrics.ClickHouseErrors.WithLabelValues("append_congestion").Inc()
		return err
	}
	if err := batch.Send(); err != nil {
		metrics.ClickHouseErrors.WithLabelValues("send_congestion").Inc()
		return err
	}
	return nil
}

// FirstAddr убирает схему из адреса для clickhouse.Options.Addr (host:port).
func FirstAddr(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "tcp://")
	return raw
}
