package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/TuliMyrskyTaivas/godfather/internal/godfather"
	"github.com/golang-jwt/jwt/v5"
	slogecho "github.com/samber/slog-echo"
	slogformatter "github.com/samber/slog-formatter"

	echojwt "github.com/labstack/echo-jwt/v4" // Import echo-jwt separately
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
func setupAdminUser(db *godfather.Database) error {
	adminPasswd, err := godfather.GenerateHash("admin")
	if err != nil {
		return fmt.Errorf("failed to generate admin password hash: %w", err)
	}

	adminUser := &godfather.User{
		Name:     "admin",
		Password: adminPasswd,
	}

	existingUser, err := db.GetUserByName(adminUser.Name)
	if err != nil && err != err.(*godfather.UserNotFound) {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser == nil {
		if err := db.CreateUser(adminUser); err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
	}

	return nil
}

// ----------------------------------------------------------------
type DefaultValidator struct{}

func (v *DefaultValidator) Validate(i interface{}) error {
	if _, ok := i.(interface{ Validate() error }); !ok {
		return nil
	}
	return i.(interface{ Validate() error }).Validate()
}

// ----------------------------------------------------------------
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

	config, err := ReadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Setup database connection
	db, err := godfather.InitDB(config.Database.Host, config.Database.Port, config.Database.User,
		config.Database.Passwd, config.Database.Database)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Migrate()
	if err != nil {
		log.Fatal(err)
	}

	// Create admin user if not exists
	if err := setupAdminUser(db); err != nil {
		log.Fatal(err)
	}

	// Get JWT secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	// Setup HTTP server
	service := echo.New()
	service.Use(slogecho.New(logger.WithGroup("http")))
	service.Use(middleware.Recover())
	service.Validator = &DefaultValidator{}

	service.File("/*", "brower/index.html")
	service.Static("/", "browser")

	// Public routes (no authentication required)
	service.POST("/api/v1/login", LoginHandler(db, jwtSecret))

	// Restricted routes
	r := service.Group("/api/v1")
	r.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  []byte(jwtSecret),
		TokenLookup: "header:Authorization",
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return &JWTClaims{}
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or expired token")
		},
	}))

	// User routes
	r.POST("/users", createUserHandler(db))
	r.GET("/users", getUsersHandler(db))
	r.PUT("/users/:id", updateUserHandler(db))
	r.DELETE("/users/:id", deleteUserHandler(db))

	err = service.StartTLS(fmt.Sprintf("%s:%d", config.Interface.Addr, config.Interface.Port),
		config.Interface.TLS.Cert, config.Interface.TLS.Key)
	if err != nil {
		log.Fatal(err)
	}
}
