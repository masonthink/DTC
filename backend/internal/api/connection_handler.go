package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/digital-twin-community/backend/internal/connection"
	apimiddleware "github.com/digital-twin-community/backend/internal/middleware"
)

// ConnectionHandler handles /connections endpoints.
type ConnectionHandler struct {
	svc  *connection.Service
	repo connection.Repository
}

// NewConnectionHandler constructs a ConnectionHandler.
func NewConnectionHandler(svc *connection.Service, repo connection.Repository) *ConnectionHandler {
	return &ConnectionHandler{svc: svc, repo: repo}
}

// Request handles POST /connections.
func (h *ConnectionHandler) Request(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	var input connection.RequestInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	input.RequesterUserID = userID
	conn, err := h.svc.Request(c.Request().Context(), input)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusCreated, conn)
}

// List handles GET /connections (merges requester + target views).
func (h *ConnectionHandler) List(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	ctx := c.Request().Context()

	sent, err := h.repo.FindByRequester(ctx, userID, 50, 0)
	if err != nil {
		return httpError(err)
	}
	received, err := h.repo.FindByTarget(ctx, userID, 50, 0)
	if err != nil {
		return httpError(err)
	}

	all := make([]*connection.Connection, 0, len(sent)+len(received))
	all = append(all, sent...)
	all = append(all, received...)
	return c.JSON(http.StatusOK, all)
}

// Respond handles POST /connections/:id/respond.
func (h *ConnectionHandler) Respond(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	var input connection.RespondInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	input.ConnectionID = id
	input.TargetUserID = userID
	conn, err := h.svc.Respond(c.Request().Context(), input)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, conn)
}

// GetContacts handles GET /connections/:id/contacts.
func (h *ConnectionHandler) GetContacts(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	requesterContact, targetContact, err := h.svc.GetContacts(c.Request().Context(), id, userID)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{
		"requester_contact": requesterContact,
		"target_contact":    targetContact,
	})
}
