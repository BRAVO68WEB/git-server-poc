package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// MigrateToOIDC performs the migration from password-based auth to OIDC
// This should be run after AutoMigrate to handle the password_hash column removal
func (d *Database) MigrateToOIDC() error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		// Check if the password_hash column exists
		var exists bool
		err := tx.Raw(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'users'
				AND column_name = 'password_hash'
			)
		`).Scan(&exists).Error

		if err != nil {
			return fmt.Errorf("failed to check for password_hash column: %w", err)
		}

		if exists {
			log.Println("Migrating users table: removing password_hash column")

			// Drop the password_hash column
			if err := tx.Exec("ALTER TABLE users DROP COLUMN IF EXISTS password_hash").Error; err != nil {
				return fmt.Errorf("failed to drop password_hash column: %w", err)
			}

			log.Println("Successfully removed password_hash column")
		}

		// Ensure OIDC columns exist (AutoMigrate should have created these)
		// This is just a safety check
		var oidcSubjectExists, oidcIssuerExists bool

		err = tx.Raw(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'users'
				AND column_name = 'oidc_subject'
			)
		`).Scan(&oidcSubjectExists).Error
		if err != nil {
			return fmt.Errorf("failed to check for oidc_subject column: %w", err)
		}

		err = tx.Raw(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'users'
				AND column_name = 'oidc_issuer'
			)
		`).Scan(&oidcIssuerExists).Error
		if err != nil {
			return fmt.Errorf("failed to check for oidc_issuer column: %w", err)
		}

		if !oidcSubjectExists {
			log.Println("Adding oidc_subject column to users table")
			if err := tx.Exec("ALTER TABLE users ADD COLUMN oidc_subject VARCHAR(255)").Error; err != nil {
				return fmt.Errorf("failed to add oidc_subject column: %w", err)
			}
		}

		if !oidcIssuerExists {
			log.Println("Adding oidc_issuer column to users table")
			if err := tx.Exec("ALTER TABLE users ADD COLUMN oidc_issuer VARCHAR(255)").Error; err != nil {
				return fmt.Errorf("failed to add oidc_issuer column: %w", err)
			}
		}

		// Create the composite unique index for OIDC subject + issuer if it doesn't exist
		var indexExists bool
		err = tx.Raw(`
			SELECT EXISTS (
				SELECT 1
				FROM pg_indexes
				WHERE tablename = 'users'
				AND indexname = 'idx_oidc_subject_issuer'
			)
		`).Scan(&indexExists).Error
		if err != nil {
			return fmt.Errorf("failed to check for OIDC index: %w", err)
		}

		if !indexExists && oidcSubjectExists && oidcIssuerExists {
			log.Println("Creating composite unique index for OIDC subject and issuer")
			if err := tx.Exec(`
				CREATE UNIQUE INDEX idx_oidc_subject_issuer
				ON users (oidc_subject, oidc_issuer)
				WHERE oidc_subject IS NOT NULL AND oidc_issuer IS NOT NULL
			`).Error; err != nil {
				return fmt.Errorf("failed to create OIDC index: %w", err)
			}
		}

		return nil
	})
}

// RunMigrations runs all database migrations
func (d *Database) RunMigrations() error {
	// First run AutoMigrate to create/update tables
	if err := d.AutoMigrate(); err != nil {
		return fmt.Errorf("auto-migrate failed: %w", err)
	}

	// Then run OIDC-specific migrations
	if err := d.MigrateToOIDC(); err != nil {
		return fmt.Errorf("OIDC migration failed: %w", err)
	}

	return nil
}
