package godfather

import (
	"log/slog"
	"os"
	"time"

	slogformatter "github.com/samber/slog-formatter"
)

// ----------------------------------------------------------------
func SetupLogger(verbose bool) *slog.Logger {
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
