package storage

import (
	"context"
	"errors"
	"io"
)

type S3Storage struct{}

func NewS3Storage() *S3Storage {
	return &S3Storage{}
}

func (s *S3Storage) Put(ctx context.Context, key string, r io.Reader) error {
	return errors.New("not implemented")
}

func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	return errors.New("not implemented")
}
