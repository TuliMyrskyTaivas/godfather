package godfather

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Database struct {
	conn *pgx.Conn
}

func InitDB(host string, port int, user string, passwd string, database string) (*Database, error) {
	uri := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, passwd, host, port, database)
	conn, err := pgx.Connect(context.Background(), uri)
	if err != nil {
		return nil, err
	}

	return &Database{conn: conn}, nil
}

func (db *Database) Migrate() error {
	sql := `
	CREATE TABLE IF NOT EXISTS sources (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR NOT NULL,
		update_interval INTERVAL NOT NULL,
		last_update TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS moex_operands (
	);
	`
	_, err := db.conn.Exec(context.Background(), sql)
	return err
}
