package services

// Лимиты и таймауты слоя сервисов ingest.
const (
	maxIngestBodyBytes     = 8 << 20
	maxLabelLen            = 128
	clickHouseQueryTimeout = 8 // секунд
)
