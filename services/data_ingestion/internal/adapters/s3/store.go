package s3store

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client — обёртка над S3 API (MinIO-совместимый endpoint).
type Client struct {
	api    *s3.Client
	bucket string
}

// New создаёт клиент S3 для указанного endpoint, региона и bucket.
func New(ctx context.Context, endpoint, region, bucket, accessKey, secretKey string) (*Client, error) {
	endpoint = strings.TrimRight(endpoint, "/")
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	return &Client{api: client, bucket: bucket}, nil
}

// EnsureBucket создаёт bucket, если HeadBucket возвращает ошибку.
func (c *Client) EnsureBucket(ctx context.Context) error {
	_, err := c.api.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(c.bucket)})
	if err == nil {
		return nil
	}
	_, err = c.api.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(c.bucket)})
	if err != nil {
		return fmt.Errorf("create bucket %q: %w", c.bucket, err)
	}
	return nil
}

// JPEGBytesToPNG перекодирует байты JPEG в PNG.
func JPEGBytesToPNG(jpegBytes []byte) ([]byte, error) {
	img, err := jpeg.Decode(bytes.NewReader(jpegBytes))
	if err != nil {
		return nil, fmt.Errorf("jpeg decode: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("png encode: %w", err)
	}
	return buf.Bytes(), nil
}

// PutJPEG кладёт объект с Content-Type image/jpeg.
func (c *Client) PutJPEG(ctx context.Context, key string, body []byte) error {
	_, err := c.api.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("image/jpeg"),
	})
	if err != nil {
		return fmt.Errorf("put s3://%s/%s: %w", c.bucket, key, err)
	}
	return nil
}

// PutPNG кладёт объект с Content-Type image/png.
func (c *Client) PutPNG(ctx context.Context, key string, body []byte) error {
	_, err := c.api.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("image/png"),
	})
	if err != nil {
		return fmt.Errorf("put s3://%s/%s: %w", c.bucket, key, err)
	}
	return nil
}
