CREATE TABLE IF NOT EXISTS sources (
    source_id TEXT PRIMARY KEY,
    data_class TEXT NOT NULL,
    zone_id TEXT NOT NULL,
    segment_id TEXT NOT NULL DEFAULT '',
    camera_id TEXT NOT NULL DEFAULT '',
    rtsp_url TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_sources_zone ON sources(zone_id);

CREATE TABLE IF NOT EXISTS ingestion_instances (
    zone_id TEXT NOT NULL,
    cluster_id TEXT NOT NULL,
    instance_id TEXT NOT NULL,
    url TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (zone_id, cluster_id, instance_id)
);

CREATE INDEX IF NOT EXISTS idx_ingestion_instances_zone ON ingestion_instances(zone_id);

CREATE TABLE IF NOT EXISTS worker_heartbeats (
    zone_id TEXT NOT NULL,
    cluster_id TEXT NOT NULL,
    instance_id TEXT NOT NULL,
    load DOUBLE PRECISION NOT NULL DEFAULT 0,
    assignments INTEGER NOT NULL DEFAULT 0,
    observed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (zone_id, cluster_id, instance_id)
);

