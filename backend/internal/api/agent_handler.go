package api

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/digital-twin-community/backend/internal/agent"
	"github.com/digital-twin-community/backend/internal/embedding"
	apimiddleware "github.com/digital-twin-community/backend/internal/middleware"
)

// AgentHandler handles /agents endpoints.
type AgentHandler struct {
	svc          *agent.Service
	embeddingSvc *embedding.Service
	logger       *zap.Logger
}

// NewAgentHandler constructs an AgentHandler.
func NewAgentHandler(svc *agent.Service, embeddingSvc *embedding.Service, logger *zap.Logger) *AgentHandler {
	return &AgentHandler{svc: svc, embeddingSvc: embeddingSvc, logger: logger}
}

// Create handles POST /agents.
func (h *AgentHandler) Create(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	var req agent.CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	req.UserID = userID
	a, err := h.svc.Create(c.Request().Context(), req)
	if err != nil {
		return httpError(err)
	}
	h.embedAgentAsync(a)
	return c.JSON(http.StatusCreated, a)
}

// List handles GET /agents.
func (h *AgentHandler) List(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	agents, err := h.svc.ListByUser(c.Request().Context(), userID)
	if err != nil {
		return httpError(err)
	}
	return c.JSON(http.StatusOK, agents)
}

// Get handles GET /agents/:id.
func (h *AgentHandler) Get(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	a, err := h.svc.GetByID(c.Request().Context(), id, userID)
	if err != nil {
		return httpError(err)
	}
	if a == nil {
		return echo.NewHTTPError(http.StatusNotFound, "agent not found")
	}
	return c.JSON(http.StatusOK, a)
}

// Update handles PUT /agents/:id.
func (h *AgentHandler) Update(c echo.Context) error {
	userID := apimiddleware.UserIDFromContext(c)
	id := c.Param("id")
	var req agent.UpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	a, err := h.svc.Update(c.Request().Context(), id, userID, req)
	if err != nil {
		return httpError(err)
	}
	h.embedAgentAsync(a)
	return c.JSON(http.StatusOK, a)
}

// embedAgentAsync triggers vector embedding in a background goroutine.
// The HTTP response is not blocked; errors are logged.
func (h *AgentHandler) embedAgentAsync(a *agent.Agent) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		text := a.EmbeddingText()
		if text == "" {
			return
		}

		lastActive := a.LastActiveAt.Unix()
		if a.LastActiveAt.IsZero() {
			lastActive = time.Now().Unix()
		}

		payload := embedding.AgentPayload{
			AgentID:        a.ID,
			UserID:         a.UserID,
			AnonID:         a.AnonID,
			Industries:     a.Industries,
			AgentType:      string(a.AgentType),
			QualityScore:   a.QualityScore,
			LastActiveUnix: lastActive,
		}

		pointID, err := h.embeddingSvc.EmbedAndUpsertAgent(ctx, a.ID, text, payload)
		if err != nil {
			h.logger.Error("agent embedding failed",
				zap.String("agent_id", a.ID),
				zap.Error(err),
			)
			return
		}

		if err := h.svc.SaveEmbeddingID(ctx, a.ID, pointID); err != nil {
			h.logger.Error("save agent embedding id failed",
				zap.String("agent_id", a.ID),
				zap.Error(err),
			)
		}
	}()
}
