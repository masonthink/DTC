// Package scheduler handles time-based triggers for the discussion lifecycle.
// It runs as a background goroutine and fires tasks at T+1h, T+12h, T+48h.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
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
	pool      *pgxpool.Pool
	logger    *zap.Logger
	stopCh    chan struct{}
}

// NewScheduler constructs a Scheduler.
func NewScheduler(client *asynq.Client, inspector *asynq.Inspector, repo TopicRepository, pool *pgxpool.Pool, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		client:    client,
		inspector: inspector,
		repo:      repo,
		pool:      pool,
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
		{"repair-inconsistent-statuses", s.repairInconsistentStatuses},
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
		1: matchedAt.Add(2 * time.Minute),  // T+2min (prod: T+1.5h)
		2: matchedAt.Add(5 * time.Minute),  // T+5min (prod: T+4h)
		3: matchedAt.Add(8 * time.Minute),  // T+8min (prod: T+8h)
		4: matchedAt.Add(11 * time.Minute), // T+11min (prod: T+12h)
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
		asynq.Unique(5*time.Minute),
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

// repairInconsistentStatuses fixes status mismatches caused by partial failures.
// It runs every scheduler tick (5 min) and repairs the following cases:
//  1. Discussion has status REPORT_GENERATING but a report already exists → mark COMPLETED
//  2. Topic has status report_generating but discussion is COMPLETED → mark completed
//  3. Discussion stuck in ROUND_4_COMPLETED for >15 min without report task → re-enqueue report
//  3b. Discussion stuck in REPORT_GENERATING for >10 min without report → reset to ROUND_4_COMPLETED
func (s *Scheduler) repairInconsistentStatuses(ctx context.Context) {
	if s.pool == nil {
		return
	}

	// Case 1: discussion = REPORT_GENERATING but report exists → fix discussion
	res, err := s.pool.Exec(ctx, `
		UPDATE discussions d
		SET status = 'completed'::discussion_status, updated_at = NOW()
		FROM reports r
		WHERE r.discussion_id = d.id
		  AND d.status = 'report_generating'::discussion_status`)
	if err != nil {
		s.logger.Error("repair: fix discussion status from report_generating", zap.Error(err))
	} else if res.RowsAffected() > 0 {
		s.logger.Warn("repair: fixed discussion status REPORT_GENERATING → COMPLETED",
			zap.Int64("count", res.RowsAffected()))
	}

	// Case 2: topic = report_generating but discussion = COMPLETED → fix topic
	res, err = s.pool.Exec(ctx, `
		UPDATE topics t
		SET status = 'completed'::topic_status, completed_at = NOW(), updated_at = NOW()
		FROM discussions d
		WHERE d.topic_id = t.id
		  AND d.status = 'completed'::discussion_status
		  AND t.status = 'report_generating'::topic_status`)
	if err != nil {
		s.logger.Error("repair: fix topic status from report_generating", zap.Error(err))
	} else if res.RowsAffected() > 0 {
		s.logger.Warn("repair: fixed topic status report_generating → completed",
			zap.Int64("count", res.RowsAffected()))
	}

	// Case 3: discussion stuck in ROUND_4_COMPLETED > 15 min, no report → re-enqueue
	rows, err := s.pool.Query(ctx, `
		SELECT d.id, d.topic_id
		FROM discussions d
		LEFT JOIN reports r ON r.discussion_id = d.id
		WHERE d.status = 'round_4_completed'::discussion_status
		  AND r.id IS NULL
		  AND d.updated_at < NOW() - INTERVAL '15 minutes'`)
	if err != nil {
		s.logger.Error("repair: find stuck round_4_completed discussions", zap.Error(err))
		return
	}
	defer rows.Close()

	for rows.Next() {
		var discID, topicID string
		if err := rows.Scan(&discID, &topicID); err != nil {
			s.logger.Error("repair: scan stuck discussion", zap.Error(err))
			continue
		}
		s.logger.Warn("repair: re-enqueuing report generation for stuck discussion",
			zap.String("discussion_id", discID))
		if err := s.EnqueueReportGeneration(ctx, discID, topicID); err != nil {
			s.logger.Error("repair: re-enqueue report failed",
				zap.String("discussion_id", discID), zap.Error(err))
		}
	}

	// Case 3b: discussion stuck in REPORT_GENERATING > 10 min, no report → reset to ROUND_4_COMPLETED
	// This handles the case where the report generation task was lost (e.g., after a restart).
	// Case 3 will then re-enqueue the report on the next tick.
	res, err = s.pool.Exec(ctx, `
		UPDATE discussions d
		SET status = 'round_4_completed'::discussion_status, updated_at = NOW() - INTERVAL '20 minutes'
		WHERE d.status = 'report_generating'::discussion_status
		  AND NOT EXISTS (SELECT 1 FROM reports r WHERE r.discussion_id = d.id)
		  AND d.updated_at < NOW() - INTERVAL '10 minutes'`)
	if err != nil {
		s.logger.Error("repair: reset stuck report_generating discussions", zap.Error(err))
	} else if res.RowsAffected() > 0 {
		s.logger.Warn("repair: reset stuck REPORT_GENERATING → ROUND_4_COMPLETED for re-enqueue",
			zap.Int64("count", res.RowsAffected()))
	}

	// Case 4: discussion stuck in early round states (queued/running/completed for rounds 1-3)
	// for >30 min with 0 messages for that round → re-enqueue the round
	s.repairStuckRounds(ctx)
}

// repairStuckRounds finds discussions stuck in round-level states and re-enqueues them.
func (s *Scheduler) repairStuckRounds(ctx context.Context) {
	// Find discussions stuck in a round state for >30 min
	rows, err := s.pool.Query(ctx, `
		SELECT d.id, d.topic_id, d.status::text,
		       (SELECT COALESCE(MAX(dm.round_number), 0)
		        FROM discussion_messages dm WHERE dm.discussion_id = d.id) AS max_round
		FROM discussions d
		WHERE d.status IN (
			'round_1_queued'::discussion_status, 'round_1_running'::discussion_status,
			'round_1_completed'::discussion_status,
			'round_2_queued'::discussion_status, 'round_2_running'::discussion_status,
			'round_2_completed'::discussion_status,
			'round_3_queued'::discussion_status, 'round_3_running'::discussion_status,
			'round_3_completed'::discussion_status,
			'round_4_queued'::discussion_status, 'round_4_running'::discussion_status
		)
		AND d.updated_at < NOW() - INTERVAL '30 minutes'`)
	if err != nil {
		s.logger.Error("repair: find stuck round discussions", zap.Error(err))
		return
	}
	defer rows.Close()

	for rows.Next() {
		var discID, topicID, status string
		var maxRound int
		if err := rows.Scan(&discID, &topicID, &status, &maxRound); err != nil {
			s.logger.Error("repair: scan stuck round discussion", zap.Error(err))
			continue
		}

		// Determine which round to re-enqueue based on completed messages
		nextRound := maxRound + 1
		if nextRound < 1 {
			nextRound = 1
		}
		if nextRound > 4 {
			// All rounds done but status not advanced → enqueue report
			s.logger.Warn("repair: all rounds done but status stuck, re-enqueuing report",
				zap.String("discussion_id", discID), zap.String("status", status))
			if err := s.EnqueueReportGeneration(ctx, discID, topicID); err != nil {
				s.logger.Error("repair: re-enqueue report failed",
					zap.String("discussion_id", discID), zap.Error(err))
			}
			continue
		}

		s.logger.Warn("repair: re-enqueuing stuck round",
			zap.String("discussion_id", discID),
			zap.String("status", status),
			zap.Int("next_round", nextRound))
		scheduleAt := time.Now().Add(30 * time.Second)
		if err := s.EnqueueDiscussionRound(ctx, discID, nextRound, scheduleAt); err != nil {
			s.logger.Error("repair: re-enqueue round failed",
				zap.String("discussion_id", discID),
				zap.Int("round", nextRound),
				zap.Error(err))
		}
	}
}

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mustMarshal: %v", err))
	}
	return b
}
