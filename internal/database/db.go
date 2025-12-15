package database

// import (
// 	"context"

// 	"github.com/jackc/pgx/v5"
// )

// type DB struct {
// 	Conn *pgx.Conn
// }

// func Connect(ctx context.Context, dsn string) (*DB, error) {
// 	conn, err := pgx.Connect(ctx, dsn)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &DB{Conn: conn}, nil
// }

// func (d *DB) Close(ctx context.Context) error {
// 	return d.Conn.Close(ctx)
// }
