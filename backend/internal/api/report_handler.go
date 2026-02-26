package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/digital-twin-community/backend/internal/report"
)

// ReportHandler handles /reports endpoints.
type ReportHandler struct {
	repo report.Repository
}

// NewReportHandler constructs a ReportHandler.
func NewReportHandler(repo report.Repository) *ReportHandler {
	return &ReportHandler{repo: repo}
}

// Get handles GET /reports/:id.
func (h *ReportHandler) Get(c echo.Context) error {
	id := c.Param("id")
	r, err := h.repo.FindByID(c.Request().Context(), id)
	if err != nil {
		return httpError(err)
	}
	if r == nil {
		return echo.NewHTTPError(http.StatusNotFound, "report not found")
	}
	return c.JSON(http.StatusOK, r)
}

// Rate handles POST /reports/:id/rating.
func (h *ReportHandler) Rate(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		Rating   int    `json:"rating"`
		Feedback string `json:"feedback,omitempty"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if body.Rating < 1 || body.Rating > 5 {
		return echo.NewHTTPError(http.StatusBadRequest, "rating must be between 1 and 5")
	}
	if err := h.repo.UpdateUserRating(c.Request().Context(), id, body.Rating, body.Feedback); err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
