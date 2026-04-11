INSERT INTO sources (source_id, data_class, zone_id, segment_id, camera_id, rtsp_url, enabled) VALUES
('cam-01', 'road_segment_video', 'zone-a', 'ring-road-5', 'cam-01', 'rtsp://mediamtx:8554/cam-01', TRUE),
('cam-02', 'road_segment_video', 'zone-a', 'ring-road-5', 'cam-02', 'rtsp://mediamtx:8554/cam-02', TRUE),
('cam-03', 'road_segment_video', 'zone-a', 'ring-road-5', 'cam-03', 'rtsp://mediamtx:8554/cam-03', TRUE),
('cam-04', 'road_segment_video', 'zone-a', 'ring-road-5', 'cam-04', 'rtsp://mediamtx:8554/cam-04', TRUE),
('telemetry-zone-a', 'vehicle_bus_telemetry', 'zone-a', '', '', '', TRUE)
ON CONFLICT (source_id) DO UPDATE
SET data_class = EXCLUDED.data_class,
    zone_id = EXCLUDED.zone_id,
    segment_id = EXCLUDED.segment_id,
    camera_id = EXCLUDED.camera_id,
    rtsp_url = EXCLUDED.rtsp_url,
    enabled = EXCLUDED.enabled;

INSERT INTO ingestion_instances (zone_id, cluster_id, instance_id, url, enabled) VALUES
('zone-a', 'cluster-1', 'ingest-a1', 'http://data-ingestion:9091/metrics', TRUE),
('zone-a', 'cluster-1', 'ingest-a2', 'http://data-ingestion:9091/metrics', TRUE)
ON CONFLICT (zone_id, cluster_id, instance_id) DO UPDATE
SET url = EXCLUDED.url,
    enabled = EXCLUDED.enabled;

