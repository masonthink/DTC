package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
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
	logger         *zap.Logger
}

// NewReportGenerateWorker constructs a ReportGenerateWorker.
func NewReportGenerateWorker(
	discussionRepo discussion.Repository,
	topicRepo topic.Repository,
	reportRepo report.Repository,
	generator *report.Generator,
	logger *zap.Logger,
) *ReportGenerateWorker {
	return &ReportGenerateWorker{
		discussionRepo: discussionRepo,
		topicRepo:      topicRepo,
		reportRepo:     reportRepo,
		generator:      generator,
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
			// InsightCount, Complementarity, CollabSignal, ActivityScore left at 0;
			// the generator's scoreConnectionCandidates will compute from messages.
		})
	}

	// ── 8. Generate report ───────────────────────────────────────────────────
	rep, err := w.generator.Generate(ctx, reportTopic, reportMsgs, candidates)
	if err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	// ── 9. Save report ────────────────────────────────────────────────────────
	if err := w.reportRepo.Save(ctx, rep); err != nil {
		return fmt.Errorf("save report: %w", err)
	}

	// ── 10. Final status transitions ──────────────────────────────────────────
	if err := w.topicRepo.MarkCompleted(ctx, payload.TopicID); err != nil {
		w.logger.Warn("report_worker: mark topic completed failed (non-fatal)", zap.Error(err))
	}
	if err := w.discussionRepo.UpdateStatus(ctx, payload.DiscussionID, discussion.StatusCompleted); err != nil {
		w.logger.Warn("report_worker: update discussion status failed (non-fatal)", zap.Error(err))
	}

	w.logger.Info("report_worker: done",
		zap.String("discussion_id", payload.DiscussionID),
		zap.String("report_id", rep.ID),
		zap.Float64("quality_score", rep.QualityScore),
	)
	return nil
}
