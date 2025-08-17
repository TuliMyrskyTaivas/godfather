package godfather

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// ----------------------------------------------------------------
func TestGetMOEXWatchlist_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	rows1 := sqlmock.NewRows([]string{"ticker", "class_id", "notification_id", "target_price", "condition", "is_active"}).
		AddRow("SBER", "stock", 1, 250.5, "above", true).
		AddRow("GAZP", "stock", 2, 150.0, "below", false)
	rows2 := sqlmock.NewRows([]string{"ticker", "class_id", "notification_id", "target_price", "condition", "is_active"}).
		AddRow("SBER", "stock", 1, 250.5, "above", true).
		AddRow("GAZP", "stock", 2, 150.0, "below", false)

	mock.ExpectQuery("SELECT moex_assets.ticker, moex_assets.class_id, moex_watchlist.notification_id, moex_watchlist.target_price::numeric, moex_watchlist.condition, moex_watchlist.is_active FROM moex_watchlist INNER JOIN moex_assets ON moex_watchlist.ticker_id = moex_assets.ticker WHERE moex_watchlist.is_active = true").
		WillReturnRows(rows1)
	mock.ExpectQuery("SELECT moex_assets.ticker, moex_assets.class_id, moex_watchlist.notification_id, moex_watchlist.target_price::numeric, moex_watchlist.condition, moex_watchlist.is_active FROM moex_watchlist INNER JOIN moex_assets ON moex_watchlist.ticker_id = moex_assets.ticker").
		WillReturnRows(rows2)

	database := &Database{handle: db}

	// Test successful retrieval of active watchlist items only
	watchlist, err := database.GetMOEXWatchlist(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(watchlist) != 2 {
		t.Errorf("expected 2 items, got %d", len(watchlist))
	}
	if watchlist[0].Ticker != "SBER" {
		t.Errorf("unexpected tickers: %+v", watchlist)
	}

	// Test successful retrieval of all watchlist items
	watchlist, err = database.GetMOEXWatchlist(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(watchlist) != 2 {
		t.Errorf("expected 2 items, got %d", len(watchlist))
	}
	if watchlist[0].Ticker != "SBER" || watchlist[1].Ticker != "GAZP" {
		t.Errorf("unexpected tickers: %+v", watchlist)
	}
}

// ----------------------------------------------------------------
func TestGetMOEXWatchlist_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery("SELECT moex_assets.ticker").
		WillReturnError(errors.New("query failed"))

	database := &Database{handle: db}
	_, err = database.GetMOEXWatchlist(false)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ----------------------------------------------------------------
func TestGetMOEXWatchlist_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	columns := []string{"ticker", "notification_id", "target_price", "condition", "is_active"}
	mock.ExpectQuery("SELECT moex_assets.ticker").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("SBER", "not-an-int", 250.5, "above", true))

	database := &Database{handle: db}
	_, err = database.GetMOEXWatchlist(false)
	if err == nil {
		t.Error("expected scan error, got nil")
	}
}

// ----------------------------------------------------------------
func TestInitDBFromEnv_MissingEnv(t *testing.T) {
	t.Setenv("GODFATHER_DB_CONN", "")
	database, err := InitDBFromEnv()
	if err == nil {
		t.Error("expected error for missing env var, got nil")
	}
	if database != nil {
		t.Error("expected nil database on error")
	}
}

// ----------------------------------------------------------------
func TestInitDBFromEnv_OpenError(t *testing.T) {
	t.Setenv("GODFATHER_DB_CONN", "bad-conn-string")

	database, err := InitDBFromEnv()
	if err == nil || database != nil {
		t.Error("expected error from sql.Open")
	}
}

// ----------------------------------------------------------------
func TestSetMOEXWatchlistItemActiveStatus_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	ticker := "SBER"
	active := true
	mock.ExpectExec("UPDATE moex_watchlist SET is_active =").
		WithArgs(active, ticker).
		WillReturnResult(sqlmock.NewResult(1, 1))

	database := &Database{handle: db}
	err = database.SetMOEXWatchlistItemActiveStatus(ticker, active)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ----------------------------------------------------------------
func TestSetMOEXWatchlistItemActiveStatus_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	ticker := "GAZP"
	active := false
	mock.ExpectExec("UPDATE moex_watchlist SET is_active =").
		WithArgs(active, ticker).
		WillReturnError(errors.New("exec failed"))

	database := &Database{handle: db}
	err = database.SetMOEXWatchlistItemActiveStatus(ticker, active)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ----------------------------------------------------------------
func TestGetUserByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	rows := sqlmock.NewRows([]string{"id", "name", "password", "created_at", "updated_at"}).
		AddRow(1, "alice", "hashedpass", time.Now(), time.Now())
	mock.ExpectQuery("SELECT id, name, password, created_at::timestamp, updated_at::timestamp FROM users WHERE id =").
		WithArgs(1).
		WillReturnRows(rows)

	database := &Database{handle: db}
	user, err := database.GetUserByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Name != "alice" {
		t.Errorf("expected name alice, got %s", user.Name)
	}
}

// ----------------------------------------------------------------
func TestGetUserByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery("SELECT id, name, password, created_at::timestamp, updated_at::timestamp FROM users WHERE id =").
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "password", "created_at", "updated_at"}))

	database := &Database{handle: db}
	user, err := database.GetUserByID(2)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if user != nil {
		t.Error("expected nil user")
	}
}

// ----------------------------------------------------------------
func TestGetUserByName_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	rows := sqlmock.NewRows([]string{"id", "name", "password", "created_at", "updated_at"}).
		AddRow(1, "bob", "hashedpass", time.Now(), time.Now())
	mock.ExpectQuery("SELECT id, name, password, created_at::timestamp, updated_at::timestamp FROM users WHERE name =").
		WithArgs("bob").
		WillReturnRows(rows)

	database := &Database{handle: db}
	user, err := database.GetUserByName("bob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Name != "bob" {
		t.Errorf("expected name bob, got %s", user.Name)
	}
}

// ----------------------------------------------------------------
func TestGetUserByName_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery("SELECT id, name, password, created_at::timestamp, updated_at::timestamp FROM users WHERE name =").
		WithArgs("charlie").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "password", "created_at", "updated_at"}))

	database := &Database{handle: db}
	user, err := database.GetUserByName("charlie")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if user != nil {
		t.Error("expected nil user")
	}
}

// ----------------------------------------------------------------
func TestCreateUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	mock.ExpectExec("INSERT INTO users").
		WithArgs("dave", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	database := &Database{handle: db}
	user := &User{Name: "dave", Password: "secret"}
	err = database.CreateUser(user)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ----------------------------------------------------------------
func TestCreateUser_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	mock.ExpectExec("INSERT INTO users").
		WithArgs("frank", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("insert failed"))

	database := &Database{handle: db}
	user := &User{Name: "frank", Password: "secret"}
	err = database.CreateUser(user)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ----------------------------------------------------------------
func TestGetUsers_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	rows := sqlmock.NewRows([]string{"id", "name", "password", "created_at", "updated_at"}).
		AddRow(1, "alice", "pass1", time.Now(), time.Now()).
		AddRow(2, "bob", "pass2", time.Now(), time.Now())
	mock.ExpectQuery("SELECT id, name, password, created_at::timestamp, updated_at::timestamp FROM users").
		WillReturnRows(rows)

	database := &Database{handle: db}
	users, err := database.GetUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

// ----------------------------------------------------------------
func TestGetUsers_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery("SELECT id, name, password, created_at::timestamp, updated_at::timestamp FROM users").
		WillReturnError(errors.New("query failed"))

	database := &Database{handle: db}
	_, err = database.GetUsers()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ----------------------------------------------------------------
func TestGetUsers_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	rows := sqlmock.NewRows([]string{"id", "name", "password", "created_at", "updated_at"}).
		AddRow("not-an-int", "bob", "pass", "2023-01-01 00:00:00", "2023-01-02 00:00:00")
	mock.ExpectQuery("SELECT id, name, password, created_at::timestamp, updated_at::timestamp FROM users").
		WillReturnRows(rows)

	database := &Database{handle: db}
	_, err = database.GetUsers()
	if err == nil {
		t.Error("expected scan error, got nil")
	}
}

// ----------------------------------------------------------------
func TestUpdateUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	mock.ExpectExec("UPDATE users SET name =").
		WithArgs("alice", "hashedpass", sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	database := &Database{handle: db}
	user := &User{ID: 1, Name: "alice", Password: "hashedpass"}
	err = database.UpdateUser(user)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ----------------------------------------------------------------
func TestUpdateUser_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	mock.ExpectExec("UPDATE users SET name =").
		WithArgs("bob", "hashedpass", sqlmock.AnyArg(), 2).
		WillReturnError(errors.New("update failed"))

	database := &Database{handle: db}
	user := &User{ID: 2, Name: "bob", Password: "hashedpass"}
	err = database.UpdateUser(user)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ----------------------------------------------------------------
func TestDeleteUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	mock.ExpectExec("DELETE FROM users WHERE id =").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	database := &Database{handle: db}
	err = database.DeleteUser(1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ----------------------------------------------------------------
func TestDeleteUser_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	mock.ExpectExec("DELETE FROM users WHERE id =").
		WithArgs(2).
		WillReturnError(errors.New("delete failed"))

	database := &Database{handle: db}
	err = database.DeleteUser(2)
	if err == nil {
		t.Error("expected error, got nil")
	}
}
