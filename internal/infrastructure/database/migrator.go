package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"ariga.io/atlas-go-sdk/atlasexec"

	"github.com/bravo68web/stasis/pkg/logger"
)

//go:embed all:migrations
var migrationsFS embed.FS

// Migrator handles database migrations using Atlas
type Migrator struct {
	db              *Database
	dryRun          bool
	baselineVersion string
	log             *logger.Logger
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *Database) *Migrator {
	return &Migrator{
		db:              db,
		dryRun:          false,
		baselineVersion: "",
		log:             logger.Get().WithFields(logger.Component("migrator")),
	}
}

// WithDryRun sets the migrator to dry-run mode (no actual changes)
func (m *Migrator) WithDryRun(dryRun bool) *Migrator {
	m.dryRun = dryRun
	return m
}

// WithBaseline sets a baseline version to skip migrations up to that version
// Use this when the database already has the schema from a previous setup
func (m *Migrator) WithBaseline(version string) *Migrator {
	m.baselineVersion = version
	return m
}

// ApplyMigrations applies all pending migrations to the database
func (m *Migrator) ApplyMigrations(ctx context.Context) error {
	m.log.Info("Starting database migration process...")

	// First, detect if this database already has our application tables
	// This MUST happen before any Atlas operations to properly determine baseline
	hasExistingSchema := m.detectExistingSchema(ctx)
	hasRevisionTable := m.detectRevisionTable(ctx)

	if hasExistingSchema {
		m.log.Info("Detected existing application schema in database")
	}
	if hasRevisionTable {
		m.log.Info("Detected existing Atlas revision table")
	}

	// Get the migrations subdirectory from the embedded filesystem
	m.log.Debug("Loading embedded migrations...")
	migrationsDir, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		m.log.Error("Failed to get migrations subdirectory",
			logger.Error(err),
		)
		return fmt.Errorf("failed to get migrations subdirectory: %w", err)
	}

	// Create a working directory with embedded migrations
	workdir, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(migrationsDir),
	)
	if err != nil {
		m.log.Error("Failed to create working directory",
			logger.Error(err),
		)
		return fmt.Errorf("failed to create working directory: %w", err)
	}
	defer workdir.Close()

	// Initialize the Atlas client
	m.log.Debug("Initializing Atlas migration client...")
	client, err := atlasexec.NewClient(workdir.Path(), "atlas")
	if err != nil {
		m.log.Error("Failed to initialize Atlas client",
			logger.Error(err),
		)
		return fmt.Errorf("failed to initialize atlas client: %w", err)
	}

	// Build migration apply parameters
	params := &atlasexec.MigrateApplyParams{
		URL:    m.db.config.URL(),
		DryRun: m.dryRun,
	}

	if m.dryRun {
		m.log.Info("Running migrations in dry-run mode (no actual changes)")
	}

	// If database has existing tables but NO revision table, we need to baseline
	// This means it's a database that was set up before Atlas migrations were added
	// Note: baseline and allow-dirty are mutually exclusive in Atlas
	if hasExistingSchema && !hasRevisionTable {
		// Get the latest migration version to use as baseline
		baselineVersion := m.baselineVersion
		if baselineVersion == "" {
			baselineVersion, err = m.getLatestMigrationVersion(migrationsDir)
			if err != nil {
				m.log.Error("Failed to determine baseline version",
					logger.Error(err),
				)
				return fmt.Errorf("failed to determine baseline version: %w", err)
			}
		}

		if baselineVersion != "" {
			m.log.Info("Database has existing schema without migration history, setting baseline",
				logger.String("baseline_version", baselineVersion),
			)
			params.BaselineVersion = baselineVersion
		}
	} else {
		// Only use AllowDirty when NOT using baseline
		// This handles cases where the database has a public schema but no application tables
		params.AllowDirty = true
	}

	// Apply pending migrations
	m.log.Info("Applying database migrations...")
	result, err := client.MigrateApply(ctx, params)
	if err != nil {
		// Handle partial migration failures
		if applyErr, ok := err.(*atlasexec.MigrateApplyError); ok {
			m.log.Error("Migration failed with partial apply",
				logger.Error(applyErr),
			)
			for _, r := range applyErr.Result {
				m.log.Error("Partial migration result",
					logger.Int("applied_count", len(r.Applied)),
				)
			}
		} else {
			m.log.Error("Failed to apply migrations",
				logger.Error(err),
			)
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Log migration results
	if result == nil {
		m.log.Info("Database migrations completed (baseline set)")
		return nil
	}

	if len(result.Applied) == 0 {
		m.log.Info("Database is up to date, no migrations applied")
	} else {
		m.log.Info("Successfully applied migrations",
			logger.Int("count", len(result.Applied)),
		)
		for _, applied := range result.Applied {
			m.log.Debug("Applied migration",
				logger.String("name", applied.Name),
			)
		}
	}

	if len(result.Pending) > 0 {
		m.log.Warn("Some migrations are still pending",
			logger.Int("pending_count", len(result.Pending)),
		)
	}

	return nil
}

// GetStatus returns the current migration status
func (m *Migrator) GetStatus(ctx context.Context) (*atlasexec.MigrateStatus, error) {
	m.log.Debug("Getting migration status...")

	// Get the migrations subdirectory from the embedded filesystem
	migrationsDir, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		m.log.Error("Failed to get migrations subdirectory",
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to get migrations subdirectory: %w", err)
	}

	workdir, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(migrationsDir),
	)
	if err != nil {
		m.log.Error("Failed to create working directory",
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to create working directory: %w", err)
	}
	defer workdir.Close()

	client, err := atlasexec.NewClient(workdir.Path(), "atlas")
	if err != nil {
		m.log.Error("Failed to initialize Atlas client",
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to initialize atlas client: %w", err)
	}

	status, err := client.MigrateStatus(ctx, &atlasexec.MigrateStatusParams{
		URL: m.db.config.URL(),
	})
	if err != nil {
		m.log.Error("Failed to get migration status",
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}

	m.log.Debug("Migration status retrieved successfully",
		logger.String("current", status.Current),
		logger.Int("pending_count", len(status.Pending)),
		logger.Int("applied_count", len(status.Applied)),
	)

	return status, nil
}

// MustApplyMigrations applies migrations and panics on error
// Useful for application startup where migration failure should prevent startup
func (m *Migrator) MustApplyMigrations(ctx context.Context) {
	if err := m.ApplyMigrations(ctx); err != nil {
		m.log.Fatal("Failed to apply database migrations - cannot continue",
			logger.Error(err),
		)
	}
}

// detectExistingSchema checks if the database already has application tables
func (m *Migrator) detectExistingSchema(ctx context.Context) bool {
	tables := []string{"users", "repositories", "ssh_keys", "tokens"}

	sqlDB, err := m.db.db.DB()
	if err != nil {
		m.log.Debug("Failed to get SQL DB for schema detection",
			logger.Error(err),
		)
		return false
	}

	for _, table := range tables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)`
		err := sqlDB.QueryRowContext(ctx, query, table).Scan(&exists)
		if err == nil && exists {
			m.log.Debug("Found existing application table",
				logger.String("table", table),
			)
			return true
		}
	}

	return false
}

// detectRevisionTable checks if Atlas revision table exists
func (m *Migrator) detectRevisionTable(ctx context.Context) bool {
	sqlDB, err := m.db.db.DB()
	if err != nil {
		m.log.Debug("Failed to get SQL DB for revision table detection",
			logger.Error(err),
		)
		return false
	}

	var exists bool
	query := `SELECT EXISTS (
		SELECT FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name = 'atlas_schema_revisions'
	)`
	err = sqlDB.QueryRowContext(ctx, query).Scan(&exists)
	if err == nil && exists {
		m.log.Debug("Atlas revision table exists")
	}
	return err == nil && exists
}

// getLatestMigrationVersion reads the migration files and returns the latest version
func (m *Migrator) getLatestMigrationVersion(migrationsDir fs.FS) (string, error) {
	entries, err := fs.ReadDir(migrationsDir, ".")
	if err != nil {
		m.log.Error("Failed to read migrations directory",
			logger.Error(err),
		)
		return "", fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var latestVersion string
	migrationCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Migration files are named like: 20251220201834.sql
		if len(name) >= 14 && name[len(name)-4:] == ".sql" {
			version := name[:len(name)-4] // Remove .sql extension
			// Keep the highest version (they sort lexicographically since they're timestamps)
			if version > latestVersion {
				latestVersion = version
			}
			migrationCount++
		}
	}

	m.log.Debug("Found migration files",
		logger.Int("count", migrationCount),
		logger.String("latest_version", latestVersion),
	)

	return latestVersion, nil
}
