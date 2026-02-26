package agent_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/digital-twin-community/backend/internal/agent"
)

// ── mock repository ───────────────────────────────────────────────────────────

type mockRepo struct {
	agents map[string]*agent.Agent
}

func newMockRepo() *mockRepo {
	return &mockRepo{agents: make(map[string]*agent.Agent)}
}

func (m *mockRepo) Create(_ context.Context, a *agent.Agent) error {
	m.agents[a.ID] = a
	return nil
}
func (m *mockRepo) FindByID(_ context.Context, id string) (*agent.Agent, error) {
	a, ok := m.agents[id]
	if !ok {
		return nil, nil
	}
	return a, nil
}
func (m *mockRepo) FindByUserID(_ context.Context, userID string) ([]*agent.Agent, error) {
	var out []*agent.Agent
	for _, a := range m.agents {
		if a.UserID == userID {
			out = append(out, a)
		}
	}
	return out, nil
}
func (m *mockRepo) FindByAnonID(_ context.Context, anonID string) (*agent.Agent, error) {
	for _, a := range m.agents {
		if a.AnonID == anonID {
			return a, nil
		}
	}
	return nil, nil
}
func (m *mockRepo) Update(_ context.Context, id string, req agent.UpdateRequest) (*agent.Agent, error) {
	a, ok := m.agents[id]
	if !ok {
		return nil, errors.New("not found")
	}
	if req.DisplayName != nil {
		a.DisplayName = *req.DisplayName
	}
	return a, nil
}
func (m *mockRepo) UpdateEmbedding(_ context.Context, id, pointID string) error {
	if a, ok := m.agents[id]; ok {
		a.QdrantPointID = pointID
	}
	return nil
}
func (m *mockRepo) UpdateQualityScore(_ context.Context, id string, score float64) error {
	if a, ok := m.agents[id]; ok {
		a.QualityScore = score
	}
	return nil
}
func (m *mockRepo) IncrementDiscussionCount(_ context.Context, id string) error {
	if a, ok := m.agents[id]; ok {
		a.DiscussionCount++
	}
	return nil
}
func (m *mockRepo) SetActive(_ context.Context, id string, active bool) error {
	if a, ok := m.agents[id]; ok {
		a.IsActive = active
	}
	return nil
}
func (m *mockRepo) FindActiveAgents(_ context.Context, limit, offset int) ([]*agent.Agent, error) {
	var out []*agent.Agent
	for _, a := range m.agents {
		if a.IsActive {
			out = append(out, a)
		}
	}
	return out, nil
}
func (m *mockRepo) FindAgentsForEmbedding(_ context.Context, limit int) ([]*agent.Agent, error) {
	return nil, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func validCreateReq() agent.CreateRequest {
	return agent.CreateRequest{
		UserID:    "user-123",
		AgentType: agent.AgentTypeProfessional,
		Questionnaire: agent.Questionnaire{
			PrimaryIndustry: "互联网",
			YearsExperience: 5,
			Bio:             "资深工程师，专注分布式系统",
		},
	}
}

// ── EmbeddingText tests ───────────────────────────────────────────────────────

func TestEmbeddingText_WithAllFields(t *testing.T) {
	a := &agent.Agent{
		ExperienceYears: 5,
		Industries:      []string{"互联网"},
		Questionnaire: agent.Questionnaire{
			Bio:               "分布式系统专家",
			DiscussionStrength: "擅长系统设计讨论",
			AdditionalContext: "曾主导多个大规模项目",
			DecisionStyle:     "data-driven",
		},
	}
	text := a.EmbeddingText()
	if text == "" {
		t.Error("expected non-empty embedding text")
	}
	for _, want := range []string{"分布式系统专家", "擅长系统设计讨论", "曾主导多个大规模项目"} {
		if !containsStr(text, want) {
			t.Errorf("embedding text missing %q", want)
		}
	}
}

func TestEmbeddingText_EmptyQuestionnaire(t *testing.T) {
	a := &agent.Agent{}
	text := a.EmbeddingText()
	if text != "" {
		t.Errorf("expected empty text for empty agent, got %q", text)
	}
}

func TestEmbeddingText_OnlyBio(t *testing.T) {
	a := &agent.Agent{
		Questionnaire: agent.Questionnaire{Bio: "只有简介"},
	}
	text := a.EmbeddingText()
	if text != "只有简介" {
		t.Errorf("got %q, want %q", text, "只有简介")
	}
}

// ── BackgroundSummary tests ───────────────────────────────────────────────────

func TestBackgroundSummary_Full(t *testing.T) {
	a := &agent.Agent{
		ExperienceYears: 8,
		Industries:      []string{"金融"},
		Skills:          []string{"风控", "量化"},
		Questionnaire:   agent.Questionnaire{DecisionStyle: "data-driven"},
	}
	summary := a.BackgroundSummary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	if !containsStr(summary, "8年") {
		t.Errorf("summary missing experience years: %q", summary)
	}
}

func TestBackgroundSummary_Empty(t *testing.T) {
	a := &agent.Agent{}
	if summary := a.BackgroundSummary(); summary != "" {
		t.Errorf("expected empty summary for empty agent, got %q", summary)
	}
}

// ── Service.Create tests ──────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	svc := agent.NewService(newMockRepo(), zap.NewNop())
	a, err := svc.Create(context.Background(), validCreateReq())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if a.ID == "" {
		t.Error("expected non-empty agent ID")
	}
	if a.AnonID == "" {
		t.Error("expected non-empty anon ID")
	}
	if a.UserID != "user-123" {
		t.Errorf("got UserID %q, want user-123", a.UserID)
	}
	if !a.IsActive {
		t.Error("new agent should be active")
	}
}

func TestCreate_AnonIDFormat(t *testing.T) {
	svc := agent.NewService(newMockRepo(), zap.NewNop())
	a, err := svc.Create(context.Background(), validCreateReq())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(a.AnonID) < 4 || a.AnonID[:4] != "AGT-" {
		t.Errorf("expected AnonID to start with AGT-, got %q", a.AnonID)
	}
}

func TestCreate_AnonIDUnique(t *testing.T) {
	svc := agent.NewService(newMockRepo(), zap.NewNop())
	ctx := context.Background()
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		req := validCreateReq()
		a, err := svc.Create(ctx, req)
		if err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
		if seen[a.AnonID] {
			t.Errorf("duplicate AnonID %q", a.AnonID)
		}
		seen[a.AnonID] = true
	}
}

func TestCreate_MissingUserID(t *testing.T) {
	svc := agent.NewService(newMockRepo(), zap.NewNop())
	req := validCreateReq()
	req.UserID = ""
	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Error("expected error for missing UserID")
	}
}

func TestCreate_MissingIndustry(t *testing.T) {
	svc := agent.NewService(newMockRepo(), zap.NewNop())
	req := validCreateReq()
	req.Questionnaire.PrimaryIndustry = ""
	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Error("expected error for missing PrimaryIndustry")
	}
}

func TestCreate_DerivedFields(t *testing.T) {
	svc := agent.NewService(newMockRepo(), zap.NewNop())
	req := validCreateReq()
	req.Questionnaire.ProblemApproach = "systematic"
	req.Questionnaire.Expertise = []string{"Go", "Kubernetes"}
	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(a.Skills) == 0 {
		t.Error("expected skills to be derived from expertise")
	}
	if a.ThinkingStyle.Analytical == 0 {
		t.Error("expected analytical score > 0 for systematic approach")
	}
}

// ── Service.Update tests ──────────────────────────────────────────────────────

func TestUpdate_Success(t *testing.T) {
	svc := agent.NewService(newMockRepo(), zap.NewNop())
	ctx := context.Background()

	a, _ := svc.Create(ctx, validCreateReq())

	newName := "新名字"
	updated, err := svc.Update(ctx, a.ID, a.UserID, agent.UpdateRequest{
		DisplayName: &newName,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.DisplayName != newName {
		t.Errorf("got DisplayName %q, want %q", updated.DisplayName, newName)
	}
}

func TestUpdate_WrongOwner(t *testing.T) {
	svc := agent.NewService(newMockRepo(), zap.NewNop())
	ctx := context.Background()

	a, _ := svc.Create(ctx, validCreateReq())

	newName := "恶意修改"
	_, err := svc.Update(ctx, a.ID, "other-user", agent.UpdateRequest{
		DisplayName: &newName,
	})
	if err == nil {
		t.Error("expected error when non-owner tries to update")
	}
}

// ── SaveEmbeddingID test ──────────────────────────────────────────────────────

func TestSaveEmbeddingID(t *testing.T) {
	repo := newMockRepo()
	svc := agent.NewService(repo, zap.NewNop())
	ctx := context.Background()

	a, _ := svc.Create(ctx, validCreateReq())

	pointID := "qdrant-point-uuid"
	if err := svc.SaveEmbeddingID(ctx, a.ID, pointID); err != nil {
		t.Fatalf("SaveEmbeddingID: %v", err)
	}
	if repo.agents[a.ID].QdrantPointID != pointID {
		t.Errorf("QdrantPointID not saved: got %q", repo.agents[a.ID].QdrantPointID)
	}
}

// ── LastActiveAt zero handling (used by agent_handler embedAgentAsync) ────────

func TestAgent_LastActiveAt_Zero(t *testing.T) {
	a := &agent.Agent{}
	if !a.LastActiveAt.IsZero() {
		t.Error("unset LastActiveAt should be zero")
	}
	// Callers should substitute time.Now().Unix() when LastActiveAt is zero
	if a.LastActiveAt.Unix() >= time.Now().Unix() {
		t.Error("zero time should be before now")
	}
}

// ── helper ────────────────────────────────────────────────────────────────────

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
