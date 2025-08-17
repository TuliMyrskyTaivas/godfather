package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/TuliMyrskyTaivas/godfather/internal/godfather"
	"github.com/golang-jwt/jwt/v5"
	slogecho "github.com/samber/slog-echo"

	echojwt "github.com/labstack/echo-jwt/v4" // Import echo-jwt separately
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

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

	logger := godfather.SetupLogger(verbose)

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
		TokenLookup: "header:Authorization:Bearer ",
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return &JWTClaims{}
		},
		ErrorHandler: func(c echo.Context, err error) error {
			slog.Error("JWT validation failed", "error", err, "path", c.Path(), "headers", c.Request().Header)
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or expired token")
		},
	}))

	// Middleware to log incoming JWT tokens (for debugging purposes)
	r.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				slog.Debug("Incoming token",
					"token", tokenString,
					"path", c.Path())

				// Parse without validation to inspect claims
				token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &JWTClaims{})
				if err == nil {
					if claims, ok := token.Claims.(*JWTClaims); ok {
						slog.Debug("Token claims",
							"user_id", claims.UserID,
							"name", claims.Name,
							"expires_at", claims.ExpiresAt,
							"issued_at", claims.IssuedAt)
					}
				}
			}
			return next(c)
		}
	})

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
