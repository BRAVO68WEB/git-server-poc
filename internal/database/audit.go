package database

// import (
// 	"context"
// 	"net"
// )

// func (d *DB) InsertAuditByOwnerRepo(ctx context.Context, ownerUsername, repoName string, actorID *int64, action string, ip net.IP, meta string) error {
// 	row := d.Conn.QueryRow(ctx, `SELECT r.id FROM repos r JOIN users u ON r.owner_id=u.id WHERE u.username=$1 AND r.name=$2`, ownerUsername, repoName)
// 	var repoID int64
// 	if err := row.Scan(&repoID); err != nil {
// 		return err
// 	}
// 	_, err := d.Conn.Exec(ctx, `INSERT INTO audit_logs (actor_id, action, repo_id, ip, meta) VALUES ($1, $2, $3, $4, $5)`, actorID, action, repoID, ip, meta)
// 	return err
// }

// type AuditRow struct {
// 	ID      int64
// 	ActorID *int64
// 	Action  string
// 	RepoID  int64
// 	IP      string
// 	Meta    string
// }

// func (d *DB) ListAuditByOwnerRepo(ctx context.Context, ownerUsername, repoName string, limit int) ([]AuditRow, error) {
// 	row := d.Conn.QueryRow(ctx, `SELECT r.id FROM repos r JOIN users u ON r.owner_id=u.id WHERE u.username=$1 AND r.name=$2`, ownerUsername, repoName)
// 	var repoID int64
// 	if err := row.Scan(&repoID); err != nil {
// 		return nil, err
// 	}
// 	rows, err := d.Conn.Query(ctx, `SELECT id, actor_id, action, repo_id, ip::text, meta::text FROM audit_logs WHERE repo_id=$1 ORDER BY id DESC LIMIT $2`, repoID, limit)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()
// 	var out []AuditRow
// 	for rows.Next() {
// 		var ar AuditRow
// 		if err := rows.Scan(&ar.ID, &ar.ActorID, &ar.Action, &ar.RepoID, &ar.IP, &ar.Meta); err != nil {
// 			return nil, err
// 		}
// 		out = append(out, ar)
// 	}
// 	return out, rows.Err()
// }
