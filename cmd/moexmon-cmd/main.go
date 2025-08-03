package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/TuliMyrskyTaivas/godfather/internal/godfather"
	slogformatter "github.com/samber/slog-formatter"
)

// ///////////////////////////////////////////////////////////////////
// Setup a global logger
// ///////////////////////////////////////////////////////////////////
func setupLogger(verbose bool) *slog.Logger {
	var log *slog.Logger
	var logOptions slog.HandlerOptions

	if verbose {
		logOptions.Level = slog.LevelDebug
	} else {
		logOptions.Level = slog.LevelInfo
	}

	log = slog.New(slogformatter.NewFormatterHandler(
		slogformatter.TimezoneConverter(time.UTC),
		slogformatter.TimeFormatter(time.RFC3339, nil),
	)(
		slog.NewTextHandler(os.Stdout, &logOptions),
	))

	slog.SetDefault(log)
	return log
}

func main() {
	var verbose bool
	var help bool

	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()

	if help {
		flag.Usage()
		os.Exit(0)
	}

	logger := setupLogger(verbose)

	// Initialize the database connection
	db, err := godfather.InitDBFromEnv()
	if err != nil {
		logger.Error("Failed to initialize database connection", "error", err)
		os.Exit(1)
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection", "error", err)
		} else {
			logger.Info("Database connection closed successfully")
		}
	}()

	// Get MOEX watchlist
	watchlist, err := db.GetMOEXWatchlist()
	if err != nil {
		logger.Error("Failed to retrieve MOEX watchlist", "error", err)
		return
	}
	logger.Info("Retrieved MOEX watchlist", "count", len(watchlist))
}
