// Package api contains HTTP handlers for the API v1 routes.
package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/digital-twin-community/backend/internal/auth"
	"github.com/digital-twin-community/backend/internal/connection"
)

// httpError maps domain errors to Echo HTTP errors.
func httpError(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, auth.ErrUserAlreadyExists):
		return echo.NewHTTPError(http.StatusConflict, "user already exists")
	case errors.Is(err, auth.ErrInvalidCredentials):
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, auth.ErrUserNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	case errors.Is(err, auth.ErrTokenExpired):
		return echo.NewHTTPError(http.StatusUnauthorized, "token expired")
	case errors.Is(err, auth.ErrTokenInvalid):
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	case errors.Is(err, connection.ErrAlreadyConnected):
		return echo.NewHTTPError(http.StatusConflict, "connection already exists")
	case errors.Is(err, connection.ErrConnectionNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "connection not found")
	case errors.Is(err, connection.ErrNotAuthorized):
		return echo.NewHTTPError(http.StatusForbidden, "not authorized")
	case errors.Is(err, connection.ErrConnectionExpired):
		return echo.NewHTTPError(http.StatusGone, "connection request expired")
	case errors.Is(err, connection.ErrContactNotAvailable):
		return echo.NewHTTPError(http.StatusForbidden, "contact not available until connection is accepted")
	}
	// PostgreSQL invalid UUID / enum input → treat as not found
	if strings.Contains(err.Error(), "invalid input syntax for type uuid") ||
		strings.Contains(err.Error(), "invalid input syntax for type") {
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	}
	return echo.NewHTTPError(http.StatusInternalServerError, "internal server error").SetInternal(err)
}
