package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/digital-twin-community/backend/internal/discussion"
)

// DiscussionHandler handles /discussions endpoints.
type DiscussionHandler struct {
	repo discussion.Repository
}

// NewDiscussionHandler constructs a DiscussionHandler.
func NewDiscussionHandler(repo discussion.Repository) *DiscussionHandler {
	return &DiscussionHandler{repo: repo}
}

// Get handles GET /discussions/:id.
func (h *DiscussionHandler) Get(c echo.Context) error {
	id := c.Param("id")
	d, err := h.repo.FindByID(c.Request().Context(), id)
	if err != nil {
		return httpError(err)
	}
	if d == nil {
		return echo.NewHTTPError(http.StatusNotFound, "discussion not found")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":            d.ID,
		"topic_id":      d.TopicID,
		"status":        d.Status,
		"current_round": d.CurrentRound,
		"participants":  d.Participants,
		"is_degraded":   d.IsDegraded,
	})
}

// GetMessages handles GET /discussions/:id/messages.
func (h *DiscussionHandler) GetMessages(c echo.Context) error {
	id := c.Param("id")
	msgs, err := h.repo.FindMessages(c.Request().Context(), id)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, msgs)
}
