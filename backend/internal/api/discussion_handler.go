package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/digital-twin-community/backend/internal/discussion"
	apimiddleware "github.com/digital-twin-community/backend/internal/middleware"
	"github.com/digital-twin-community/backend/internal/topic"
)

// DiscussionHandler handles /discussions endpoints.
type DiscussionHandler struct {
	repo      discussion.Repository
	topicRepo topic.Repository
}

// NewDiscussionHandler constructs a DiscussionHandler.
func NewDiscussionHandler(repo discussion.Repository, topicRepo topic.Repository) *DiscussionHandler {
	return &DiscussionHandler{repo: repo, topicRepo: topicRepo}
}

// Get handles GET /discussions/:id.
func (h *DiscussionHandler) Get(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	d, err := h.repo.FindByID(c.Request().Context(), id)
	if err != nil {
		return httpError(err)
	}
	if d == nil {
		return echo.NewHTTPError(http.StatusNotFound, "discussion not found")
	}
	// Verify ownership through the associated topic
	t, err := h.topicRepo.FindByID(c.Request().Context(), d.TopicID)
	if err != nil || t == nil || t.SubmitterUserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
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
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	// First fetch the discussion to get topic_id for ownership check
	d, err := h.repo.FindByID(c.Request().Context(), id)
	if err != nil {
		return httpError(err)
	}
	if d == nil {
		return echo.NewHTTPError(http.StatusNotFound, "discussion not found")
	}
	t, err := h.topicRepo.FindByID(c.Request().Context(), d.TopicID)
	if err != nil || t == nil || t.SubmitterUserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
	}
	msgs, err := h.repo.FindMessages(c.Request().Context(), id)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, msgs)
}
