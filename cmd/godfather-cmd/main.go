package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"github.com/TuliMyrskyTaivas/godfather/internal/godfather"
	slogecho "github.com/samber/slog-echo"
	slogformatter "github.com/samber/slog-formatter"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	sha1ver   string // sha1 revision used to build the program
	buildTime string // when the executable was built
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

// ///////////////////////////////////////////////////////////////////
// Entry point
// ///////////////////////////////////////////////////////////////////
func main() {
	var configPath string
	var verbose bool
	var help bool

	flag.StringVar(&configPath, "c", "", "path to the configuration file")
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()

	if help {
		fmt.Printf("Usage: %s [OPTIONS]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

	logger := setupLogger(verbose)

	buildInfo, _ := debug.ReadBuildInfo()
	slog.Debug(fmt.Sprintf("Built by %s at %s (SHA1=%s)", buildInfo.GoVersion, buildTime, sha1ver))

	config, err := ReadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	db, err := godfather.InitDB(config.Database.Host, config.Database.Port, config.Database.User,
		config.Database.Passwd, config.Database.Database)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Migrate()
	if err != nil {
		log.Fatal(err)
	}

	service := echo.New()
	service.Use(slogecho.New(logger.WithGroup("http")))
	service.Use(middleware.Recover())

	service.File("/*", "brower/index.html")
	service.Static("/", "browser")

	service.GET("/sources", godfather.GetSources(db))
	service.PUT("/sources", godfather.PutSource(db))
	service.DELETE("/sources/:id", godfather.DeleteSource(db))

	err = service.StartTLS(fmt.Sprintf("%s:%d", config.Interface.Addr, config.Interface.Port),
		config.Interface.TLS.Cert, config.Interface.TLS.Key)
	if err != nil {
		log.Fatal(err)
	}
}
