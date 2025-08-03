package godfather

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetMOEXWatchlist_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	columns := []string{"ticker", "notification_id", "target_price", "condition"}
	mock.ExpectQuery("SELECT moex_assets.ticker").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("SBER", 1, 250.5, "above").
			AddRow("GAZP", 2, 150.0, "below"))

	database := &Database{handle: db}
	watchlist, err := database.GetMOEXWatchlist()
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

func TestGetMOEXWatchlist_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery("SELECT moex_assets.ticker").
		WillReturnError(errors.New("query failed"))

	database := &Database{handle: db}
	_, err = database.GetMOEXWatchlist()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetMOEXWatchlist_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	columns := []string{"ticker", "notification_id", "target_price", "condition"}
	mock.ExpectQuery("SELECT moex_assets.ticker").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("SBER", "not-an-int", 250.5, "above"))

	database := &Database{handle: db}
	_, err = database.GetMOEXWatchlist()
	if err == nil {
		t.Error("expected scan error, got nil")
	}
}

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

func TestInitDBFromEnv_OpenError(t *testing.T) {
	t.Setenv("GODFATHER_DB_CONN", "bad-conn-string")

	database, err := InitDBFromEnv()
	if err == nil || database != nil {
		t.Error("expected error from sql.Open")
	}
}
