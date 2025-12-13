package database

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type UserRow struct {
	ID       int64
	Username string
	Email    string
	Role     string
	Disabled bool
}

func (d *DB) CreateUser(ctx context.Context, username, email, role, rawPassword string) error {
	if strings.TrimSpace(rawPassword) == "" {
		return errors.New("password required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = d.Conn.Exec(ctx, `INSERT INTO users (username, email, role, password_hash) VALUES ($1, $2, $3, $4)
ON CONFLICT (username) DO UPDATE SET password_hash=EXCLUDED.password_hash`, username, email, role, string(hash))
	return err
}

func (d *DB) DeleteUserByUsername(ctx context.Context, username string) error {
	_, err := d.Conn.Exec(ctx, `DELETE FROM users WHERE username=$1`, username)
	return err
}

func (d *DB) ListUsers(ctx context.Context) ([]UserRow, error) {
	rows, err := d.Conn.Query(ctx, `SELECT id, username, email, role, disabled FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserRow
	for rows.Next() {
		var u UserRow
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Disabled); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (d *DB) GetUserByUsername(ctx context.Context, username string) (*UserRow, error) {
	row := d.Conn.QueryRow(ctx, `SELECT id, username, email, role, disabled FROM users WHERE username=$1`, username)
	var u UserRow
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Disabled); err != nil {
		return nil, err
	}
	return &u, nil
}

func (d *DB) ValidateUserPassword(ctx context.Context, username, rawPassword string) (*UserRow, error) {
	row := d.Conn.QueryRow(ctx, `SELECT u.id, u.username, u.email, u.role, u.disabled, u.password_hash FROM users u WHERE u.username=$1`, username)
	var u UserRow
	var pw string
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Disabled, &pw); err != nil {
		return nil, err
	}
	if pw == "" {
		return nil, errors.New("no password set")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(pw), []byte(rawPassword)); err != nil {
		return nil, err
	}
	return &u, nil
}
