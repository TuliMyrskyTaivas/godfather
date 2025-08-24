package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/TuliMyrskyTaivas/godfather/internal/godfather"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// WebUser represents the user model for the web
type UserRequest struct {
	ID       int    `json:"id"`
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Password string `json:"password" validate:"required,min=10,max=100"`
}

// ----------------------------------------------------------------
func (r *UserRequest) Validate() error {
	if r.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Name is required")
	}
	if r.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Password is required")
	}
	return nil
}

// ----------------------------------------------------------------
func createUserHandler(db *godfather.Database) echo.HandlerFunc {
	return func(c echo.Context) error {
		token := c.Get("user").(*jwt.Token)
		claims := token.Claims.(*JWTClaims)
		userID := claims.UserID

		u := new(UserRequest)
		if err := c.Bind(u); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
		}

		if err := c.Validate(u); err != nil {
			return err
		}

		slog.Debug(fmt.Sprintf("User %d is creating a new user %s", userID, u.Name))

		// Check if user already exists
		_, err := db.GetUserByName(u.Name)
		if err != nil {
			if err != err.(*godfather.UserNotFound) {
				slog.Error("Failed to check existing user", "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "Database error 0x3")
			}
			return echo.NewHTTPError(http.StatusConflict, "User already exists")
		}

		hashedPassword, err := godfather.GenerateHash(u.Password)
		if err != nil {
			slog.Error("Failed to generate password hash", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Internal error 0x1")
		}

		// Create user
		if err := db.CreateUser(&godfather.User{
			Name:     u.Name,
			Password: hashedPassword,
		}); err != nil {
			slog.Error("Failed to create user", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Database error")
		}

		// Return the created user (without password in real app)
		return c.JSON(http.StatusCreated, u)
	}
}

// ----------------------------------------------------------------
func getUsersHandler(db *godfather.Database) echo.HandlerFunc {
	return func(c echo.Context) error {
		users, err := db.GetUsers()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Database error")
		}
		return c.JSON(http.StatusOK, users)
	}
}

// ----------------------------------------------------------------
func updateUserHandler(db *godfather.Database) echo.HandlerFunc {
	return func(c echo.Context) error {
		u := new(UserRequest)
		if err := c.Bind(u); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
		}

		if err := c.Validate(u); err != nil {
			return err
		}

		token := c.Get("user").(*jwt.Token)
		claims := token.Claims.(*JWTClaims)
		userID := claims.UserID
		slog.Debug(fmt.Sprintf("User %d is updating user %s", userID, u.Name))

		// Check if user exists
		existingUser, err := db.GetUserByID(u.ID)
		if err != nil {
			if err != err.(*godfather.UserNotFound) {
				slog.Error("Failed to retrieve user by ID", "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "Database error 0x3")
			}
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}

		// Hash the new password
		hashedPassword, err := godfather.GenerateHash(u.Password)
		if err != nil {
			slog.Error("Failed to generate password hash", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Internal error 0x1")
		}

		// Update user
		existingUser.Name = u.Name
		existingUser.Password = hashedPassword
		if err := db.UpdateUser(existingUser); err != nil {
			slog.Error("Failed to update user", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Database error 0x4")
		}

		return c.JSON(http.StatusOK, existingUser)
	}
}

// ----------------------------------------------------------------
func deleteUserHandler(db *godfather.Database) echo.HandlerFunc {
	return func(c echo.Context) error {
		u := new(UserRequest)
		if err := c.Bind(u); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
		}

		if err := c.Validate(u); err != nil {
			return err
		}

		token := c.Get("user").(*jwt.Token)
		claims := token.Claims.(*JWTClaims)
		userID := claims.UserID
		slog.Debug(fmt.Sprintf("User %d is deleting user %s", userID, u.Name))

		// Check if user exists
		_, err := db.GetUserByID(u.ID)
		if err != nil {
			if err != err.(*godfather.UserNotFound) {
				slog.Error("Failed to retrieve user by ID", "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "Database error 0x3")
			}
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}

		// Delete user
		if err := db.DeleteUser(u.ID); err != nil {
			slog.Error("Failed to delete user", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Database error 0x4")
		}

		return c.NoContent(http.StatusNoContent)
	}
}
