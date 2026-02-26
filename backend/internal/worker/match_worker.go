// Package worker implements Asynq task handlers for the discussion lifecycle.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/digital-twin-community/backend/internal/agent"
	"github.com/digital-twin-community/backend/internal/discussion"
	"github.com/digital-twin-community/backend/internal/embedding"
	"github.com/digital-twin-community/backend/internal/llm"
	"github.com/digital-twin-community/backend/internal/matching"
	"github.com/digital-twin-community/backend/internal/scheduler"
	"github.com/digital-twin-community/backend/internal/topic"
)

// MatchTopicWorker handles the "topic:match" task.
// It runs Phase 1 (ANN recall) + Phase 2 (MMR) and creates a Discussion.
type MatchTopicWorker struct {
	topicRepo     topic.Repository
	agentRepo     agent.Repository
	embeddingSvc  *embedding.Service
	llmGateway   *llm.Gateway
	matcher       *matching.Matcher
	discussionRepo discussion.Repository
	sched         *scheduler.Scheduler
	logger        *zap.Logger
}

// NewMatchTopicWorker constructs a MatchTopicWorker.
func NewMatchTopicWorker(
	topicRepo topic.Repository,
	agentRepo agent.Repository,
	embeddingSvc *embedding.Service,
	llmGateway *llm.Gateway,
	matcher *matching.Matcher,
	discussionRepo discussion.Repository,
	sched *scheduler.Scheduler,
	logger *zap.Logger,
) *MatchTopicWorker {
	return &MatchTopicWorker{
		topicRepo:      topicRepo,
		agentRepo:      agentRepo,
		embeddingSvc:   embeddingSvc,
		llmGateway:    llmGateway,
		matcher:        matcher,
		discussionRepo: discussionRepo,
		sched:          sched,
		logger:         logger,
	}
}

// Handle processes one "topic:match" task.
func (w *MatchTopicWorker) Handle(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		TopicID string `json:"topic_id"`
	}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	w.logger.Info("match_worker: starting", zap.String("topic_id", payload.TopicID))

	// ── 1. Load topic ───────────────────────────────────────────────────────
	t, err := w.topicRepo.FindByID(ctx, payload.TopicID)
	if err != nil {
		return fmt.Errorf("load topic: %w", err)
	}
	if t == nil {
		return fmt.Errorf("topic %s not found", payload.TopicID)
	}

	topicText := t.TextForEmbedding()

	// ── 2. Phase 1: topic embedding + ANN recall ────────────────────────────
	topicVec, embedErr := w.llmGateway.Embed(ctx, topicText, "matching")
	if embedErr != nil {
		w.logger.Warn("topic embedding failed; using zero vector for matching",
			zap.String("topic_id", t.ID),
			zap.Error(embedErr),
		)
	}
	// Ensure vector has correct dimension (matcher requires non-empty)
	if len(topicVec) == 0 {
		topicVec = make([]float32, embedding.VectorDimension)
	}

	var candidates []matching.Candidate

	annResults, _ := w.embeddingSvc.SearchSimilarAgents(ctx, topicVec, 100, 0.5, 30)
	for _, r := range annResults {
		candidates = append(candidates, matching.Candidate{
			AgentID:  r.AgentID,
			AnonID:   r.AnonID,
			// Embeddings not available from ANN payload; diversity score will be 0
			Industries:   r.Payload.Industries,
			QualityScore: r.Payload.QualityScore,
			LastActiveAt: time.Unix(r.Payload.LastActiveUnix, 0),
		})
	}

	// ── 3. Fallback: load active agents from DB when ANN pool is insufficient ─
	agentMap := make(map[string]*agent.Agent)
	if len(candidates) < matching.DefaultN {
		activeAgents, err := w.agentRepo.FindActiveAgents(ctx, 200, 0)
		if err != nil {
			return fmt.Errorf("find active agents: %w", err)
		}
		for _, a := range activeAgents {
			agentMap[a.ID] = a
		}
		candidates = agentsToCandidates(activeAgents)
		w.logger.Info("match_worker: using DB fallback candidates",
			zap.Int("count", len(candidates)),
		)
	}

	// ── 4. Phase 2: MMR matching ─────────────────────────────────────────────
	result, err := w.matcher.Match(ctx, topicVec, candidates, matching.DefaultN)
	if err != nil {
		return fmt.Errorf("matching: %w", err)
	}

	// ── 5. Build discussion participants ─────────────────────────────────────
	participants := make([]discussion.Participant, 0, len(result.Participants))
	for _, ra := range result.Participants {
		p := discussion.Participant{
			AgentID:       ra.Candidate.AgentID,
			AnonID:        ra.Candidate.AnonID,
			Role:          discussion.Role(ra.Role),
			Industries:    ra.Candidate.Industries,
			Skills:        ra.Candidate.Skills,
			ThinkingStyle: ra.Candidate.ThinkingStyle,
		}
		if a, ok := agentMap[ra.Candidate.AgentID]; ok {
			p.Background = a.BackgroundSummary()
		}
		participants = append(participants, p)
	}

	// ── 6. Persist discussion ────────────────────────────────────────────────
	disc := &discussion.Discussion{
		ID:           uuid.NewString(),
		TopicID:      t.ID,
		TopicText:    topicText,
		Status:       discussion.StatusPendingMatching,
		Participants: participants,
	}
	if err := w.discussionRepo.Create(ctx, disc); err != nil {
		return fmt.Errorf("create discussion: %w", err)
	}

	// ── 7. Save anon ID mappings (non-fatal) ──────────────────────────────────
	if err := w.discussionRepo.SaveAnonMappings(ctx, disc.ID, participants); err != nil {
		w.logger.Warn("match_worker: save anon mappings failed",
			zap.String("discussion_id", disc.ID),
			zap.Error(err),
		)
	}

	// ── 8. Mark topic as matched ──────────────────────────────────────────────
	now := time.Now()
	if err := w.topicRepo.MarkMatched(ctx, t.ID, now); err != nil {
		return fmt.Errorf("mark topic matched: %w", err)
	}

	// ── 9. Enqueue all 4 discussion rounds at their scheduled times ───────────
	schedule := scheduler.DiscussionSchedule(now)
	for round := 1; round <= 4; round++ {
		if err := w.sched.EnqueueDiscussionRound(ctx, disc.ID, round, schedule[round]); err != nil {
			w.logger.Error("match_worker: enqueue round failed",
				zap.Int("round", round),
				zap.Error(err),
			)
		}
	}

	w.logger.Info("match_worker: done",
		zap.String("discussion_id", disc.ID),
		zap.String("topic_id", t.ID),
		zap.Int("participants", len(participants)),
	)
	return nil
}

// agentsToCandidates converts active Agent records to matching.Candidate slice.
func agentsToCandidates(agents []*agent.Agent) []matching.Candidate {
	out := make([]matching.Candidate, 0, len(agents))
	for _, a := range agents {
		out = append(out, matching.Candidate{
			AgentID:   a.ID,
			AnonID:    a.AnonID,
			Industries: a.Industries,
			Skills:     a.Skills,
			ThinkingStyle: map[string]float64{
				"analytical":    a.ThinkingStyle.Analytical,
				"creative":      a.ThinkingStyle.Creative,
				"critical":      a.ThinkingStyle.Critical,
				"collaborative": a.ThinkingStyle.Collaborative,
			},
			ExperienceYears: a.ExperienceYears,
			QualityScore:    a.QualityScore,
			LastActiveAt:    a.LastActiveAt,
			QuestionnaireMeta: map[string]interface{}{
				"questioning_ability": a.ThinkingStyle.Questioning,
			},
		})
	}
	return out
}
