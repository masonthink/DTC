// Package topic manages Topic submission and state transitions.
package topic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TopicType classifies the discussion topic.
type TopicType string

const (
	TopicTypeBusinessIdea   TopicType = "business_idea"
	TopicTypeCareerDecision TopicType = "career_decision"
	TopicTypeTechChoice     TopicType = "tech_choice"
	TopicTypeProductDesign  TopicType = "product_design"
	TopicTypeInvestment     TopicType = "investment"
	TopicTypeOther          TopicType = "other"
)

// Status mirrors the DB enum.
type Status string

const (
	StatusPendingMatching  Status = "pending_matching"
	StatusMatching         Status = "matching"
	StatusMatched          Status = "matched"
	StatusDiscussionActive Status = "discussion_active"
	StatusReportGenerating Status = "report_generating"
	StatusCompleted        Status = "completed"
	StatusFailed           Status = "failed"
	StatusCancelled        Status = "cancelled"
)

// Topic is the core domain object.
type Topic struct {
	ID                  string     `json:"id"`
	SubmitterUserID     string     `json:"submitter_user_id"`
	SubmitterAgentID    string     `json:"submitter_agent_id"`
	TopicType           TopicType  `json:"topic_type"`
	Title               string     `json:"title"`
	Description         string     `json:"description"`
	Background          string     `json:"background"`
	Tags                []string   `json:"tags"`
	Status              Status     `json:"status"`
	SubmittedAt         time.Time  `json:"submitted_at"`
	MatchedAt           *time.Time `json:"matched_at,omitempty"`
	DiscussionStartedAt *time.Time `json:"discussion_started_at,omitempty"`
	ReportReadyAt       *time.Time `json:"report_ready_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	Notified1h          bool       `json:"notified_1h"`
	Notified12h         bool       `json:"notified_12h"`
	Notified48h         bool       `json:"notified_48h"`
}

// SubmitRequest is the input for submitting a new topic.
type SubmitRequest struct {
	SubmitterUserID  string    `json:"-"`
	SubmitterAgentID string    `json:"submitter_agent_id"`
	TopicType        TopicType `json:"topic_type"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Background       string    `json:"background,omitempty"`
	Tags             []string  `json:"tags"`
}

// Repository abstracts data access for topics.
type Repository interface {
	Create(ctx context.Context, topic *Topic) error
	FindByID(ctx context.Context, id string) (*Topic, error)
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*Topic, error)
	UpdateStatus(ctx context.Context, id string, status Status) error
	MarkMatched(ctx context.Context, id string, matchedAt time.Time) error
	MarkDiscussionStarted(ctx context.Context, id string) error
	MarkReportReady(ctx context.Context, id string) error
	MarkCompleted(ctx context.Context, id string) error
	SetNotified(ctx context.Context, id string, notificationType string) error
	FindPendingNotifications(ctx context.Context, notifType string, since time.Time) ([]*Topic, error)
	FindPendingMatching(ctx context.Context, limit int) ([]*Topic, error)
}

// Service manages topic business logic.
type Service struct {
	repo   Repository
	logger *zap.Logger
}

// NewService constructs a Topic Service.
func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// Submit creates a new topic and queues it for matching.
func (s *Service) Submit(ctx context.Context, req SubmitRequest) (*Topic, error) {
	if err := validateSubmit(req); err != nil {
		return nil, err
	}

	topic := &Topic{
		ID:               uuid.NewString(),
		SubmitterUserID:  req.SubmitterUserID,
		SubmitterAgentID: req.SubmitterAgentID,
		TopicType:        req.TopicType,
		Title:            req.Title,
		Description:      req.Description,
		Background:       req.Background,
		Tags:             req.Tags,
		Status:           StatusPendingMatching,
		SubmittedAt:      time.Now(),
	}

	if err := s.repo.Create(ctx, topic); err != nil {
		return nil, fmt.Errorf("create topic: %w", err)
	}

	s.logger.Info("topic submitted",
		zap.String("topic_id", topic.ID),
		zap.String("user_id", topic.SubmitterUserID),
		zap.String("type", string(topic.TopicType)),
	)
	return topic, nil
}

// GetByID retrieves a topic, verifying ownership for non-public fields.
func (s *Service) GetByID(ctx context.Context, id, requestingUserID string) (*Topic, error) {
	topic, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if topic == nil {
		return nil, fmt.Errorf("topic not found")
	}
	if topic.SubmitterUserID != requestingUserID {
		return nil, fmt.Errorf("access denied")
	}
	return topic, nil
}

// ListByUser returns paginated topics for a user.
func (s *Service) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*Topic, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.FindByUserID(ctx, userID, limit, offset)
}

// Cancel cancels a pending topic.
func (s *Service) Cancel(ctx context.Context, id, userID string) error {
	topic, err := s.repo.FindByID(ctx, id)
	if err != nil || topic == nil {
		return fmt.Errorf("topic not found")
	}
	if topic.SubmitterUserID != userID {
		return fmt.Errorf("access denied")
	}
	if topic.Status != StatusPendingMatching {
		return fmt.Errorf("cannot cancel topic in status %s", topic.Status)
	}
	return s.repo.UpdateStatus(ctx, id, StatusCancelled)
}

// TextForEmbedding returns the text to use for generating the topic embedding.
func (t *Topic) TextForEmbedding() string {
	parts := []string{t.Title, t.Description}
	if t.Background != "" {
		parts = append(parts, t.Background)
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "\n"
		}
		result += p
	}
	return result
}

func validateSubmit(req SubmitRequest) error {
	if req.SubmitterUserID == "" {
		return fmt.Errorf("submitter_user_id required")
	}
	if req.SubmitterAgentID == "" {
		return fmt.Errorf("submitter_agent_id required")
	}
	if req.Title == "" {
		return fmt.Errorf("title required")
	}
	if len(req.Title) > 500 {
		return fmt.Errorf("title too long (max 500 chars)")
	}
	if req.Description == "" {
		return fmt.Errorf("description required")
	}
	return nil
}
