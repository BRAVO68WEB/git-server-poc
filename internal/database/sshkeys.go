package database

import (
	"context"
)

type SSHKeyRow struct {
	ID          int64
	UserID      int64
	Name        string
	PubKey      string
	Fingerprint string
}

func (d *DB) AddSSHKey(ctx context.Context, username, name, pubkey, fingerprint string) error {
	row := d.Conn.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, username)
	var userID int64
	if err := row.Scan(&userID); err != nil {
		return err
	}
	_, err := d.Conn.Exec(ctx, `INSERT INTO ssh_keys (user_id, name, pubkey, fingerprint) VALUES ($1, $2, $3, $4)`, userID, name, pubkey, fingerprint)
	return err
}

func (d *DB) ListSSHKeys(ctx context.Context, username string) ([]SSHKeyRow, error) {
	row := d.Conn.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, username)
	var userID int64
	if err := row.Scan(&userID); err != nil {
		return nil, err
	}
	rows, err := d.Conn.Query(ctx, `SELECT id, user_id, name, pubkey, fingerprint FROM ssh_keys WHERE user_id=$1 ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SSHKeyRow
	for rows.Next() {
		var k SSHKeyRow
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.PubKey, &k.Fingerprint); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (d *DB) DeleteSSHKeyByName(ctx context.Context, username, name string) error {
	row := d.Conn.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, username)
	var userID int64
	if err := row.Scan(&userID); err != nil {
		return err
	}
	_, err := d.Conn.Exec(ctx, `DELETE FROM ssh_keys WHERE user_id=$1 AND name=$2`, userID, name)
	return err
}

func (d *DB) GetUserByPublicKey(ctx context.Context, pubkey string) (*UserRow, error) {
	row := d.Conn.QueryRow(ctx, `SELECT u.id, u.username, u.email, u.role, u.disabled
FROM ssh_keys k
JOIN users u ON k.user_id=u.id
WHERE k.pubkey=$1`, pubkey)
	var u UserRow
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Disabled); err != nil {
		return nil, err
	}
	return &u, nil
}
