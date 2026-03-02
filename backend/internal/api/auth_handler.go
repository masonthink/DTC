package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/digital-twin-community/backend/internal/auth"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	svc *auth.Service
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(svc *auth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(c echo.Context) error {
	var req auth.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	user, tokens, err := h.svc.Register(c.Request().Context(), req)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"user":          user,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_at":    tokens.ExpiresAt,
	})
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c echo.Context) error {
	var req auth.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	user, tokens, err := h.svc.Login(c.Request().Context(), req)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user":          user,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_at":    tokens.ExpiresAt,
	})
}

// Refresh handles POST /auth/refresh.
func (h *AuthHandler) Refresh(c echo.Context) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	tokens, err := h.svc.RefreshTokens(c.Request().Context(), body.RefreshToken)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, tokens)
}
