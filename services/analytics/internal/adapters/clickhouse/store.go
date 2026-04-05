// Package clickhouse — клиент и схема таблиц в ClickHouse.
package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	chgo "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"traffic-analytics/internal/adapters/metrics"
)

// Store держит соединение и имена таблиц в выбранной базе.
type Store struct {
	// conn активное соединение ClickHouse native.
	conn driver.Conn

	// database имя БД для road_incidents / road_congestion.
	database string

	// incidentsTable имя таблицы инцидентов.
	incidentsTable string

	// congestionTable имя таблицы загруженности.
	congestionTable string
}

// New открывает соединение, ping, создаёт таблицы при необходимости.
func New(ctx context.Context, addr, database, user, password, incidentsTable, congestionTable string) (*Store, error) {
	if incidentsTable == "" {
		incidentsTable = "road_incidents"
	}
	if congestionTable == "" {
		congestionTable = "road_congestion"
	}
	opts := &chgo.Options{
		Addr: []string{addr},
		Auth: chgo.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		DialTimeout: 5 * time.Second,
	}
	conn, err := chgo.Open(opts)
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

// NormalizeAddr убирает схему из адреса для Options.Addr (host:port).
func NormalizeAddr(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "tcp://")
	return raw
}

func quoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// MunicipalityRow строка справочника municipalities.
type MunicipalityRow struct {
	// MunicipalityID код города (msk, spb, …).
	MunicipalityID string

	// NameRU отображаемое имя.
	NameRU string

	// CenterLat широта центра карты.
	CenterLat float64

	// CenterLon долгота центра карты.
	CenterLon float64

	// DefaultZoom зум по умолчанию.
	DefaultZoom uint8
}

// ListInfraMunicipalities читает справочник городов из указанной БД (обычно its_infra_sim).
func (s *Store) ListInfraMunicipalities(ctx context.Context, infraDatabase string) ([]MunicipalityRow, error) {
	if infraDatabase == "" {
		infraDatabase = "its_infra_sim"
	}
	q := fmt.Sprintf(`
SELECT municipality_id, name_ru, center_lat, center_lon, default_zoom
FROM %s.municipalities
ORDER BY municipality_id
`, quoteIdent(infraDatabase))
	rows, err := s.conn.Query(ctx, q)
	if err != nil {
		metrics.ClickHouseErrors.WithLabelValues("infra_municipalities").Inc()
		return nil, err
	}
	defer rows.Close()
	var out []MunicipalityRow
	for rows.Next() {
		var m MunicipalityRow
		if err := rows.Scan(&m.MunicipalityID, &m.NameRU, &m.CenterLat, &m.CenterLon, &m.DefaultZoom); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// StopRow проекция остановки для карты.
type StopRow struct {
	// StopID UUID строкой.
	StopID string

	// StopCode код НСИ.
	StopCode string

	// Name полное имя.
	Name string

	// NameShort краткое имя.
	NameShort string

	// Lat широта WGS84.
	Lat float64

	// Lon долгота WGS84.
	Lon float64
}

// ListInfraStops — остановки по municipality_id.
func (s *Store) ListInfraStops(ctx context.Context, infraDatabase, municipalityID string) ([]StopRow, error) {
	if infraDatabase == "" {
		infraDatabase = "its_infra_sim"
	}
	esc := strings.ReplaceAll(municipalityID, "'", "''")
	q := fmt.Sprintf(`
SELECT toString(stop_id), stop_code, name, name_short, lat, lon
FROM %s.bus_stops
WHERE municipality_id = '%s'
  AND valid_from <= today()
  AND (valid_to IS NULL OR valid_to >= today())
ORDER BY stop_code
`, quoteIdent(infraDatabase), esc)
	rows, err := s.conn.Query(ctx, q)
	if err != nil {
		metrics.ClickHouseErrors.WithLabelValues("infra_stops").Inc()
		return nil, err
	}
	defer rows.Close()
	var out []StopRow
	for rows.Next() {
		var r StopRow
		if err := rows.Scan(&r.StopID, &r.StopCode, &r.Name, &r.NameShort, &r.Lat, &r.Lon); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
