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
	return NewLocalStorage("data/lfs")
}
