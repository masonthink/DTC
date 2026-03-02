package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/digital-twin-community/backend/internal/discussion"
	"github.com/digital-twin-community/backend/internal/report"
	"github.com/digital-twin-community/backend/internal/topic"
)

// ReportGenerateWorker handles the "report:generate" task.
type ReportGenerateWorker struct {
	discussionRepo discussion.Repository
	topicRepo      topic.Repository
	reportRepo     report.Repository
	generator      *report.Generator
	pool           *pgxpool.Pool
	logger         *zap.Logger
}

// NewReportGenerateWorker constructs a ReportGenerateWorker.
func NewReportGenerateWorker(
	discussionRepo discussion.Repository,
	topicRepo topic.Repository,
	reportRepo report.Repository,
	generator *report.Generator,
	pool *pgxpool.Pool,
	logger *zap.Logger,
) *ReportGenerateWorker {
	return &ReportGenerateWorker{
		discussionRepo: discussionRepo,
		topicRepo:      topicRepo,
		reportRepo:     reportRepo,
		generator:      generator,
		pool:           pool,
		logger:         logger,
	}
}

// Handle processes one "report:generate" task.
func (w *ReportGenerateWorker) Handle(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		DiscussionID string `json:"discussion_id"`
		TopicID      string `json:"topic_id"`
	}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	w.logger.Info("report_worker: starting",
		zap.String("discussion_id", payload.DiscussionID),
	)

	// ── 1. Load discussion ───────────────────────────────────────────────────
	disc, err := w.discussionRepo.FindByID(ctx, payload.DiscussionID)
	if err != nil || disc == nil {
		return fmt.Errorf("load discussion %s: %w", payload.DiscussionID, err)
	}

	// ── 2. Load all messages ─────────────────────────────────────────────────
	dbMsgs, err := w.discussionRepo.FindMessages(ctx, payload.DiscussionID)
	if err != nil {
		return fmt.Errorf("load messages: %w", err)
	}
	if len(dbMsgs) == 0 {
		return fmt.Errorf("no messages found for discussion %s", payload.DiscussionID)
	}

	// ── 3. Load topic ────────────────────────────────────────────────────────
	t, err := w.topicRepo.FindByID(ctx, payload.TopicID)
	if err != nil || t == nil {
		return fmt.Errorf("load topic %s: %w", payload.TopicID, err)
	}

	// ── 4. Build participant AnonID lookup ────────────────────────────────────
	anonByAgent := make(map[string]string, len(disc.Participants))
	for _, p := range disc.Participants {
		anonByAgent[p.AgentID] = p.AnonID
	}

	// ── 5. Convert messages to report.DiscussionMessage ──────────────────────
	reportMsgs := make([]report.DiscussionMessage, 0, len(dbMsgs))
	for _, m := range dbMsgs {
		reportMsgs = append(reportMsgs, report.DiscussionMessage{
			DiscussionID: payload.DiscussionID,
			AgentID:      m.AgentID,
			AnonID:       anonByAgent[m.AgentID],
			RoundNumber:  m.RoundNum,
			Role:         string(m.Role),
			Content:      m.Content,
			KeyPoint:     m.KeyPoint,
			Confidence:   m.Confidence,
			ModelUsed:    m.ModelUsed,
		})
	}

	// ── 6. Build report topic ─────────────────────────────────────────────────
	reportTopic := report.Topic{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		TopicType:   string(t.TopicType),
		Tags:        t.Tags,
	}

	// ── 7. Build connection candidates from participants ───────────────────────
	candidates := make([]report.ConnectionCandidate, 0, len(disc.Participants))
	for _, p := range disc.Participants {
		candidates = append(candidates, report.ConnectionCandidate{
			AgentID: p.AgentID,
			AnonID:  p.AnonID,
		})
	}

	// ── 8. Generate report ───────────────────────────────────────────────────
	rep, err := w.generator.Generate(ctx, reportTopic, reportMsgs, candidates)
	if err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	// ── 9+10. Save report + update statuses atomically in a transaction ──────
	if err := w.saveReportAndComplete(ctx, rep, payload.DiscussionID, payload.TopicID); err != nil {
		return fmt.Errorf("report_worker: save and complete: %w", err)
	}

	w.logger.Info("report_worker: done",
		zap.String("discussion_id", payload.DiscussionID),
		zap.String("report_id", rep.ID),
		zap.Float64("quality_score", rep.QualityScore),
	)
	return nil
}

// saveReportAndComplete saves the report and updates discussion/topic statuses
// in a single database transaction, ensuring atomicity.
func (w *ReportGenerateWorker) saveReportAndComplete(
	ctx context.Context,
	rep *report.Report,
	discussionID, topicID string,
) error {
	if rep.ID == "" {
		rep.ID = uuid.NewString()
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Save report (idempotent via ON CONFLICT)
	consensusJSON, err := json.Marshal(rep.OpinionMatrix.ConsensusPoints)
	if err != nil {
		return fmt.Errorf("marshal consensus_points: %w", err)
	}
	divergenceJSON, err := json.Marshal(rep.OpinionMatrix.DivergencePoints)
	if err != nil {
		return fmt.Errorf("marshal divergence_points: %w", err)
	}
	questionsJSON, err := json.Marshal(rep.OpinionMatrix.KeyQuestions)
	if err != nil {
		return fmt.Errorf("marshal key_questions: %w", err)
	}
	actionsJSON, err := json.Marshal(rep.OpinionMatrix.ActionItems)
	if err != nil {
		return fmt.Errorf("marshal action_items: %w", err)
	}
	blindSpotsJSON, err := json.Marshal(rep.OpinionMatrix.BlindSpots)
	if err != nil {
		return fmt.Errorf("marshal blind_spots: %w", err)
	}
	recommendedJSON, err := json.Marshal(rep.RecommendedAgents)
	if err != nil {
		return fmt.Errorf("marshal recommended_agents: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO reports
			(id, discussion_id, topic_id, summary,
			 consensus_points, divergence_points, key_questions, action_items, blind_spots,
			 recommended_agents, quality_score, model_used, total_tokens,
			 generation_attempts, generated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO NOTHING`,
		rep.ID, rep.DiscussionID, rep.TopicID, rep.Summary,
		consensusJSON, divergenceJSON, questionsJSON, actionsJSON, blindSpotsJSON,
		recommendedJSON, rep.QualityScore, rep.ModelUsed, rep.TotalTokens,
		rep.GenerationAttempts, rep.GeneratedAt,
	)
	if err != nil {
		return fmt.Errorf("insert report: %w", err)
	}

	// Update discussion status to COMPLETED
	discStatus := strings.ToLower(string(discussion.StatusCompleted))
	_, err = tx.Exec(ctx, `
		UPDATE discussions SET
			status = $2::discussion_status,
			updated_at = NOW()
		WHERE id = $1`,
		discussionID, discStatus,
	)
	if err != nil {
		return fmt.Errorf("update discussion status: %w", err)
	}

	// Update topic status to completed
	_, err = tx.Exec(ctx, `
		UPDATE topics SET
			status = 'completed'::topic_status,
			completed_at = NOW(), updated_at = NOW()
		WHERE id = $1`,
		topicID,
	)
	if err != nil {
		return fmt.Errorf("update topic status: %w", err)
	}

	return tx.Commit(ctx)
}
