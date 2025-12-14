package storage

import (
	"context"
	"io"

	"githut/internal/config"
)

type Storage interface {
	Put(ctx context.Context, key string, r io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

func FromConfig(cfg config.Config) Storage {
	if cfg.S3Bucket != "" && cfg.S3Region != "" && cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		return NewS3Storage(cfg.S3Region, cfg.S3Bucket, cfg.S3Endpoint, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, cfg.AWSSessionToken)
	}
	return NewLocalStorage("data/lfs")
}
