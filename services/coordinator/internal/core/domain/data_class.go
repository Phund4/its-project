package domain

// Классификатор источника: что приходит и куда ведёт пайплайн data-ingestion.
const (
	// DataClassRoadSegmentVideo — RTSP с камер дорожного участка → S3 + ML.
	DataClassRoadSegmentVideo = "road_segment_video"
	// DataClassVehicleBusTelemetry — телеметрия ТС (gRPC/HTTP) → analytics или Kafka.
	DataClassVehicleBusTelemetry = "vehicle_bus_telemetry"
)

// ValidDataClasses допустимые значения data_class в sources.yaml.
func ValidDataClasses() []string {
	return []string{
		DataClassRoadSegmentVideo,
		DataClassVehicleBusTelemetry,
	}
}
