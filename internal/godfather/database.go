package godfather

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/gommon/log"
)

// ----------------------------------------------------------------
// Database wrapper type
// ----------------------------------------------------------------
type Database struct {
	handle *sql.DB
}

// ----------------------------------------------------------------
// MOEX watchlist item
// ----------------------------------------------------------------
type MOEXWatchlistItem struct {
	Ticker         string
	AssetClass     string
	NotificationID int
	TargetPrice    float64
	Condition      string
	Active         bool
}

// ----------------------------------------------------------------
// Notification
// ----------------------------------------------------------------
type Notification struct {
	ID                 int
	TelegramBotID      string
	TelegramChatID     int64
	SmtpHost           string
	SmtpPort           int
	SmtpUser           string
	SmtpPass           string
	SmtpFrom           string
	SmtpEncryptionType string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ----------------------------------------------------------------
// User
// ----------------------------------------------------------------
type User struct {
	ID        int
	Name      string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ----------------------------------------------------------------
// User not found error
// ----------------------------------------------------------------
type UserNotFound struct {
	ID   int
	Name string
}

func (e *UserNotFound) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("user not found: %s", e.Name)
	}
	return fmt.Sprintf("user not found: %d", e.ID)
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
// User management
// ----------------------------------------------------------------

// ----------------------------------------------------------------
// Get a user by ID
// ----------------------------------------------------------------
func (db *Database) GetUserByID(id int) (*User, error) {
	query := "SELECT id, name, password, created_at, updated_at FROM users WHERE id = $1"
	row := db.handle.QueryRow(query, id)

	var user User
	if err := row.Scan(&user.ID, &user.Name, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, &UserNotFound{ID: id}
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}
	return &user, nil
}

// ----------------------------------------------------------------
// Get a user by name
// ----------------------------------------------------------------
func (db *Database) GetUserByName(name string) (*User, error) {
	query := "SELECT id, name, password, created_at, updated_at FROM users WHERE name = $1"
	row := db.handle.QueryRow(query, name)

	var user User
	if err := row.Scan(&user.ID, &user.Name, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, &UserNotFound{Name: name}
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}
	return &user, nil
}

// ----------------------------------------------------------------
// Create a new user
// ----------------------------------------------------------------
func (db *Database) CreateUser(user *User) error {
	encryptedPassword, err := GenerateHash(user.Password)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	query := "INSERT INTO users (name, password, created_at, updated_at) VALUES ($1, $2, $3, $4)"
	_, err = db.handle.Exec(query, user.Name, encryptedPassword, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// ----------------------------------------------------------------
// MOEX watchlist management
// ----------------------------------------------------------------
// Get MOEX watchlist from the database
// ----------------------------------------------------------------
func (db *Database) GetMOEXWatchlist(activeOnly bool) ([]MOEXWatchlistItem, error) {
	var rows *sql.Rows
	var err error
	if activeOnly {
		slog.Debug("Retrieving active MOEX watchlist items")
		rows, err = db.handle.Query("SELECT moex_assets.ticker, moex_assets.class_id, moex_watchlist.notification_id, moex_watchlist.target_price::numeric, moex_watchlist.condition, moex_watchlist.is_active FROM moex_watchlist INNER JOIN moex_assets ON moex_watchlist.ticker_id = moex_assets.ticker WHERE moex_watchlist.is_active = true")
	} else {
		slog.Debug("Retrieving all MOEX watchlist items")
		rows, err = db.handle.Query("SELECT moex_assets.ticker, moex_assets.class_id, moex_watchlist.notification_id, moex_watchlist.target_price::numeric, moex_watchlist.condition, moex_watchlist.is_active FROM moex_watchlist INNER JOIN moex_assets ON moex_watchlist.ticker_id = moex_assets.ticker")
	}
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

// ----------------------------------------------------------------
func (db *Database) GetNotifications() ([]Notification, error) {
	query := "SELECT * FROM notifications"
	rows, err := db.handle.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Errorf("failed to close rows: %v", err)
		}
	}()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.TelegramBotID, &n.TelegramChatID, &n.SmtpHost, &n.SmtpPort, &n.SmtpUser, &n.SmtpPass, &n.SmtpFrom, &n.SmtpEncryptionType, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

// ----------------------------------------------------------------
func (db *Database) GetNotificationByID(id int) (*Notification, error) {
	query := "SELECT * FROM notifications WHERE id = $1"
	row := db.handle.QueryRow(query, id)

	var n Notification
	if err := row.Scan(&n.ID, &n.TelegramBotID, &n.TelegramChatID, &n.SmtpHost, &n.SmtpPort, &n.SmtpUser, &n.SmtpPass, &n.SmtpFrom, &n.SmtpEncryptionType, &n.CreatedAt, &n.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("notification with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}
	return &n, nil
}
