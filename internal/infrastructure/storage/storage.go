package storage

import (
	"context"
	"fmt"

	"github.com/bravo68web/stasis/internal/config"
	"github.com/bravo68web/stasis/internal/domain/service"
)

// StorageType represents the type of storage backend
type StorageType string

const (
	// StorageTypeFilesystem represents local filesystem storage
	StorageTypeFilesystem StorageType = "filesystem"

	// StorageTypeS3 represents AWS S3 storage
	StorageTypeS3 StorageType = "s3"
)

// Factory creates storage backends based on configuration
type Factory struct {
	config *config.StorageConfig
}

// NewFactory creates a new storage factory
func NewFactory(cfg *config.StorageConfig) *Factory {
	return &Factory{
		config: cfg,
	}
}

// Create creates a new storage backend based on the configuration
func (f *Factory) Create() (service.StorageService, error) {
	storageType := StorageType(f.config.Type)

	switch storageType {
	case StorageTypeFilesystem, "":
		return NewFilesystemStorage(f.config.BasePath)

	case StorageTypeS3:
		return NewS3Storage(
			context.Background(),
			S3Config{
				Bucket:    f.config.S3Bucket,
				Region:    f.config.S3Region,
				AccessKey: f.config.S3AccessKey,
				SecretKey: f.config.S3SecretKey,
				Endpoint:  f.config.S3Endpoint,
			},
		)

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", f.config.Type)
	}
}

// CreateFilesystemStorage creates a filesystem storage backend directly
func CreateFilesystemStorage(basePath string) (service.StorageService, error) {
	return NewFilesystemStorage(basePath)
}

// CreateS3Storage creates an S3 storage backend directly
func CreateS3Storage(ctx context.Context, cfg S3Config) (service.StorageService, error) {
	return NewS3Storage(ctx, cfg)
}

// MustCreate creates a storage backend and panics on error
func (f *Factory) MustCreate() service.StorageService {
	storage, err := f.Create()
	if err != nil {
		panic(fmt.Sprintf("failed to create storage: %v", err))
	}
	return storage
}

// ValidateConfig validates the storage configuration
func ValidateConfig(cfg *config.StorageConfig) error {
	storageType := StorageType(cfg.Type)

	switch storageType {
	case StorageTypeFilesystem, "":
		if cfg.BasePath == "" {
			return fmt.Errorf("base_path is required for filesystem storage")
		}
		return nil

	case StorageTypeS3:
		if cfg.S3Bucket == "" {
			return fmt.Errorf("s3_bucket is required for S3 storage")
		}
		if cfg.S3Region == "" {
			return fmt.Errorf("s3_region is required for S3 storage")
		}
		return nil

	default:
		return fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}

// GetStorageType returns the storage type from configuration
func GetStorageType(cfg *config.StorageConfig) StorageType {
	if cfg.Type == "" {
		return StorageTypeFilesystem
	}
	return StorageType(cfg.Type)
}

// IsFilesystem checks if the storage type is filesystem
func IsFilesystem(cfg *config.StorageConfig) bool {
	return GetStorageType(cfg) == StorageTypeFilesystem
}

// IsS3 checks if the storage type is S3
func IsS3(cfg *config.StorageConfig) bool {
	return GetStorageType(cfg) == StorageTypeS3
}
