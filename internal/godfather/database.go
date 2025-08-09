package godfather

import (
	"database/sql"
	"fmt"
	"os"

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

type MOEXWatchlistItem struct {
	Ticker         string
	AssetClass     string
	NotificationID int
	TargetPrice    float64
	Condition      string
	Active         bool
}

// ----------------------------------------------------------------
// Initialize the database connection from environment variables
// ----------------------------------------------------------------
func InitDBFromEnv() (*Database, error) {
	connString := os.Getenv("GODFATHER_DB_CONN")
	if connString == "" {
		return nil, fmt.Errorf("GODFATHER_DB_CONN environment variable is not set")
	}
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	log.Debug("Database connection established successfully")
	return &Database{handle: db}, nil
}

// ----------------------------------------------------------------
// Initialize the database connection with specific parameters
// ----------------------------------------------------------------
func InitDB(host string, port int, user string, passwd string, database string) (*Database, error) {
	uri := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, passwd, host, port, database)
	db, err := sql.Open("pgx", uri)
	if err != nil {
		return nil, err
	}

	return &Database{handle: db}, nil
}

// ----------------------------------------------------------------
// Migrate the database schema using migrations
// ----------------------------------------------------------------
func (db *Database) Migrate() error {
	instance, err := postgres.WithInstance(db.handle, new(postgres.Config))
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "pgx5", instance)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	version, _, err := m.Version()
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Database migrated to version %d", version))
	return nil
}

// ----------------------------------------------------------------
// Close the database connection
// ----------------------------------------------------------------
func (db *Database) Close() error {
	if db.handle != nil {
		if err := db.handle.Close(); err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
		log.Debug("Database connection closed successfully")
	}
	return nil
}

// ----------------------------------------------------------------
// Get MOEX watchlist from the database
// ----------------------------------------------------------------
func (db *Database) GetMOEXWatchlist() ([]MOEXWatchlistItem, error) {
	rows, err := db.handle.Query("SELECT moex_assets.ticker, moex_assets.class_id, moex_watchlist.notification_id, moex_watchlist.target_price::numeric, moex_watchlist.condition, moex_watchlist.is_active FROM moex_watchlist INNER JOIN moex_assets ON moex_watchlist.ticker_id = moex_assets.ticker")
	if err != nil {
		return nil, fmt.Errorf("failed to query MOEX watchlist: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Errorf("failed to close rows: %v", err)
		}
	}()

	var watchlist []MOEXWatchlistItem
	for rows.Next() {
		var item MOEXWatchlistItem
		if err := rows.Scan(&item.Ticker, &item.AssetClass, &item.NotificationID, &item.TargetPrice, &item.Condition, &item.Active); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		watchlist = append(watchlist, item)
	}
	return watchlist, nil
}

// ----------------------------------------------------------------
func (db *Database) SetMOEXWatchlistItemActiveStatus(ticker string, active bool) error {
	query := "UPDATE moex_watchlist SET is_active = $1 WHERE ticker_id = $2"
	_, err := db.handle.Exec(query, active, ticker)
	if err != nil {
		return fmt.Errorf("failed to update MOEX watchlist item active status: %w", err)
	}
	log.Debug(fmt.Sprintf("MOEX watchlist item %s active status set to %t", ticker, active))
	return nil
}
