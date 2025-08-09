package godfather

import (
	"errors"
	"testing"

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
