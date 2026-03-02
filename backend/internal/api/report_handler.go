package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	apimiddleware "github.com/digital-twin-community/backend/internal/middleware"
	"github.com/digital-twin-community/backend/internal/report"
	"github.com/digital-twin-community/backend/internal/topic"
)

// ReportHandler handles /reports endpoints.
type ReportHandler struct {
	repo      report.Repository
	topicRepo topic.Repository
}

// NewReportHandler constructs a ReportHandler.
func NewReportHandler(repo report.Repository, topicRepo topic.Repository) *ReportHandler {
	return &ReportHandler{repo: repo, topicRepo: topicRepo}
}

// Get handles GET /reports/:id.
func (h *ReportHandler) Get(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	r, err := h.repo.FindByID(c.Request().Context(), id)
	if err != nil {
		return httpError(err)
	}
	if r == nil {
		return echo.NewHTTPError(http.StatusNotFound, "report not found")
	}
	// Verify ownership through the associated topic
	t, err := h.topicRepo.FindByID(c.Request().Context(), r.TopicID)
	if err != nil || t == nil || t.SubmitterUserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
	}
	return c.JSON(http.StatusOK, r)
}

// Rate handles POST /reports/:id/rating.
func (h *ReportHandler) Rate(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	// Verify ownership before allowing rating
	r, err := h.repo.FindByID(c.Request().Context(), id)
	if err != nil {
		return httpError(err)
	}
	if r == nil {
		return echo.NewHTTPError(http.StatusNotFound, "report not found")
	}
	t, err := h.topicRepo.FindByID(c.Request().Context(), r.TopicID)
	if err != nil || t == nil || t.SubmitterUserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
	}
	var body struct {
		Rating   int    `json:"rating"`
		Feedback string `json:"feedback,omitempty"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if body.Rating < 1 || body.Rating > 5 {
		return echo.NewHTTPError(http.StatusBadRequest, "rating must be between 1 and 5")
	}
	if len(body.Feedback) > 5000 {
		return echo.NewHTTPError(http.StatusBadRequest, "feedback too long (max 5000 chars)")
	}
	if err := h.repo.UpdateUserRating(c.Request().Context(), id, body.Rating, body.Feedback); err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
