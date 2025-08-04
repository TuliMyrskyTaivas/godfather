package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TuliMyrskyTaivas/godfather/internal/godfather"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	slogformatter "github.com/samber/slog-formatter"
)

// ----------------------------------------------------------------
var dbFailures = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "moexmon_db_failures",
		Help: "Number of database failures when retrieving MOEX watchlist",
	},
)
var moexFailures = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "moexmon_net_failures",
		Help: "Number of failures when querying MOEX",
	},
)

// ----------------------------------------------------------------
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

// ----------------------------------------------------------------
func startMetrics(ctx context.Context, url string, port int) {
	slog.Info(fmt.Sprintf("Starting Prometheus metrics server at http://localhost:%d/%s...", port, url))

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(dbFailures)

	http.Handle(url, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		ReadHeaderTimeout: 5 * time.Second, // Prevent Slowloris attacks
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Prometheus metrics server failed", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("Prometheus metrics server stopped")

	// Gracefully shutdown the server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

// ----------------------------------------------------------------
func startMonitoring(ctx context.Context, moex MoexQuery, db *godfather.Database, interval_sec int) {
	slog.Info("Starting MOEX monitoring...")

	ticker := time.NewTicker(time.Duration(interval_sec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Monitoring stopped due to context cancellation")
			return
		case <-ticker.C:
			watchlist, err := db.GetMOEXWatchlist()
			if err != nil {
				slog.Error("Failed to retrieve MOEX watchlist", "error", err)
				dbFailures.Inc()
				continue
			}
			slog.Debug(fmt.Sprintf("MOEX watchlist retrieved with %d items", len(watchlist)))

			// Fetch prices for stocks, bonds, and currencies
			stockPrices, err := moex.FetchStocks(ctx)
			if err != nil {
				slog.Error("Failed to fetch stock prices", "error", err)
				moexFailures.Inc()
				continue
			}
			slog.Debug(fmt.Sprintf("Fetched stock prices for %d stocks", len(stockPrices)))
			bondPrices, err := moex.FetchBonds(ctx)
			if err != nil {
				slog.Error("Failed to fetch bond prices", "error", err)
				moexFailures.Inc()
				continue
			}
			slog.Debug(fmt.Sprintf("Fetched bond prices for %d bonds", len(bondPrices)))
			currencyPrices, err := moex.FetchCurrencies(ctx)
			if err != nil {
				slog.Error("Failed to fetch currency prices", "error", err)
				moexFailures.Inc()
				continue
			}
			slog.Debug(fmt.Sprintf("Fetched currency prices for %d currencies", len(currencyPrices)))

			for _, watchlistItem := range watchlist {
				price, exists := stockPrices[watchlistItem.Ticker]
				if !exists {
					slog.Warn(fmt.Sprintf("Price for watchlist item %s not found in MOEX", watchlistItem.Ticker))
					continue
				}

				slog.Info(fmt.Sprintf("Current price for %s: %.2f", watchlistItem.Ticker, price))
			}
		}
	}
}

// ----------------------------------------------------------------
func main() {
	var configPath string
	var verbose bool
	var help bool

	flag.StringVar(&configPath, "c", "moexmon.json", "path to config file")
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()

	if help {
		flag.Usage()
		os.Exit(0)
	}

	logger := setupLogger(verbose)

	config, err := ParseConfig(configPath)
	if err != nil {
		logger.Error("Failed to parse configuration", "error", err)
		os.Exit(1)
	}

	// Create a context that will be canceled on interrupt/termination
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,    // SIGINT (Ctrl+C)
		syscall.SIGTERM, // Kubernetes/Systemd termination
		syscall.SIGQUIT, // Graceful shutdown
	)
	defer stop() // Release signal resources when main exits

	// Initialize the database connection
	db, err := godfather.InitDBFromEnv()
	if err != nil {
		logger.Error("Failed to initialize database connection", "error", err)
		return
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection", "error", err)
		}
	}()

	// Create a new MOEX requester
	moexRequester := newMoexRequester()

	// Start the routines
	go startMetrics(ctx, config.Prometheus.URL, config.Prometheus.Port)
	go startMonitoring(ctx, moexRequester, db, config.CheckIntervalSeconds)

	// Wait for the signal to stop
	<-ctx.Done()
	slog.Info("Received termination signal, shutting down...")
}
