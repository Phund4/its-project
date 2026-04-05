// Package config загружает YAML-конфигурацию и применяет значения по умолчанию.
package config

import (
	"fmt"
	"os"

	"data-ingestion/internal/utils"

	"gopkg.in/yaml.v3"
)

type S3 struct {
	Endpoint string `yaml:"endpoint"`
	Bucket   string `yaml:"bucket"`
	Prefix   string `yaml:"prefix"`
	Region   string `yaml:"region"`
}

type ML struct {
	BaseURL        string `yaml:"base_url"`
	ProcessPath    string `yaml:"process_path"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type Ingest struct {
	TargetFPS             float64 `yaml:"target_fps"`
	CreateBucketIfMissing bool    `yaml:"create_bucket_if_missing"`
	FFmpegPath            string  `yaml:"ffmpeg_path"`
}

type Metrics struct {
	ListenAddr string `yaml:"listen_addr"`
}

type Camera struct {
	SegmentID string `yaml:"segment_id"`
	CameraID  string `yaml:"camera_id"`
	RTSPURL   string `yaml:"rtsp_url"`
}

type Root struct {
	S3      S3       `yaml:"s3"`
	ML      ML       `yaml:"ml"`
	Ingest  Ingest   `yaml:"ingest"`
	Metrics Metrics  `yaml:"metrics"`
	Cameras []Camera `yaml:"cameras"`
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
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// LoadFromEnv подгружает .env (ENV_FILE или .env), затем читает CONFIG_PATH или config.yaml.
func LoadFromEnv() (*Root, error) {
	envPath := os.Getenv("ENV_FILE")
	if envPath == "" {
		envPath = ".env"
	}
	if err := utils.LoadDotEnv(envPath); err != nil {
		return nil, err
	}

	p := os.Getenv("CONFIG_PATH")
	if p == "" {
		p = "config.yaml"
	}
	return Load(p)
}

// validate проверяет обязательные поля и подставляет значения по умолчанию.
func (c *Root) validate() error {
	if c.S3.Endpoint == "" {
		return fmt.Errorf("s3.endpoint is required")
	}
	if c.S3.Bucket == "" {
		return fmt.Errorf("s3.bucket is required")
	}
	if c.ML.BaseURL == "" {
		return fmt.Errorf("ml.base_url is required")
	}
	if c.ML.ProcessPath == "" {
		c.ML.ProcessPath = "/v1/process"
	}
	if c.ML.TimeoutSeconds <= 0 {
		c.ML.TimeoutSeconds = 30
	}
	if c.Ingest.TargetFPS <= 0 {
		c.Ingest.TargetFPS = 3
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
	if len(c.Cameras) == 0 {
		return fmt.Errorf("cameras: at least one camera is required")
	}
	for i, cam := range c.Cameras {
		if cam.SegmentID == "" || cam.CameraID == "" || cam.RTSPURL == "" {
			return fmt.Errorf("cameras[%d]: segment_id, camera_id, rtsp_url are required", i)
		}
	}
	return nil
}
