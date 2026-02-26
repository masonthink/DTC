package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/digital-twin-community/backend/internal/auth"
	apimiddleware "github.com/digital-twin-community/backend/internal/middleware"
)

// UserHandler handles /users endpoints.
type UserHandler struct {
	authSvc *auth.Service
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(authSvc *auth.Service) *UserHandler {
	return &UserHandler{authSvc: authSvc}
}

// GetMe handles GET /users/me.
func (h *UserHandler) GetMe(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	user, err := h.authSvc.GetMe(c.Request().Context(), userID)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, user)
}

// UpdateFCMToken handles POST /users/fcm-token.
func (h *UserHandler) UpdateFCMToken(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	var body struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if body.Token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "token required")
	}
	if err := h.authSvc.UpdateFCMToken(c.Request().Context(), userID, body.Token); err != nil {
		return httpError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
