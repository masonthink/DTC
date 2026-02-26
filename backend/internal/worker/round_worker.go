package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/digital-twin-community/backend/internal/discussion"
	"github.com/digital-twin-community/backend/internal/scheduler"
	"github.com/digital-twin-community/backend/internal/topic"
)

// DiscussionRoundWorker handles the "discussion:round" task.
type DiscussionRoundWorker struct {
	discussionRepo discussion.Repository
	topicRepo      topic.Repository
	engine         *discussion.Engine
	sched          *scheduler.Scheduler
	logger         *zap.Logger
}

// NewDiscussionRoundWorker constructs a DiscussionRoundWorker.
func NewDiscussionRoundWorker(
	discussionRepo discussion.Repository,
	topicRepo topic.Repository,
	engine *discussion.Engine,
	sched *scheduler.Scheduler,
	logger *zap.Logger,
) *DiscussionRoundWorker {
	return &DiscussionRoundWorker{
		discussionRepo: discussionRepo,
		topicRepo:      topicRepo,
		engine:         engine,
		sched:          sched,
		logger:         logger,
	}
}

// Handle processes one "discussion:round" task.
func (w *DiscussionRoundWorker) Handle(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		DiscussionID string `json:"discussion_id"`
		Round        int    `json:"round"`
	}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	w.logger.Info("round_worker: starting",
		zap.String("discussion_id", payload.DiscussionID),
		zap.Int("round", payload.Round),
	)

	// ── 1. Load discussion ───────────────────────────────────────────────────
	disc, err := w.discussionRepo.FindByID(ctx, payload.DiscussionID)
	if err != nil {
		return fmt.Errorf("load discussion: %w", err)
	}
	if disc == nil {
		return fmt.Errorf("discussion %s not found", payload.DiscussionID)
	}

	// ── 2. Load prior messages for conversation history ──────────────────────
	priorMsgs, err := w.discussionRepo.FindMessages(ctx, payload.DiscussionID)
	if err != nil {
		return fmt.Errorf("load messages: %w", err)
	}
	for _, m := range priorMsgs {
		disc.Messages = append(disc.Messages, discussion.Message{
			RoundNum: m.RoundNum,
			AgentID:  m.AgentID,
			Role:     m.Role,
			Content:  m.Content,
			KeyPoint: m.KeyPoint,
		})
	}

	// ── 3. Ensure TopicText is populated ─────────────────────────────────────
	if disc.TopicText == "" {
		t, err := w.topicRepo.FindByID(ctx, disc.TopicID)
		if err == nil && t != nil {
			disc.TopicText = t.TextForEmbedding()
		}
	}

	// ── 4. Mark topic as discussion_active on round 1 ────────────────────────
	if payload.Round == 1 {
		if err := w.topicRepo.MarkDiscussionStarted(ctx, disc.TopicID); err != nil {
			w.logger.Warn("round_worker: mark discussion started failed (non-fatal)",
				zap.Error(err),
			)
		}
	}

	// ── 5. Transition to RUNNING status before LLM calls ─────────────────────
	runningStatus := roundRunningStatus(payload.Round)
	disc.CurrentRound = payload.Round
	if err := w.discussionRepo.UpdateStatus(ctx, disc.ID, runningStatus); err != nil {
		w.logger.Warn("round_worker: update running status failed (non-fatal)", zap.Error(err))
	}

	// ── 6. Run the round ──────────────────────────────────────────────────────
	msgs, err := w.engine.RunRound(ctx, disc, payload.Round)
	if err != nil {
		_ = w.discussionRepo.UpdateStatus(ctx, disc.ID, discussion.StatusDegraded)
		return fmt.Errorf("run round %d: %w", payload.Round, err)
	}

	// ── 7. Persist messages ───────────────────────────────────────────────────
	for _, msg := range msgs {
		if err := w.discussionRepo.SaveMessage(ctx, disc.ID, payload.Round, msg); err != nil {
			w.logger.Error("round_worker: save message failed",
				zap.String("agent_id", msg.AgentID),
				zap.Error(err),
			)
		}
	}

	// ── 8. Transition to COMPLETED status ─────────────────────────────────────
	completedStatus := roundCompletedStatus(payload.Round)
	if err := w.discussionRepo.UpdateStatus(ctx, disc.ID, completedStatus); err != nil {
		w.logger.Warn("round_worker: update completed status failed (non-fatal)", zap.Error(err))
	}

	// ── 9. After round 4: enqueue report generation ───────────────────────────
	if payload.Round == 4 {
		_ = w.discussionRepo.UpdateStatus(ctx, disc.ID, discussion.StatusReportGenerating)
		if err := w.topicRepo.MarkReportReady(ctx, disc.TopicID); err != nil {
			w.logger.Warn("round_worker: mark report ready failed (non-fatal)", zap.Error(err))
		}
		if err := w.sched.EnqueueReportGeneration(ctx, disc.ID, disc.TopicID); err != nil {
			w.logger.Error("round_worker: enqueue report generation failed", zap.Error(err))
		}
	}

	w.logger.Info("round_worker: done",
		zap.String("discussion_id", disc.ID),
		zap.Int("round", payload.Round),
		zap.Int("messages_saved", len(msgs)),
	)
	return nil
}

// roundRunningStatus returns the RUNNING status for a given round number.
func roundRunningStatus(round int) discussion.DiscussionStatus {
	switch round {
	case 1:
		return discussion.StatusRound1Running
	case 2:
		return discussion.StatusRound2Running
	case 3:
		return discussion.StatusRound3Running
	case 4:
		return discussion.StatusRound4Running
	}
	return discussion.StatusDegraded
}

// roundCompletedStatus returns the COMPLETED status for a given round number.
func roundCompletedStatus(round int) discussion.DiscussionStatus {
	switch round {
	case 1:
		return discussion.StatusRound1Completed
	case 2:
		return discussion.StatusRound2Completed
	case 3:
		return discussion.StatusRound3Completed
	case 4:
		return discussion.StatusRound4Completed
	}
	return discussion.StatusDegraded
}
