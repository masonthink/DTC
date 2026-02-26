package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/digital-twin-community/backend/internal/discussion"
	"github.com/digital-twin-community/backend/internal/notification"
	"github.com/digital-twin-community/backend/internal/report"
	"github.com/digital-twin-community/backend/internal/topic"
)

// NotifyWorker handles time-based notification tasks (T+1h, T+12h, T+48h).
type NotifyWorker struct {
	notifSvc       *notification.Service
	topicRepo      topic.Repository
	discussionRepo discussion.Repository
	reportRepo     report.Repository
	logger         *zap.Logger
}

// NewNotifyWorker constructs a NotifyWorker.
func NewNotifyWorker(
	notifSvc *notification.Service,
	topicRepo topic.Repository,
	discussionRepo discussion.Repository,
	reportRepo report.Repository,
	logger *zap.Logger,
) *NotifyWorker {
	return &NotifyWorker{
		notifSvc:       notifSvc,
		topicRepo:      topicRepo,
		discussionRepo: discussionRepo,
		reportRepo:     reportRepo,
		logger:         logger,
	}
}

// Handle1h handles "notify:1h" — match preview notification sent ~T+1h after matching.
func (w *NotifyWorker) Handle1h(ctx context.Context, task *asynq.Task) error {
	topicID, err := extractTopicID(task)
	if err != nil {
		return err
	}

	t, err := w.topicRepo.FindByID(ctx, topicID)
	if err != nil || t == nil {
		return fmt.Errorf("notify_worker 1h: load topic %s: %w", topicID, err)
	}
	if t.Notified1h {
		w.logger.Info("notify_worker 1h: already sent, skipping", zap.String("topic_id", topicID))
		return nil
	}

	if err := w.notifSvc.SendMatchPreview(ctx, t.SubmitterUserID, t.ID, t.Title); err != nil {
		// Non-fatal: log and proceed to mark as notified so we don't retry spam.
		w.logger.Warn("notify_worker 1h: send failed (marking notified anyway)",
			zap.String("topic_id", topicID), zap.Error(err))
	}

	return w.topicRepo.SetNotified(ctx, topicID, "1h")
}

// Handle12h handles "notify:12h" — discussion update notification sent ~T+12h after matching.
func (w *NotifyWorker) Handle12h(ctx context.Context, task *asynq.Task) error {
	topicID, err := extractTopicID(task)
	if err != nil {
		return err
	}

	t, err := w.topicRepo.FindByID(ctx, topicID)
	if err != nil || t == nil {
		return fmt.Errorf("notify_worker 12h: load topic %s: %w", topicID, err)
	}
	if t.Notified12h {
		w.logger.Info("notify_worker 12h: already sent, skipping", zap.String("topic_id", topicID))
		return nil
	}

	if err := w.notifSvc.SendDiscussionUpdate(ctx, t.SubmitterUserID, t.ID, t.Title); err != nil {
		w.logger.Warn("notify_worker 12h: send failed (marking notified anyway)",
			zap.String("topic_id", topicID), zap.Error(err))
	}

	return w.topicRepo.SetNotified(ctx, topicID, "12h")
}

// Handle48h handles "notify:48h" — report ready notification sent ~T+48h after matching.
// It also resolves the report_id from the discussion so the client can deep-link.
func (w *NotifyWorker) Handle48h(ctx context.Context, task *asynq.Task) error {
	topicID, err := extractTopicID(task)
	if err != nil {
		return err
	}

	t, err := w.topicRepo.FindByID(ctx, topicID)
	if err != nil || t == nil {
		return fmt.Errorf("notify_worker 48h: load topic %s: %w", topicID, err)
	}
	if t.Notified48h {
		w.logger.Info("notify_worker 48h: already sent, skipping", zap.String("topic_id", topicID))
		return nil
	}

	// Resolve report ID for deep-link (best-effort; empty string is acceptable).
	reportID := w.resolveReportID(ctx, topicID)

	if err := w.notifSvc.SendReportReady(ctx, t.SubmitterUserID, t.ID, reportID, t.Title); err != nil {
		w.logger.Warn("notify_worker 48h: send failed (marking notified anyway)",
			zap.String("topic_id", topicID), zap.Error(err))
	}

	return w.topicRepo.SetNotified(ctx, topicID, "48h")
}

// resolveReportID looks up the report ID for a topic via its discussion.
func (w *NotifyWorker) resolveReportID(ctx context.Context, topicID string) string {
	disc, err := w.discussionRepo.FindByTopicID(ctx, topicID)
	if err != nil || disc == nil {
		return ""
	}
	rep, err := w.reportRepo.FindByDiscussionID(ctx, disc.ID)
	if err != nil || rep == nil {
		return ""
	}
	return rep.ID
}

// extractTopicID is a shared helper for all notify task payloads.
func extractTopicID(task *asynq.Task) (string, error) {
	var payload struct {
		TopicID string `json:"topic_id"`
	}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return "", fmt.Errorf("unmarshal notify payload: %w", err)
	}
	if payload.TopicID == "" {
		return "", fmt.Errorf("notify payload missing topic_id")
	}
	return payload.TopicID, nil
}
