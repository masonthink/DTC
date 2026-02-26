// Package agent handles digital avatar (分身) CRUD and anonymous ID management.
package agent

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AgentType categorizes a digital avatar.
type AgentType string

const (
	AgentTypeProfessional AgentType = "professional"
	AgentTypeEntrepreneur AgentType = "entrepreneur"
	AgentTypeInvestor     AgentType = "investor"
	AgentTypeGeneralist   AgentType = "generalist"
)

// ThinkingStyle captures the agent's cognitive profile.
type ThinkingStyle struct {
	Analytical    float64 `json:"analytical"`    // 0-1
	Creative      float64 `json:"creative"`      // 0-1
	Critical      float64 `json:"critical"`      // 0-1
	Collaborative float64 `json:"collaborative"` // 0-1
	Questioning   float64 `json:"questioning"`   // 0-1 （用于角色分配：提问者）
}

// Questionnaire holds structured questionnaire responses.
type Questionnaire struct {
	// 基本背景
	PrimaryIndustry   string   `json:"primary_industry"`
	YearsExperience   int      `json:"years_experience"`
	CurrentRole       string   `json:"current_role"`
	Expertise         []string `json:"expertise"`

	// 思维方式（1-5 量表）
	ProblemApproach   string  `json:"problem_approach"`   // systematic/intuitive/experimental
	DecisionStyle     string  `json:"decision_style"`     // data-driven/experience-driven/consensus
	RiskTolerance     int     `json:"risk_tolerance"`     // 1-5
	InnovationFocus   int     `json:"innovation_focus"`   // 1-5

	// 讨论偏好
	PreferredRole     string  `json:"preferred_role"`     // critic/advocate/explorer/questioner
	DiscussionStrength string `json:"discussion_strength"` // 自由描述

	// 额外自由文本（用于生成 embedding）
	Bio               string `json:"bio"`
	AdditionalContext string `json:"additional_context"`
}

// Agent is the core domain object for a digital avatar.
type Agent struct {
	ID              string
	UserID          string
	AgentType       AgentType
	DisplayName     string
	Questionnaire   Questionnaire
	Industries      []string
	Skills          []string
	ThinkingStyle   ThinkingStyle
	ExperienceYears int
	AnonID          string  // AGT-XXXXXXXX format
	QdrantPointID   string
	QualityScore    float64
	DiscussionCount int
	ConnectionCount int
	IsActive        bool
	LastActiveAt    time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateRequest is the input for creating a new agent.
type CreateRequest struct {
	UserID        string
	AgentType     AgentType
	DisplayName   string
	Questionnaire Questionnaire
}

// UpdateRequest allows partial updates to an agent profile.
type UpdateRequest struct {
	DisplayName   *string
	AgentType     *AgentType
	Questionnaire *Questionnaire
}

// EmbeddingText returns a plain-text representation of this agent for vector embedding.
func (a *Agent) EmbeddingText() string {
	parts := make([]string, 0, 4)
	if a.Questionnaire.Bio != "" {
		parts = append(parts, a.Questionnaire.Bio)
	}
	if summary := a.BackgroundSummary(); summary != "" {
		parts = append(parts, summary)
	}
	if a.Questionnaire.DiscussionStrength != "" {
		parts = append(parts, a.Questionnaire.DiscussionStrength)
	}
	if a.Questionnaire.AdditionalContext != "" {
		parts = append(parts, a.Questionnaire.AdditionalContext)
	}
	return strings.Join(parts, "\n")
}

// BackgroundSummary generates a human-readable background string for LLM prompts.
func (a *Agent) BackgroundSummary() string {
	parts := []string{}
	if a.ExperienceYears > 0 {
		parts = append(parts, fmt.Sprintf("%d年行业经验", a.ExperienceYears))
	}
	if len(a.Industries) > 0 {
		parts = append(parts, fmt.Sprintf("领域：%s", strings.Join(a.Industries, "、")))
	}
	if len(a.Skills) > 0 && len(a.Skills) <= 3 {
		parts = append(parts, fmt.Sprintf("专长：%s", strings.Join(a.Skills, "、")))
	}
	if a.Questionnaire.DecisionStyle != "" {
		styleMap := map[string]string{
			"data-driven":       "数据驱动决策风格",
			"experience-driven": "经验主导决策风格",
			"consensus":         "共识驱动决策风格",
		}
		if label, ok := styleMap[a.Questionnaire.DecisionStyle]; ok {
			parts = append(parts, label)
		}
	}
	return strings.Join(parts, "，")
}

// Repository abstracts data access for agents.
type Repository interface {
	Create(ctx context.Context, agent *Agent) error
	FindByID(ctx context.Context, id string) (*Agent, error)
	FindByUserID(ctx context.Context, userID string) ([]*Agent, error)
	FindByAnonID(ctx context.Context, anonID string) (*Agent, error)
	Update(ctx context.Context, id string, req UpdateRequest) (*Agent, error)
	UpdateEmbedding(ctx context.Context, id, qdrantPointID string) error
	UpdateQualityScore(ctx context.Context, id string, score float64) error
	IncrementDiscussionCount(ctx context.Context, id string) error
	SetActive(ctx context.Context, id string, active bool) error
	FindActiveAgents(ctx context.Context, limit, offset int) ([]*Agent, error)
	FindAgentsForEmbedding(ctx context.Context, limit int) ([]*Agent, error)
}

// Service handles agent business logic.
type Service struct {
	repo   Repository
	logger *zap.Logger
}

// NewService constructs an agent Service.
func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// Create creates a new digital avatar.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Agent, error) {
	if err := validateCreate(req); err != nil {
		return nil, err
	}

	agent := &Agent{
		ID:              uuid.NewString(),
		UserID:          req.UserID,
		AgentType:       req.AgentType,
		DisplayName:     req.DisplayName,
		Questionnaire:   req.Questionnaire,
		AnonID:          generateAnonID(),
		IsActive:        true,
		ExperienceYears: req.Questionnaire.YearsExperience,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Derive structured fields from questionnaire
	agent.Industries = deriveIndustries(req.Questionnaire)
	agent.Skills = deriveSkills(req.Questionnaire)
	agent.ThinkingStyle = deriveThinkingStyle(req.Questionnaire)

	if err := s.repo.Create(ctx, agent); err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	s.logger.Info("agent created",
		zap.String("agent_id", agent.ID),
		zap.String("user_id", agent.UserID),
		zap.String("anon_id", agent.AnonID),
	)
	return agent, nil
}

// GetByID retrieves an agent by ID, checking ownership.
func (s *Service) GetByID(ctx context.Context, id, requestingUserID string) (*Agent, error) {
	agent, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found")
	}
	// Only owner can see full agent details
	if requestingUserID != "" && agent.UserID != requestingUserID {
		return nil, fmt.Errorf("access denied")
	}
	return agent, nil
}

// ListByUser returns all agents for a user.
func (s *Service) ListByUser(ctx context.Context, userID string) ([]*Agent, error) {
	return s.repo.FindByUserID(ctx, userID)
}

// SaveEmbeddingID persists the Qdrant point ID after a successful embedding upsert.
func (s *Service) SaveEmbeddingID(ctx context.Context, agentID, pointID string) error {
	return s.repo.UpdateEmbedding(ctx, agentID, pointID)
}

// Update partially updates an agent profile and marks embedding as stale.
func (s *Service) Update(ctx context.Context, id, userID string, req UpdateRequest) (*Agent, error) {
	agent, err := s.repo.FindByID(ctx, id)
	if err != nil || agent == nil {
		return nil, fmt.Errorf("agent not found")
	}
	if agent.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}
	return s.repo.Update(ctx, id, req)
}

// generateAnonID creates a unique anonymous ID in the format AGT-XXXXXXXX.
func generateAnonID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("AGT-%X", b)
}

func validateCreate(req CreateRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user_id required")
	}
	if req.AgentType == "" {
		req.AgentType = AgentTypeProfessional
	}
	if req.Questionnaire.PrimaryIndustry == "" {
		return fmt.Errorf("primary_industry required")
	}
	return nil
}

func deriveIndustries(q Questionnaire) []string {
	industries := []string{q.PrimaryIndustry}
	return industries
}

func deriveSkills(q Questionnaire) []string {
	return q.Expertise
}

func deriveThinkingStyle(q Questionnaire) ThinkingStyle {
	ts := ThinkingStyle{}
	// Map questionnaire answers to numeric scores
	switch q.ProblemApproach {
	case "systematic":
		ts.Analytical = 0.8
		ts.Creative = 0.3
	case "intuitive":
		ts.Analytical = 0.4
		ts.Creative = 0.7
	case "experimental":
		ts.Analytical = 0.5
		ts.Creative = 0.8
	}

	switch q.DecisionStyle {
	case "data-driven":
		ts.Analytical += 0.2
	case "consensus":
		ts.Collaborative = 0.8
	}

	switch q.PreferredRole {
	case "critic":
		ts.Critical = 0.9
		ts.Questioning = 0.7
	case "advocate":
		ts.Analytical = 0.7
	case "questioner":
		ts.Questioning = 0.9
	case "explorer":
		ts.Creative = 0.8
	}

	// Clamp all values to 0-1
	ts.Analytical = clamp(ts.Analytical, 0, 1)
	ts.Creative = clamp(ts.Creative, 0, 1)
	ts.Critical = clamp(ts.Critical, 0, 1)
	ts.Collaborative = clamp(ts.Collaborative, 0, 1)
	ts.Questioning = clamp(ts.Questioning, 0, 1)
	return ts
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
