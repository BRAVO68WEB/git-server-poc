package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"

	"ariga.io/atlas-go-sdk/atlasexec"
)

//go:embed all:migrations
var migrationsFS embed.FS

// Migrator handles database migrations using Atlas
type Migrator struct {
	db              *Database
	dryRun          bool
	baselineVersion string
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *Database) *Migrator {
	return &Migrator{
		db:              db,
		dryRun:          false,
		baselineVersion: "",
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
	// First, detect if this database already has our application tables
	// This MUST happen before any Atlas operations to properly determine baseline
	hasExistingSchema := m.detectExistingSchema(ctx)
	hasRevisionTable := m.detectRevisionTable(ctx)

	if hasExistingSchema {
		log.Println("Detected existing application schema in database")
	}
	if hasRevisionTable {
		log.Println("Detected existing Atlas revision table")
	}

	// Get the migrations subdirectory from the embedded filesystem
	migrationsDir, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to get migrations subdirectory: %w", err)
	}

	// Create a working directory with embedded migrations
	workdir, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(migrationsDir),
	)
	if err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}
	defer workdir.Close()

	// Initialize the Atlas client
	client, err := atlasexec.NewClient(workdir.Path(), "atlas")
	if err != nil {
		return fmt.Errorf("failed to initialize atlas client: %w", err)
	}

	// Build migration apply parameters
	params := &atlasexec.MigrateApplyParams{
		URL:    m.db.config.URL(),
		DryRun: m.dryRun,
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
				return fmt.Errorf("failed to determine baseline version: %w", err)
			}
		}

		if baselineVersion != "" {
			log.Printf("Database has existing schema without migration history, setting baseline to version: %s", baselineVersion)
			params.BaselineVersion = baselineVersion
		}
	} else {
		// Only use AllowDirty when NOT using baseline
		// This handles cases where the database has a public schema but no application tables
		params.AllowDirty = true
	}

	// Apply pending migrations
	result, err := client.MigrateApply(ctx, params)
	if err != nil {
		// Handle partial migration failures
		if applyErr, ok := err.(*atlasexec.MigrateApplyError); ok {
			log.Printf("Migration failed with partial apply: %s", applyErr.Error())
			for _, r := range applyErr.Result {
				log.Printf("Applied %d migrations before failure", len(r.Applied))
			}
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Log migration results
	if result == nil {
		log.Println("Database migrations completed (baseline set)")
		return nil
	}

	if len(result.Applied) == 0 {
		log.Println("Database is up to date, no migrations applied")
	} else {
		log.Printf("Successfully applied %d migration(s)", len(result.Applied))
		for _, applied := range result.Applied {
			log.Printf("  - Applied: %s", applied.Name)
		}
	}

	if len(result.Pending) > 0 {
		log.Printf("Note: %d migration(s) still pending", len(result.Pending))
	}

	return nil
}

// GetStatus returns the current migration status
func (m *Migrator) GetStatus(ctx context.Context) (*atlasexec.MigrateStatus, error) {
	// Get the migrations subdirectory from the embedded filesystem
	migrationsDir, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to get migrations subdirectory: %w", err)
	}

	workdir, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(migrationsDir),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create working directory: %w", err)
	}
	defer workdir.Close()

	client, err := atlasexec.NewClient(workdir.Path(), "atlas")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize atlas client: %w", err)
	}

	status, err := client.MigrateStatus(ctx, &atlasexec.MigrateStatusParams{
		URL: m.db.config.URL(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}

	return status, nil
}

// MustApplyMigrations applies migrations and panics on error
// Useful for application startup where migration failure should prevent startup
func (m *Migrator) MustApplyMigrations(ctx context.Context) {
	if err := m.ApplyMigrations(ctx); err != nil {
		panic(fmt.Sprintf("failed to apply database migrations: %v", err))
	}
}

// detectExistingSchema checks if the database already has application tables
func (m *Migrator) detectExistingSchema(ctx context.Context) bool {
	tables := []string{"users", "repositories", "ssh_keys", "tokens"}

	sqlDB, err := m.db.db.DB()
	if err != nil {
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
			return true
		}
	}

	return false
}

// detectRevisionTable checks if Atlas revision table exists
func (m *Migrator) detectRevisionTable(ctx context.Context) bool {
	sqlDB, err := m.db.db.DB()
	if err != nil {
		return false
	}

	var exists bool
	query := `SELECT EXISTS (
		SELECT FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name = 'atlas_schema_revisions'
	)`
	err = sqlDB.QueryRowContext(ctx, query).Scan(&exists)
	return err == nil && exists
}

// getLatestMigrationVersion reads the migration files and returns the latest version
func (m *Migrator) getLatestMigrationVersion(migrationsDir fs.FS) (string, error) {
	entries, err := fs.ReadDir(migrationsDir, ".")
	if err != nil {
		return "", fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var latestVersion string
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
		}
	}

	return latestVersion, nil
}
