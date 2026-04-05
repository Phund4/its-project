// Package constants содержит константы времени выполнения сервиса data_ingestion.
package constants

const (
	// FrameLogEveryN — логировать прогресс каждые N кадров.
	FrameLogEveryN uint64 = 30
	// MaxJPEGSize — верхняя граница размера одного JPEG-кадра (байты).
	MaxJPEGSize = 16 << 20
	// MJPEGScannerChunk — размер временного буфера чтения MJPEG.
	MJPEGScannerChunk = 32768
	// MetricsReadHeaderTimeoutSec — таймаут чтения заголовка HTTP для /metrics.
	MetricsReadHeaderTimeoutSec = 10
	// MetricsShutdownTimeoutSec — таймаут graceful shutdown HTTP /metrics.
	MetricsShutdownTimeoutSec = 10
	// ReconnectBackoffSec — пауза перед переподключением к источнику (сек).
	ReconnectBackoffSec = 1
	// SourceWaitLogIntervalSec — минимальный интервал между одинаковыми WARN по камере (сек).
	SourceWaitLogIntervalSec = 45
)
