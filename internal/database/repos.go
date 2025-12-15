package database

import (
	"context"
)

type RepoRow struct {
	ID         int64
	OwnerID    int64
	Name       string
	Visibility string
	Archived   bool
}

func (d *DB) GetRepoByOwnerAndName(ctx context.Context, ownerUsername, name string) (*RepoRow, error) {
	row := d.Conn.QueryRow(ctx, `SELECT r.id, r.owner_id, r.name, r.visibility, r.archived
FROM repos r
JOIN users u ON r.owner_id = u.id
WHERE u.username=$1 AND r.name=$2`, ownerUsername, name)
	var rr RepoRow
	if err := row.Scan(&rr.ID, &rr.OwnerID, &rr.Name, &rr.Visibility, &rr.Archived); err != nil {
		return nil, err
	}
	return &rr, nil
}

func (d *DB) CreateRepo(ctx context.Context, ownerUsername, name, visibility string) error {
	row := d.Conn.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, ownerUsername)
	var ownerID int64
	if err := row.Scan(&ownerID); err != nil {
		return err
	}
	_, err := d.Conn.Exec(ctx, `INSERT INTO repos (owner_id, name, visibility) VALUES ($1, $2, $3)
ON CONFLICT (owner_id, name) DO NOTHING`, ownerID, name, visibility)
	return err
}

func (d *DB) DeleteRepo(ctx context.Context, ownerUsername, name string) error {
	row := d.Conn.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, ownerUsername)
	var ownerID int64
	if err := row.Scan(&ownerID); err != nil {
		return err
	}
	_, err := d.Conn.Exec(ctx, `DELETE FROM repos WHERE owner_id=$1 AND name=$2`, ownerID, name)
	return err
}

func (d *DB) ListReposByOwner(ctx context.Context, ownerUsername string) ([]RepoRow, error) {
	row := d.Conn.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, ownerUsername)
	var ownerID int64
	if err := row.Scan(&ownerID); err != nil {
		return nil, err
	}
	rows, err := d.Conn.Query(ctx, `SELECT id, owner_id, name, visibility, archived FROM repos WHERE owner_id=$1 ORDER BY name`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RepoRow
	for rows.Next() {
		var r RepoRow
		if err := rows.Scan(&r.ID, &r.OwnerID, &r.Name, &r.Visibility, &r.Archived); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type RepoWithUserRow struct {
	OwnerName   string
	Name        string
	Description string
	Visibility  string
}

func (d *DB) ListAllRepos(ctx context.Context) ([]RepoWithUserRow, error) {
	rows, err := d.Conn.Query(ctx, `SELECT u.username, r.name, r.description, r.visibility 
FROM repos r 
JOIN users u ON r.owner_id = u.id 
ORDER BY u.username, r.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RepoWithUserRow
	for rows.Next() {
		var r RepoWithUserRow
		var desc *string
		if err := rows.Scan(&r.OwnerName, &r.Name, &desc, &r.Visibility); err != nil {
			return nil, err
		}
		if desc != nil {
			r.Description = *desc
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
