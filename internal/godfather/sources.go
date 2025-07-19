package godfather

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type H map[string]interface{}

func GetSources(db *Database) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return ctx.JSON(http.StatusOK, "sources")
	}
}

func PutSource(db *Database) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return ctx.JSON(http.StatusCreated, H{"created": 123})
	}
}

func DeleteSource(db *Database) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		id, err := strconv.Atoi(ctx.Param("id"))
		if err != nil {
			slog.Error(fmt.Sprintf("invalid ID on DeleteSource: %s", ctx.Param("id")))
			return ctx.JSON(http.StatusBadRequest, "")
		}
		return ctx.JSON(http.StatusOK, H{"deleted": id})
	}
}
