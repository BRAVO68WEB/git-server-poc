package database

// import "context"

// type MemberRow struct {
// 	UserID int64
// 	RepoID int64
// 	Role   string
// 	Username string
// }

// func (d *DB) AddMember(ctx context.Context, ownerUsername, repoName, memberUsername, role string) error {
// 	row := d.Conn.QueryRow(ctx, `SELECT r.id FROM repos r JOIN users u ON r.owner_id=u.id WHERE u.username=$1 AND r.name=$2`, ownerUsername, repoName)
// 	var repoID int64
// 	if err := row.Scan(&repoID); err != nil {
// 		return err
// 	}
// 	row2 := d.Conn.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, memberUsername)
// 	var userID int64
// 	if err := row2.Scan(&userID); err != nil {
// 		return err
// 	}
// 	_, err := d.Conn.Exec(ctx, `INSERT INTO repo_members (repo_id, user_id, role) VALUES ($1, $2, $3) ON CONFLICT (repo_id, user_id) DO UPDATE SET role=EXCLUDED.role`, repoID, userID, role)
// 	return err
// }

// func (d *DB) RemoveMember(ctx context.Context, ownerUsername, repoName, memberUsername string) error {
// 	row := d.Conn.QueryRow(ctx, `SELECT r.id FROM repos r JOIN users u ON r.owner_id=u.id WHERE u.username=$1 AND r.name=$2`, ownerUsername, repoName)
// 	var repoID int64
// 	if err := row.Scan(&repoID); err != nil {
// 		return err
// 	}
// 	row2 := d.Conn.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, memberUsername)
// 	var userID int64
// 	if err := row2.Scan(&userID); err != nil {
// 		return err
// 	}
// 	_, err := d.Conn.Exec(ctx, `DELETE FROM repo_members WHERE repo_id=$1 AND user_id=$2`, repoID, userID)
// 	return err
// }

// func (d *DB) ListMembers(ctx context.Context, ownerUsername, repoName string) ([]MemberRow, error) {
// 	row := d.Conn.QueryRow(ctx, `SELECT r.id FROM repos r JOIN users u ON r.owner_id=u.id WHERE u.username=$1 AND r.name=$2`, ownerUsername, repoName)
// 	var repoID int64
// 	if err := row.Scan(&repoID); err != nil {
// 		return nil, err
// 	}
// 	rows, err := d.Conn.Query(ctx, `SELECT rm.user_id, rm.repo_id, rm.role, u.username FROM repo_members rm JOIN users u ON rm.user_id=u.id WHERE rm.repo_id=$1 ORDER BY u.username`, repoID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()
// 	var out []MemberRow
// 	for rows.Next() {
// 		var mr MemberRow
// 		if err := rows.Scan(&mr.UserID, &mr.RepoID, &mr.Role, &mr.Username); err != nil {
// 			return nil, err
// 		}
// 		out = append(out, mr)
// 	}
// 	return out, rows.Err()
// }
