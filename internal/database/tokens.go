package database

// import (
// 	"context"
// 	"crypto/rand"
// 	"crypto/sha256"
// 	"encoding/hex"
// )

// type TokenRow struct {
// 	ID        int64
// 	UserID    int64
// 	Name      string
// 	TokenHash string
// 	Revoked   bool
// }

// func (d *DB) CreateToken(ctx context.Context, username, name string) (string, error) {
// 	row := d.Conn.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, username)
// 	var userID int64
// 	if err := row.Scan(&userID); err != nil {
// 		return "", err
// 	}
// 	raw, err := newTokenRaw()
// 	if err != nil {
// 		return "", err
// 	}
// 	hash := hashToken(raw)
// 	_, err = d.Conn.Exec(ctx, `INSERT INTO tokens (user_id, name, token_hash) VALUES ($1, $2, $3)`, userID, name, hash)
// 	if err != nil {
// 		return "", err
// 	}
// 	return raw, nil
// }

// func (d *DB) ValidateToken(ctx context.Context, raw string) (*UserRow, error) {
// 	hash := hashToken(raw)
// 	row := d.Conn.QueryRow(ctx, `SELECT u.id, u.username, u.email, u.role, u.disabled
// FROM tokens t
// JOIN users u ON t.user_id=u.id
// WHERE t.token_hash=$1 AND t.revoked=false`, hash)
// 	var u UserRow
// 	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Disabled); err != nil {
// 		return nil, err
// 	}
// 	return &u, nil
// }

// func (d *DB) RevokeToken(ctx context.Context, raw string) error {
// 	hash := hashToken(raw)
// 	_, err := d.Conn.Exec(ctx, `UPDATE tokens SET revoked=true WHERE token_hash=$1`, hash)
// 	return err
// }

// func (d *DB) ListTokensByUser(ctx context.Context, username string) ([]TokenRow, error) {
// 	row := d.Conn.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, username)
// 	var userID int64
// 	if err := row.Scan(&userID); err != nil {
// 		return nil, err
// 	}
// 	rows, err := d.Conn.Query(ctx, `SELECT id, user_id, name, token_hash, revoked FROM tokens WHERE user_id=$1 ORDER BY id DESC`, userID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()
// 	var out []TokenRow
// 	for rows.Next() {
// 		var tr TokenRow
// 		if err := rows.Scan(&tr.ID, &tr.UserID, &tr.Name, &tr.TokenHash, &tr.Revoked); err != nil {
// 			return nil, err
// 		}
// 		out = append(out, tr)
// 	}
// 	return out, rows.Err()
// }

// func newTokenRaw() (string, error) {
// 	b := make([]byte, 32)
// 	if _, err := rand.Read(b); err != nil {
// 		return "", err
// 	}
// 	return hex.EncodeToString(b), nil
// }

// func hashToken(raw string) string {
// 	sum := sha256.Sum256([]byte(raw))
// 	return hex.EncodeToString(sum[:])
// }
