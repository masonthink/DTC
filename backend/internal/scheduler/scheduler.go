// Package scheduler handles time-based triggers for the discussion lifecycle.
// It runs as a background goroutine and fires tasks at T+1h, T+12h, T+48h.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// TaskType constants for Asynq queues.
const (
	TaskTypeMatchTopic        = "topic:match"
	TaskTypeDiscussionRound   = "discussion:round"
	TaskTypeGenerateReport    = "report:generate"
	TaskTypeNotify1h          = "notify:1h"
	TaskTypeNotify12h         = "notify:12h"
	TaskTypeNotify48h         = "notify:48h"
	TaskTypeExpireConnections = "connection:expire"
)

// Queue priority names.
const (
	QueueDiscussionHigh = "discussion:high"
	QueueLLMStandard    = "llm:standard"
	QueueNotification   = "notification"
	QueueReport         = "report:generate"
)

// TopicRepository is used to find topics needing time-based actions.
type TopicRepository interface {
	FindPendingMatching(ctx context.Context, limit int) ([]*TopicRef, error)
	FindReadyFor1hNotification(ctx context.Context) ([]*TopicRef, error)
	FindReadyFor12hNotification(ctx context.Context) ([]*TopicRef, error)
	FindReadyFor48hNotification(ctx context.Context) ([]*TopicRef, error)
	FindActiveDiscussions(ctx context.Context) ([]*TopicRef, error)
}

// TopicRef is a lightweight reference to a topic.
type TopicRef struct {
	ID           string
	Status       string
	SubmittedAt  time.Time
	MatchedAt    *time.Time
	Notified1h   bool
	Notified12h  bool
	Notified48h  bool
}

// Scheduler runs periodic background jobs.
type Scheduler struct {
	client    *asynq.Client
	inspector *asynq.Inspector
	repo      TopicRepository
	logger    *zap.Logger
	stopCh    chan struct{}
}

// NewScheduler constructs a Scheduler.
func NewScheduler(client *asynq.Client, inspector *asynq.Inspector, repo TopicRepository, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		client:    client,
		inspector: inspector,
		repo:      repo,
		logger:    logger,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the scheduler loop. It should be run in a goroutine.
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("scheduler started")

	// Run immediately on startup, then on interval
	s.runAllJobs(ctx)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopping")
			return
		case <-s.stopCh:
			s.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.runAllJobs(ctx)
		}
	}
}

// Stop signals the scheduler to stop.
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) runAllJobs(ctx context.Context) {
	jobs := []struct {
		name string
		fn   func(context.Context)
	}{
		{"enqueue-pending-matching", s.enqueuePendingMatching},
		{"enqueue-1h-notifications", s.enqueue1hNotifications},
		{"enqueue-12h-notifications", s.enqueue12hNotifications},
		{"enqueue-48h-notifications", s.enqueue48hNotifications},
		{"expire-connections", s.enqueueExpireConnections},
	}

	for _, job := range jobs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("scheduler job panicked",
						zap.String("job", job.name),
						zap.Any("panic", r),
					)
				}
			}()
			job.fn(ctx)
		}()
	}
}

// enqueuePendingMatching enqueues topics that need agent matching.
func (s *Scheduler) enqueuePendingMatching(ctx context.Context) {
	topics, err := s.repo.FindPendingMatching(ctx, 100)
	if err != nil {
		s.logger.Error("find pending matching topics", zap.Error(err))
		return
	}
	for _, topic := range topics {
		task := asynq.NewTask(TaskTypeMatchTopic, mustMarshal(map[string]string{
			"topic_id": topic.ID,
		}))
		info, err := s.client.EnqueueContext(ctx, task,
			asynq.Queue(QueueDiscussionHigh),
			asynq.MaxRetry(3),
			asynq.Unique(30*time.Minute), // 30分钟内不重复入队
		)
		if err != nil {
			s.logger.Warn("enqueue match topic failed",
				zap.String("topic_id", topic.ID),
				zap.Error(err),
			)
			continue
		}
		s.logger.Debug("enqueued match topic", zap.String("task_id", info.ID))
	}
}

// enqueue1hNotifications enqueues "match preview" notifications (T+1h).
func (s *Scheduler) enqueue1hNotifications(ctx context.Context) {
	s.enqueueNotifications(ctx, s.repo.FindReadyFor1hNotification, TaskTypeNotify1h, "1h")
}

// enqueue12hNotifications enqueues "discussion update" notifications (T+12h).
func (s *Scheduler) enqueue12hNotifications(ctx context.Context) {
	s.enqueueNotifications(ctx, s.repo.FindReadyFor12hNotification, TaskTypeNotify12h, "12h")
}

// enqueue48hNotifications enqueues "full report ready" notifications (T+48h).
func (s *Scheduler) enqueue48hNotifications(ctx context.Context) {
	s.enqueueNotifications(ctx, s.repo.FindReadyFor48hNotification, TaskTypeNotify48h, "48h")
}

func (s *Scheduler) enqueueNotifications(
	ctx context.Context,
	finder func(context.Context) ([]*TopicRef, error),
	taskType, label string,
) {
	topics, err := finder(ctx)
	if err != nil {
		s.logger.Error("find notification topics", zap.String("type", label), zap.Error(err))
		return
	}
	for _, topic := range topics {
		task := asynq.NewTask(taskType, mustMarshal(map[string]string{
			"topic_id": topic.ID,
		}))
		_, err := s.client.EnqueueContext(ctx, task,
			asynq.Queue(QueueNotification),
			asynq.MaxRetry(3),
			asynq.Unique(1*time.Hour),
		)
		if err != nil {
			s.logger.Warn("enqueue notification failed",
				zap.String("topic_id", topic.ID),
				zap.String("type", label),
				zap.Error(err),
			)
		}
	}
}

// enqueueExpireConnections cleans up expired pending connections.
func (s *Scheduler) enqueueExpireConnections(ctx context.Context) {
	task := asynq.NewTask(TaskTypeExpireConnections, nil)
	_, err := s.client.EnqueueContext(ctx, task,
		asynq.Queue(QueueNotification),
		asynq.MaxRetry(1),
		asynq.Unique(1*time.Hour),
	)
	if err != nil {
		s.logger.Warn("enqueue expire connections failed", zap.Error(err))
	}
}

// DiscussionSchedule returns the wall clock times for each discussion round
// relative to the matching time.
func DiscussionSchedule(matchedAt time.Time) map[int]time.Time {
	return map[int]time.Time{
		1: matchedAt.Add(1*time.Hour + 30*time.Minute), // T+1.5h
		2: matchedAt.Add(4 * time.Hour),                // T+4h
		3: matchedAt.Add(8 * time.Hour),                // T+8h
		4: matchedAt.Add(12 * time.Hour),               // T+12h (with 12h notification)
	}
}

// EnqueueDiscussionRound enqueues a specific discussion round at the scheduled time.
func (s *Scheduler) EnqueueDiscussionRound(ctx context.Context, discussionID string, roundNum int, scheduledAt time.Time) error {
	task := asynq.NewTask(TaskTypeDiscussionRound, mustMarshal(map[string]interface{}{
		"discussion_id": discussionID,
		"round":         roundNum,
	}))

	processAt := asynq.ProcessAt(scheduledAt)
	info, err := s.client.EnqueueContext(ctx, task,
		asynq.Queue(QueueDiscussionHigh),
		asynq.MaxRetry(3),
		processAt,
		asynq.Unique(scheduledAt.Sub(time.Now())+30*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("enqueue discussion round %d: %w", roundNum, err)
	}
	s.logger.Info("enqueued discussion round",
		zap.String("task_id", info.ID),
		zap.String("discussion_id", discussionID),
		zap.Int("round", roundNum),
		zap.Time("scheduled_at", scheduledAt),
	)
	return nil
}

// EnqueueReportGeneration enqueues report generation after Round 4 completes.
func (s *Scheduler) EnqueueReportGeneration(ctx context.Context, discussionID, topicID string) error {
	task := asynq.NewTask(TaskTypeGenerateReport, mustMarshal(map[string]string{
		"discussion_id": discussionID,
		"topic_id":      topicID,
	}))

	info, err := s.client.EnqueueContext(ctx, task,
		asynq.Queue(QueueReport),
		asynq.MaxRetry(2),
	)
	if err != nil {
		return fmt.Errorf("enqueue report generation: %w", err)
	}
	s.logger.Info("enqueued report generation",
		zap.String("task_id", info.ID),
		zap.String("discussion_id", discussionID),
	)
	return nil
}

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mustMarshal: %v", err))
	}
	return b
}
