package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/bravo68web/stasis/internal/config"
	"github.com/bravo68web/stasis/pkg/logger"
)

// Connection pool settings
const (
	maxIdleConns    = 10
	maxOpenConns    = 100
	connMaxLifetime = time.Hour
	connMaxIdleTime = 10 * time.Minute
)

// Database wraps the GORM database connection
type Database struct {
	db     *gorm.DB
	config *config.DatabaseConfig
	log    *logger.Logger
}

// NewDatabase creates a new database connection
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	log := logger.Get().WithFields(logger.Component("database"))

	log.Info("Initializing database connection...",
		logger.String("host", cfg.Host),
		logger.Int("port", cfg.Port),
		logger.String("database", cfg.DBName),
		logger.String("user", cfg.User),
		logger.String("sslmode", cfg.SSLMode),
	)

	// Configure GORM logger based on environment
	var gormLogger gormlogger.Interface
	gormLogger = gormlogger.Default.LogMode(gormlogger.Silent)

	// Build connection string
	dsn := cfg.DSN()

	log.Debug("Connecting to PostgreSQL...")

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: false,
		PrepareStmt:                              true,
	})
	if err != nil {
		log.Error("Failed to connect to database",
			logger.Error(err),
			logger.String("host", cfg.Host),
			logger.Int("port", cfg.Port),
			logger.String("database", cfg.DBName),
		)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Debug("Database connection established, configuring connection pool...")

	// Get underlying SQL DB and configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Error("Failed to get underlying SQL DB",
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	log.Debug("Connection pool configured",
		logger.Int("max_idle_conns", maxIdleConns),
		logger.Int("max_open_conns", maxOpenConns),
		logger.Duration("conn_max_lifetime", connMaxLifetime),
		logger.Duration("conn_max_idle_time", connMaxIdleTime),
	)

	database := &Database{
		db:     db,
		config: cfg,
		log:    log,
	}

	// Verify connection
	log.Debug("Verifying database connection with ping...")
	if err := database.Ping(context.Background()); err != nil {
		log.Error("Failed to ping database",
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Database connection established successfully",
		logger.String("host", cfg.Host),
		logger.Int("port", cfg.Port),
		logger.String("database", cfg.DBName),
	)

	return database, nil
}

// DB returns the underlying GORM database instance
func (d *Database) DB() *gorm.DB {
	return d.db
}

// Ping checks the database connection
func (d *Database) Ping(ctx context.Context) error {
	sqlDB, err := d.db.DB()
	if err != nil {
		d.log.Error("Failed to get underlying SQL DB for ping",
			logger.Error(err),
		)
		return fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		d.log.Error("Database ping failed",
			logger.Error(err),
		)
		return err
	}

	d.log.Debug("Database ping successful")
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	d.log.Info("Closing database connection...")

	sqlDB, err := d.db.DB()
	if err != nil {
		d.log.Error("Failed to get underlying SQL DB for close",
			logger.Error(err),
		)
		return fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		d.log.Error("Failed to close database connection",
			logger.Error(err),
		)
		return err
	}

	d.log.Info("Database connection closed successfully")
	return nil
}

// Stats returns database connection pool statistics
func (d *Database) Stats() map[string]any {
	sqlDB, err := d.db.DB()
	if err != nil {
		return nil
	}

	stats := sqlDB.Stats()
	return map[string]any{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}

// LogStats logs the current database connection pool statistics
func (d *Database) LogStats() {
	stats := d.Stats()
	if stats == nil {
		return
	}

	d.log.Info("Database connection pool statistics",
		logger.Int("max_open_connections", stats["max_open_connections"].(int)),
		logger.Int("open_connections", stats["open_connections"].(int)),
		logger.Int("in_use", stats["in_use"].(int)),
		logger.Int("idle", stats["idle"].(int)),
		logger.Int64("wait_count", stats["wait_count"].(int64)),
		logger.String("wait_duration", stats["wait_duration"].(string)),
	)
}
