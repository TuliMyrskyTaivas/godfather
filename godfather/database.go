package godfather

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/gommon/log"
)

type Database struct {
	handle *sql.DB
}

func InitDB(host string, port int, user string, passwd string, database string) (*Database, error) {
	uri := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, passwd, host, port, database)
	db, err := sql.Open("pgx", uri)
	if err != nil {
		return nil, err
	}

	return &Database{handle: db}, nil
}

func (db *Database) Migrate() error {
	instance, err := postgres.WithInstance(db.handle, new(postgres.Config))
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance("file://db/migrations", "pgx5", instance)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		return err
	}

	version, _, err := m.Version()
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Database migrated to version %d", version))
	return nil
}
