package storage

import (
	"context"
	"fmt"

	"github.com/bravo68web/stasis/internal/config"
	"github.com/bravo68web/stasis/internal/domain/service"
	"github.com/bravo68web/stasis/pkg/logger"
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
	log    *logger.Logger
}

// NewFactory creates a new storage factory
func NewFactory(cfg *config.StorageConfig) *Factory {
	return &Factory{
		config: cfg,
		log:    logger.Get().WithFields(logger.Component("storage-factory")),
	}
}

// Create creates a new storage backend based on the configuration
func (f *Factory) Create() (service.StorageService, error) {
	storageType := StorageType(f.config.Type)

	f.log.Info("Creating storage backend",
		logger.String("type", string(storageType)),
	)

	switch storageType {
	case StorageTypeFilesystem, "":
		f.log.Debug("Initializing filesystem storage",
			logger.String("base_path", f.config.BasePath),
		)
		storage, err := NewFilesystemStorage(f.config.BasePath)
		if err != nil {
			f.log.Error("Failed to create filesystem storage",
				logger.Error(err),
				logger.String("base_path", f.config.BasePath),
			)
			return nil, err
		}
		f.log.Info("Filesystem storage initialized successfully",
			logger.String("base_path", f.config.BasePath),
		)
		return storage, nil

	case StorageTypeS3:
		f.log.Debug("Initializing S3 storage",
			logger.String("bucket", f.config.S3Bucket),
			logger.String("region", f.config.S3Region),
			logger.String("endpoint", f.config.S3Endpoint),
		)
		storage, err := NewS3Storage(
			context.Background(),
			S3Config{
				Bucket:    f.config.S3Bucket,
				Region:    f.config.S3Region,
				AccessKey: f.config.S3AccessKey,
				SecretKey: f.config.S3SecretKey,
				Endpoint:  f.config.S3Endpoint,
			},
		)
		if err != nil {
			f.log.Error("Failed to create S3 storage",
				logger.Error(err),
				logger.String("bucket", f.config.S3Bucket),
				logger.String("region", f.config.S3Region),
			)
			return nil, err
		}
		f.log.Info("S3 storage initialized successfully",
			logger.String("bucket", f.config.S3Bucket),
			logger.String("region", f.config.S3Region),
		)
		return storage, nil

	default:
		f.log.Error("Unsupported storage type",
			logger.String("type", string(storageType)),
		)
		return nil, fmt.Errorf("unsupported storage type: %s", f.config.Type)
	}
}

// CreateFilesystemStorage creates a filesystem storage backend directly
func CreateFilesystemStorage(basePath string) (service.StorageService, error) {
	log := logger.Get().WithFields(logger.Component("storage"))
	log.Debug("Creating filesystem storage directly",
		logger.String("base_path", basePath),
	)
	return NewFilesystemStorage(basePath)
}

// CreateS3Storage creates an S3 storage backend directly
func CreateS3Storage(ctx context.Context, cfg S3Config) (service.StorageService, error) {
	log := logger.Get().WithFields(logger.Component("storage"))
	log.Debug("Creating S3 storage directly",
		logger.String("bucket", cfg.Bucket),
		logger.String("region", cfg.Region),
	)
	return NewS3Storage(ctx, cfg)
}

// MustCreate creates a storage backend and panics on error
func (f *Factory) MustCreate() service.StorageService {
	storage, err := f.Create()
	if err != nil {
		f.log.Fatal("Failed to create storage - cannot continue",
			logger.Error(err),
		)
	}
	return storage
}

// ValidateConfig validates the storage configuration
func ValidateConfig(cfg *config.StorageConfig) error {
	log := logger.Get().WithFields(logger.Component("storage"))
	storageType := StorageType(cfg.Type)

	log.Debug("Validating storage configuration",
		logger.String("type", string(storageType)),
	)

	switch storageType {
	case StorageTypeFilesystem, "":
		if cfg.BasePath == "" {
			log.Error("Filesystem storage validation failed - base_path is required")
			return fmt.Errorf("base_path is required for filesystem storage")
		}
		log.Debug("Filesystem storage configuration is valid",
			logger.String("base_path", cfg.BasePath),
		)
		return nil

	case StorageTypeS3:
		if cfg.S3Bucket == "" {
			log.Error("S3 storage validation failed - s3_bucket is required")
			return fmt.Errorf("s3_bucket is required for S3 storage")
		}
		if cfg.S3Region == "" {
			log.Error("S3 storage validation failed - s3_region is required")
			return fmt.Errorf("s3_region is required for S3 storage")
		}
		log.Debug("S3 storage configuration is valid",
			logger.String("bucket", cfg.S3Bucket),
			logger.String("region", cfg.S3Region),
		)
		return nil

	default:
		log.Error("Storage validation failed - unsupported storage type",
			logger.String("type", string(storageType)),
		)
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
