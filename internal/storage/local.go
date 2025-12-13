package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	root string
}

func NewLocalStorage(root string) *LocalStorage {
	return &LocalStorage{root: root}
}

func (s *LocalStorage) path(key string) string {
	return filepath.Join(s.root, key)
}

func (s *LocalStorage) Put(ctx context.Context, key string, r io.Reader) error {
	p := s.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (s *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	p := s.path(key)
	return os.Open(p)
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	p := s.path(key)
	return os.Remove(p)
}
