package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/digital-twin-community/backend/internal/topic"
	apimiddleware "github.com/digital-twin-community/backend/internal/middleware"
)

// TopicHandler handles /topics endpoints.
type TopicHandler struct {
	svc *topic.Service
}

// NewTopicHandler constructs a TopicHandler.
func NewTopicHandler(svc *topic.Service) *TopicHandler {
	return &TopicHandler{svc: svc}
}

// Submit handles POST /topics.
func (h *TopicHandler) Submit(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	var req topic.SubmitRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.SubmitterUserID = userID
	t, err := h.svc.Submit(c.Request().Context(), req)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusCreated, t)
}

// List handles GET /topics?limit=20&offset=0.
func (h *TopicHandler) List(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	limit := queryInt(c, "limit", 20)
	offset := queryInt(c, "offset", 0)
	topics, err := h.svc.ListByUser(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, topics)
}

// Get handles GET /topics/:id.
func (h *TopicHandler) Get(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	t, err := h.svc.GetByID(c.Request().Context(), id, userID)
	if err != nil {
		return httpError(err)
	}
	if t == nil {
		return echo.NewHTTPError(http.StatusNotFound, "topic not found")
	}
	return c.JSON(http.StatusOK, t)
}

// Cancel handles DELETE /topics/:id.
func (h *TopicHandler) Cancel(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	if err := h.svc.Cancel(c.Request().Context(), id, userID); err != nil {
		return httpError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// queryInt extracts an integer query parameter with a default value.
func queryInt(c echo.Context, key string, def int) int {
	raw := c.QueryParam(key)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return def
	}
	return v
}
