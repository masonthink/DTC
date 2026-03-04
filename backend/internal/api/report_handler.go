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
// Accepts either a report ID or a topic ID for convenience.
func (h *ReportHandler) Get(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	ctx := c.Request().Context()

	// Try by report ID first, then fall back to topic ID.
	r, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return httpError(err)
	}
	if r == nil {
		r, err = h.repo.FindByTopicID(ctx, id)
		if err != nil {
			return httpError(err)
		}
	}
	if r == nil {
		return echo.NewHTTPError(http.StatusNotFound, "report not found")
	}
	// Verify ownership through the associated topic
	t, err := h.topicRepo.FindByID(ctx, r.TopicID)
	if err != nil || t == nil || t.SubmitterUserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
	}
	return c.JSON(http.StatusOK, reportResponse(r))
}

// reportResponse flattens the Report struct into the snake_case JSON the frontend expects.
func reportResponse(r *report.Report) map[string]interface{} {
	userRating := interface{}(nil)
	if r.UserRating > 0 {
		userRating = r.UserRating
	}
	return map[string]interface{}{
		"id":                 r.ID,
		"discussion_id":     r.DiscussionID,
		"topic_id":          r.TopicID,
		"summary":           r.Summary,
		"consensus_points":  r.OpinionMatrix.ConsensusPoints,
		"divergence_points": r.OpinionMatrix.DivergencePoints,
		"key_questions":     r.OpinionMatrix.KeyQuestions,
		"action_items":      r.OpinionMatrix.ActionItems,
		"blind_spots":       r.OpinionMatrix.BlindSpots,
		"recommended_agents": r.RecommendedAgents,
		"quality_score":     r.QualityScore,
		"user_rating":       userRating,
		"generated_at":      r.GeneratedAt,
	}
}

// Rate handles POST /reports/:id/rating.
// Accepts either a report ID or a topic ID for convenience.
func (h *ReportHandler) Rate(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	ctx := c.Request().Context()
	// Verify ownership before allowing rating
	r, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return httpError(err)
	}
	if r == nil {
		r, err = h.repo.FindByTopicID(ctx, id)
		if err != nil {
			return httpError(err)
		}
	}
	if r == nil {
		return echo.NewHTTPError(http.StatusNotFound, "report not found")
	}
	t, err := h.topicRepo.FindByID(ctx, r.TopicID)
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
	if err := h.repo.UpdateUserRating(ctx, r.ID, body.Rating, body.Feedback); err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
