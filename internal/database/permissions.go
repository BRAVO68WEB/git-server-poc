package database

// import "context"

// func (d *DB) HasPullAccess(ctx context.Context, username, ownerUsername, repoName string) (bool, error) {
// 	row := d.Conn.QueryRow(ctx, `SELECT u.role, u.id, r.owner_id
// FROM users u
// JOIN repos r ON r.owner_id = (SELECT id FROM users WHERE username=$2)
// WHERE u.username=$1 AND r.name=$3`, username, ownerUsername, repoName)
// 	var userRole string
// 	var userID int64
// 	var ownerID int64
// 	if err := row.Scan(&userRole, &userID, &ownerID); err != nil {
// 		return false, err
// 	}
// 	if userRole == "admin" || userID == ownerID {
// 		return true, nil
// 	}
// 	row2 := d.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM repo_members rm WHERE rm.repo_id = (SELECT r.id FROM repos r JOIN users u ON r.owner_id=u.id WHERE u.username=$1 AND r.name=$2) AND rm.user_id=$3`, ownerUsername, repoName, userID)
// 	var cnt int
// 	if err := row2.Scan(&cnt); err != nil {
// 		return false, err
// 	}
// 	return cnt > 0, nil
// }

// func (d *DB) HasPushAccess(ctx context.Context, username, ownerUsername, repoName string) (bool, error) {
// 	row := d.Conn.QueryRow(ctx, `SELECT u.role, u.id, r.owner_id
// FROM users u
// JOIN repos r ON r.owner_id = (SELECT id FROM users WHERE username=$2)
// WHERE u.username=$1 AND r.name=$3`, username, ownerUsername, repoName)
// 	var userRole string
// 	var userID int64
// 	var ownerID int64
// 	if err := row.Scan(&userRole, &userID, &ownerID); err != nil {
// 		return false, err
// 	}
// 	if userRole == "admin" || userID == ownerID {
// 		return true, nil
// 	}
// 	row2 := d.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM repo_members rm WHERE rm.repo_id = (SELECT r.id FROM repos r JOIN users u ON r.owner_id=u.id WHERE u.username=$1 AND r.name=$2) AND rm.user_id=$3 AND rm.role IN ('maintainer','developer')`, ownerUsername, repoName, userID)
// 	var cnt int
// 	if err := row2.Scan(&cnt); err != nil {
// 		return false, err
// 	}
// 	return cnt > 0, nil
// }
