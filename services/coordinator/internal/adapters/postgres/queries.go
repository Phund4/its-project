package postgres

const (
	baseSourcesQuery        = `select source_id,data_class,zone_id,segment_id,camera_id,rtsp_url,enabled from sources where enabled=true`
	baseZoneWorkersQuery    = `select zone_id,cluster_id,instance_id,url from ingestion_instances where enabled=true`
	workerStatusesQuery     = `select zone_id,cluster_id,instance_id,load,assignments,observed_at from worker_statuses`
	upsertWorkerStatusQuery = `
		insert into worker_statuses(zone_id,cluster_id,instance_id,load,assignments,observed_at)
		values ($1,$2,$3,$4,$5,$6)
		on conflict (zone_id,cluster_id,instance_id)
		do update set load=excluded.load, assignments=excluded.assignments, observed_at=excluded.observed_at
	`
)

func buildSourcesQuery(zoneID string) (string, []any) {
	base := baseSourcesQuery
	args := []any{}

	if zoneID != "" {
		base += ` and zone_id=$1`
		args = append(args, zoneID)
	}

	return base, args
}

func buildZoneWorkersQuery(zoneID string) (string, []any) {
	base := baseZoneWorkersQuery
	args := []any{}

	if zoneID != "" {
		base += ` and zone_id=$1`
		args = append(args, zoneID)
	}

	return base, args
}
