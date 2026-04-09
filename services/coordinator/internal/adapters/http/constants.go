package httpserver

const (
	workerStatusUpsertPath = "/v1/workers/status"
	workersPath            = "/v1/workers"
	ingestionInstancesPath = "/v1/ingestion_instances"
	healthPath             = "/health"
	sourcesPath            = "/v1/sources"
	assignmentsPath        = "/v1/assignments"

	healthStatusKey          = "status"
	healthStatusOK           = "ok"
	itemsKey                 = "items"
	errorKey                 = "error"
	sourcesErrorKey          = "error"
	sourcesZoneIDKey         = "zone_id"
	assignmentsZoneIDKey     = "zone_id"
	assignmentsClusterIDKey  = "cluster_id"
	assignmentsInstanceIDKey = "instance_id"
	assignmentsDataClassKey  = "data_class"

	zoneIDRequiredError                    = "zone_id is required"
	clusterIDRequiredError                 = "cluster_id is required"
	instanceIDRequiredError                = "instance_id is required"
	dataClassRequiredError                 = "data_class is required"
	invalidJSONError                       = "invalid json"
	zoneIDClusterIDInstanceIDRequiredError = "zone_id, cluster_id, instance_id are required"
)
