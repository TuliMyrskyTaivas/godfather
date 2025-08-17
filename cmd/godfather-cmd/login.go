package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/TuliMyrskyTaivas/godfather/internal/godfather"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// ----------------------------------------------------------------
// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

// ----------------------------------------------------------------
type LoginRequest struct {
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// ----------------------------------------------------------------
func (r *LoginRequest) Validate() error {
	if r.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Name is required")
	}
	if r.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Password is required")
	}
	return nil
}

// ----------------------------------------------------------------
func LoginHandler(db *godfather.Database, jwtSecret string) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := new(LoginRequest)
		if err := c.Bind(req); err != nil {
			slog.Error("Failed to bind login request", "error", err)
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
		}

		if err := c.Validate(req); err != nil {
			slog.Error("Validation failed for login request", "error", err)
			return err
		}

		dbUser, err := db.GetUserByName(req.Name)
		if err != nil {
			if err == err.(*godfather.UserNotFound) {
				slog.Error("Login not found", "name", req.Name)
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
			}
			slog.Error("Failed to retrieve user by name", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Database error 0x1")
		}

		validPassword, err := godfather.VerifyPassword(req.Password, dbUser.Password)
		if err != nil {
			slog.Error("Failed to verify password", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Database error 0x2")
		}

		if !validPassword {
			slog.Error("Invalid password for user", "name", req.Name, "password", req.Password, "hashed", dbUser.Password)
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
		}

		// Set custom claims
		claims := &JWTClaims{
			UserID: dbUser.ID,
			Name:   dbUser.Name,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 2)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				NotBefore: jwt.NewNumericDate(time.Now()),
			},
		}

		// Create token with claims
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		// Generate encoded token and send it as response
		t, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			slog.Error("Failed to generate token", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Internal error 0x1")
		}

		slog.Debug("User logged in successfully", "user_id", dbUser.ID, "name", dbUser.Name)
		return c.JSON(http.StatusOK, map[string]string{
			"token": t,
		})
	}
}
