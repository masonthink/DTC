// Package middleware provides Echo middleware for the API server.
package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const contextKeyUserID = "user_id"

// TokenValidator validates a JWT token string and returns the user ID.
type TokenValidator interface {
	ValidateAccessToken(token string) (string, error)
}

// JWTAuth returns middleware that validates Bearer tokens and injects userID into context.
func JWTAuth(validator TokenValidator, logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
			}

			userID, err := validator.ValidateAccessToken(parts[1])
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set(contextKeyUserID, userID)
			return next(c)
		}
	}
}

// UserIDFromContext extracts the authenticated user ID from the Echo context.
func UserIDFromContext(c echo.Context) string {
	id, _ := c.Get(contextKeyUserID).(string)
	return id
}

// RequestLogger returns middleware that logs incoming requests.
func RequestLogger(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			err := next(c)
			res := c.Response()

			logger.Info("request",
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
				zap.Int("status", res.Status),
				zap.String("ip", c.RealIP()),
			)
			return err
		}
	}
}

// RateLimiter returns a simple per-IP rate limiter middleware.
// In production, use Redis-backed sliding window (golang.org/x/time/rate).
func RateLimiter(rps int) echo.MiddlewareFunc {
	// Placeholder: full implementation uses Redis + token bucket
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}
}
