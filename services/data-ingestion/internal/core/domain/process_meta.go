package domain

// ProcessMeta метаданные кадра для multipart-запроса к ML.
type ProcessMeta struct {
	// SegmentID сегмент дороги.
	SegmentID string

	// CameraID идентификатор камеры.
	CameraID string

	// S3Key ключ объекта в бакете.
	S3Key string

	// ObservedAt время кадра для ML.
	ObservedAt string
}
