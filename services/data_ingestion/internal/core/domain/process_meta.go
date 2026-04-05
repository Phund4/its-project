package domain

// ProcessMeta — метаданные кадра для multipart-запроса к ML.
type ProcessMeta struct {
	SegmentID  string
	CameraID   string
	S3Key      string
	ObservedAt string
}
