// Package apperrors объявляет доменные ошибки сервиса data_ingestion.
package apperrors

import "errors"

// ErrMissingAWSCredentials возвращается, если в окружении нет ключей S3.
var ErrMissingAWSCredentials = errors.New("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set (e.g. for MinIO)")
