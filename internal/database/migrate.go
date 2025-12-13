package database

import (
	"context"
	"os"
	"path/filepath"
	"sort"
)

type migrationFile struct {
	Name string
	Path string
}

func (d *DB) ensureSchemaMigrations(ctx context.Context) error {
	_, err := d.Conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ DEFAULT now())`)
	return err
}

func listMigrationFiles() ([]migrationFile, error) {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	var files []migrationFile
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, migrationFile{
				Name: e.Name(),
				Path: filepath.Join("migrations", e.Name()),
			})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}

func (d *DB) appliedVersions(ctx context.Context) (map[string]struct{}, error) {
	if err := d.ensureSchemaMigrations(ctx); err != nil {
		return nil, err
	}
	rows, err := d.Conn.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = struct{}{}
	}
	return out, rows.Err()
}

func (d *DB) ApplyMigrations(ctx context.Context) error {
	if err := d.ensureSchemaMigrations(ctx); err != nil {
		return err
	}
	files, err := listMigrationFiles()
	if err != nil {
		return err
	}
	applied, err := d.appliedVersions(ctx)
	if err != nil {
		return err
	}
	for _, f := range files {
		if _, ok := applied[f.Name]; ok {
			continue
		}
		b, err := os.ReadFile(f.Path)
		if err != nil {
			return err
		}
		_, err = d.Conn.Exec(ctx, string(b))
		if err != nil {
			return err
		}
		_, err = d.Conn.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, f.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

type MigrationStatus struct {
	Name    string
	Applied bool
}

func (d *DB) MigrationsStatus(ctx context.Context) ([]MigrationStatus, error) {
	files, err := listMigrationFiles()
	if err != nil {
		return nil, err
	}
	applied, err := d.appliedVersions(ctx)
	if err != nil {
		return nil, err
	}
	var out []MigrationStatus
	for _, f := range files {
		_, ok := applied[f.Name]
		out = append(out, MigrationStatus{
			Name:    f.Name,
			Applied: ok,
		})
	}
	return out, nil
}

func (d *DB) StashAll(ctx context.Context) error {
	// Drop in dependency order; ignore missing tables
	stmts := []string{
		`DROP TABLE IF EXISTS audit_logs`,
		`DROP TABLE IF EXISTS repo_members`,
		`DROP TABLE IF EXISTS repos`,
		`DROP TABLE IF EXISTS ssh_keys`,
		`DROP TABLE IF EXISTS tokens`,
		`DROP TABLE IF EXISTS users`,
		`DROP TABLE IF EXISTS schema_migrations`,
	}
	for _, s := range stmts {
		if _, err := d.Conn.Exec(ctx, s); err != nil {
			return err
		}
	}
	return nil
}
