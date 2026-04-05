package app

import "errors"

// ErrMissingAWSCredentials — нет ключей для S3 в окружении.
var ErrMissingAWSCredentials = errors.New("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set (e.g. for MinIO)")
