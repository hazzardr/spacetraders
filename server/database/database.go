package database

import (
	"fmt"
	"github.com/hazzardr/spacetraders/generated/domain"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"net/http"
)

type Operations struct {
	DB      *pgxpool.Pool
	Queries *domain.Queries
}

func (dbo *Operations) HandlePGError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgerrcode.UniqueViolation {
			return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Cannot insert new record. code=%s, message=%s", pgErr.Code, pgErr.Message))
		} else if pgerrcode.IsConnectionException(pgErr.Code) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("Database connection error, please try again later. code=%s, message=%s", pgErr.Code, pgErr.Message))
		} else if pgerrcode.IsDataException(pgErr.Code) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, fmt.Sprintf("Failure processing request. code=%s, message=%s", pgErr.Code, pgErr.Message))
		} else {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Unhandled Postgres error. code=%s message=%s", pgErr.Code, pgErr.Message))
		}
	}

	return err
}
