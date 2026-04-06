// Package config загружает YAML-конфигурацию и применяет значения по умолчанию.
package config

import (
	"fmt"
	"math"
	"os"

	"gopkg.in/yaml.v3"
)

// S3 параметры объектного хранилища (MinIO-совместимый endpoint).
type S3 struct {
	// Endpoint URL API S3 без завершающего слэша.
	Endpoint string `yaml:"endpoint"`

	// Bucket имя бакета для кадров.
	Bucket string `yaml:"bucket"`

	// Prefix необязательный префикс ключей.
	Prefix string `yaml:"prefix"`

	// Region регион для подписи запросов AWS SDK.
	Region string `yaml:"region"`
}

// ML HTTP-сервис инференса по кадру.
type ML struct {
	// BaseURL корень сервиса ML.
	BaseURL string `yaml:"base_url"`

	// ProcessPath путь multipart-обработки (например /v1/process).
	ProcessPath string `yaml:"process_path"`

	// TimeoutSeconds таймаут HTTP-запроса к ML.
	TimeoutSeconds int `yaml:"timeout_seconds"`
}

// Ingest поведение захвата и выгрузки кадров.
type Ingest struct {
	// TargetFPS целевой FPS для ffmpeg при дискретизации потока.
	TargetFPS float64 `yaml:"target_fps"`

	// CreateBucketIfMissing создать bucket при старте, если нет.
	CreateBucketIfMissing bool `yaml:"create_bucket_if_missing"`

	// FFmpegPath исполняемый файл ffmpeg.
	FFmpegPath string `yaml:"ffmpeg_path"`

	// ProcessWorkers параллельных обработчиков кадра (S3 + ML) на одну камеру.
	// Если 0 — по умолчанию ceil(target_fps), чтобы скорость обработки могла совпасть с дискретизацией ffmpeg.
	ProcessWorkers int `yaml:"process_workers"`
}

// Metrics экспорт Prometheus.
type Metrics struct {
	// ListenAddr адрес HTTP :port для /metrics.
	ListenAddr string `yaml:"listen_addr"`
}

// Camera один RTSP-источник в конфиге.
type Camera struct {
	// SegmentID сегмент дороги для метаданных и analytics.
	SegmentID string `yaml:"segment_id"`

	// CameraID идентификатор камеры.
	CameraID string `yaml:"camera_id"`

	// RTSPURL URL потока для ffmpeg.
	RTSPURL string `yaml:"rtsp_url"`
}

// Root корневая конфигурация YAML.
type Root struct {
	// S3 настройки хранилища.
	S3 S3 `yaml:"s3"`

	// ML настройки сервиса обработки кадров.
	ML ML `yaml:"ml"`

	// Ingest параметры пайплайна кадров.
	Ingest Ingest `yaml:"ingest"`

	// Metrics адрес метрик.
	Metrics Metrics `yaml:"metrics"`

	// Cameras список камер (контур RTSP).
	Cameras []Camera `yaml:"cameras"`

	// ConfigFile путь к загруженному YAML (не из файла).
	ConfigFile string `yaml:"-"`
}

// Load читает YAML по пути path, парсит и валидирует структуру Root.
func Load(path string) (*Root, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Root
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("yaml: %w", err)
	}
	c.ConfigFile = path
	ApplyEnvOverrides(&c)
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// LoadFromEnv подгружает .env (ENV_FILE или .env), затем CONFIG_PATH или config.telemetry.yaml по умолчанию.
func LoadFromEnv() (*Root, error) {
	envPath := os.Getenv("ENV_FILE")
	if envPath == "" {
		envPath = ".env"
	}
	if err := tryLoadDotEnv(); err != nil {
		return nil, err
	}

	p := os.Getenv("CONFIG_PATH")
	if p == "" {
		p = "config.telemetry.yaml"
	}
	return Load(p)
}

// validate проверяет базовую конфигурацию и подставляет значения по умолчанию.
func (c *Root) validate() error {
	if c.ML.ProcessPath == "" {
		c.ML.ProcessPath = "/v1/process"
	}
	if c.ML.TimeoutSeconds <= 0 {
		c.ML.TimeoutSeconds = 30
	}
	if c.Ingest.TargetFPS <= 0 {
		c.Ingest.TargetFPS = 3
	}
	if c.Ingest.ProcessWorkers <= 0 {
		c.Ingest.ProcessWorkers = int(math.Ceil(c.Ingest.TargetFPS))
	}
	if c.Ingest.ProcessWorkers < 1 {
		c.Ingest.ProcessWorkers = 1
	}
	if c.Ingest.ProcessWorkers > 64 {
		c.Ingest.ProcessWorkers = 64
	}
	if c.Ingest.FFmpegPath == "" {
		c.Ingest.FFmpegPath = "ffmpeg"
	}
	if c.Metrics.ListenAddr == "" {
		c.Metrics.ListenAddr = ":9091"
	}
	if c.S3.Region == "" {
		c.S3.Region = "us-east-1"
	}
	return nil
}
