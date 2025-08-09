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
var alertsPublished = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "moexmon_alerts_published",
		Help: "Number of alerts published to NATS",
	},
)
var alertFailures = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "moexmon_alert_failures",
		Help: "Number of failures when publishing alerts to NATS",
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
	slog.Info(fmt.Sprintf("Starting Prometheus metrics server at http://localhost:%d%s...", port, url))

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
func conditionMatch(ctx context.Context, item godfather.MOEXWatchlistItem, moex MoexQuery) bool {
	price, err := moex.FetchPrice(ctx, item.Ticker, item.AssetClass)
	if err != nil {
		if _, ok := err.(*AssetNotFoundError); ok {
			slog.Warn(fmt.Sprintf("Asset %s not found on MOEX", item.Ticker))
		} else {
			slog.Error(fmt.Sprintf("Failed to fetch price for %s: %s", item.Ticker, err.Error()))
			moexFailures.Inc()
		}
		return false
	}
	slog.Debug(fmt.Sprintf("Current price for %s: %.2f", item.Ticker, price))
	switch item.Condition {
	case "above":
		return price > item.TargetPrice
	case "below":
		return price < item.TargetPrice
	default:
		return false
	}
}

// ----------------------------------------------------------------
func startMonitoring(ctx context.Context, moex MoexQuery, db *godfather.Database, mb *godfather.MessageBus, interval_sec int) {
	slog.Info(fmt.Sprintf("Starting MOEX monitoring, check interval is %d seconds...", interval_sec))

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
			// TODO: now GetMOEXWatchlist returns all items, filter them by active status
			slog.Debug(fmt.Sprintf("MOEX watchlist retrieved %d active items", len(watchlist)))
			for _, watchlistItem := range watchlist {
				if conditionMatch(ctx, watchlistItem, moex) {
					slog.Info(fmt.Sprintf("Condition met for %s, deactivating watchlist item", watchlistItem.Ticker))
					err := db.SetMOEXWatchlistItemActiveStatus(watchlistItem.Ticker, false)
					if err != nil {
						slog.Error("Failed to deactivate watchlist item", "error", err)
						dbFailures.Inc()
					}
					alert := fmt.Sprintf("The price for %s is %s %.2f", watchlistItem.Ticker, watchlistItem.Condition, watchlistItem.TargetPrice)
					err = mb.PublishAlert("alerts.MOEX", []byte(alert))
					if err != nil {
						slog.Error("Failed to publish alert", "error", err)
						alertFailures.Inc()
					}
					alertsPublished.Inc()
					slog.Debug("Alert published", "message", alert)
				}
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

	// Initialize the message bus (NATS)
	mb, err := godfather.NewMessageBus(config.NATS.Host, config.NATS.Port, config.NATS.User)
	if err != nil {
		logger.Error("Failed to initialize message bus", "error", err)
		return
	}
	defer mb.Close()

	// Create a new MOEX requester
	moexRequester := newMoexRequester()

	// Start the routines
	go startMetrics(ctx, config.Prometheus.URL, config.Prometheus.Port)
	go startMonitoring(ctx, moexRequester, db, mb, config.CheckIntervalSeconds)

	// Wait for the signal to stop
	<-ctx.Done()
	slog.Info("Received termination signal, shutting down...")
}
