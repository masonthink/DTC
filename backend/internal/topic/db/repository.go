// Package topicdb implements topic.Repository and scheduler.TopicRepository using pgx.
package topicdb

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/digital-twin-community/backend/internal/scheduler"
	"github.com/digital-twin-community/backend/internal/topic"
)

// Repository implements topic.Repository backed by PostgreSQL.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository constructs a topic Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create inserts a new topic.
func (r *Repository) Create(ctx context.Context, t *topic.Topic) error {
	const q = `
		INSERT INTO topics
			(id, submitter_user_id, submitter_agent_id, topic_type,
			 title, description, background, tags,
			 status, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::topic_status, $10)`
	_, err := r.db.Exec(ctx, q,
		t.ID, t.SubmitterUserID, t.SubmitterAgentID, string(t.TopicType),
		t.Title, t.Description, t.Background, t.Tags,
		string(t.Status), t.SubmittedAt,
	)
	return err
}

// FindByID retrieves a topic by primary key.
func (r *Repository) FindByID(ctx context.Context, id string) (*topic.Topic, error) {
	const q = topicSelectCols + ` WHERE t.id = $1`
	return r.scanOne(r.db.QueryRow(ctx, q, id))
}

// FindByUserID returns paginated topics for a user.
func (r *Repository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*topic.Topic, error) {
	const q = topicSelectCols + `
		WHERE t.submitter_user_id = $1
		ORDER BY t.submitted_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.collectRows(rows)
}

// UpdateStatus updates the topic status.
func (r *Repository) UpdateStatus(ctx context.Context, id string, status topic.Status) error {
	const q = `UPDATE topics SET status = $2::topic_status, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, string(status))
	return err
}

// MarkMatched records the matched_at timestamp and transitions to 'matched'.
func (r *Repository) MarkMatched(ctx context.Context, id string, matchedAt time.Time) error {
	const q = `
		UPDATE topics
		SET status = 'matched'::topic_status, matched_at = $2, updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, matchedAt)
	return err
}

// MarkDiscussionStarted transitions the topic to 'discussion_active'.
func (r *Repository) MarkDiscussionStarted(ctx context.Context, id string) error {
	const q = `
		UPDATE topics
		SET status = 'discussion_active'::topic_status,
		    discussion_started_at = NOW(), updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

// MarkReportReady transitions the topic to 'report_generating'.
func (r *Repository) MarkReportReady(ctx context.Context, id string) error {
	const q = `
		UPDATE topics
		SET status = 'report_generating'::topic_status,
		    report_ready_at = NOW(), updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

// MarkCompleted transitions the topic to 'completed'.
func (r *Repository) MarkCompleted(ctx context.Context, id string) error {
	const q = `
		UPDATE topics
		SET status = 'completed'::topic_status,
		    completed_at = NOW(), updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

// SetNotified records that a time-based notification was sent.
// notificationType must be one of "1h", "12h", "48h".
func (r *Repository) SetNotified(ctx context.Context, id string, notificationType string) error {
	var col string
	switch notificationType {
	case "1h":
		col = "notified_1h"
	case "12h":
		col = "notified_12h"
	case "48h":
		col = "notified_48h"
	default:
		return fmt.Errorf("unknown notification type: %s", notificationType)
	}
	q := fmt.Sprintf(`UPDATE topics SET %s = true, updated_at = NOW() WHERE id = $1`, col)
	_, err := r.db.Exec(ctx, q, id)
	return err
}

// FindPendingNotifications finds topics that matched >= `since` ago and haven't been notified yet.
func (r *Repository) FindPendingNotifications(ctx context.Context, notifType string, since time.Time) ([]*topic.Topic, error) {
	var col string
	switch notifType {
	case "1h":
		col = "notified_1h"
	case "12h":
		col = "notified_12h"
	case "48h":
		col = "notified_48h"
	default:
		return nil, fmt.Errorf("unknown notif type: %s", notifType)
	}
	q := topicSelectCols + fmt.Sprintf(`
		WHERE t.%s = false
		  AND t.matched_at IS NOT NULL
		  AND t.matched_at <= $1
		  AND t.status NOT IN ('cancelled', 'failed')
		ORDER BY t.matched_at ASC
		LIMIT 100`, col)
	rows, err := r.db.Query(ctx, q, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.collectRows(rows)
}

// FindPendingMatching returns topics waiting for agent matching.
func (r *Repository) FindPendingMatching(ctx context.Context, limit int) ([]*topic.Topic, error) {
	const q = topicSelectCols + `
		WHERE t.status = 'pending_matching'::topic_status
		ORDER BY t.submitted_at ASC
		LIMIT $1`
	rows, err := r.db.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.collectRows(rows)
}

// topicSelectCols is the base SELECT clause for topics, including the discussion ID if one exists.
const topicSelectCols = `
	SELECT t.id, t.submitter_user_id, t.submitter_agent_id, t.topic_type,
	       t.title, t.description, t.background, t.tags, t.status::text,
	       t.submitted_at, t.matched_at, t.discussion_started_at, t.report_ready_at, t.completed_at,
	       t.notified_1h, t.notified_12h, t.notified_48h,
	       d.id AS discussion_id
	FROM topics t
	LEFT JOIN discussions d ON d.topic_id = t.id`

func (r *Repository) scanOne(row pgx.Row) (*topic.Topic, error) {
	t, err := scanTopicRow(row.Scan)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repository) collectRows(rows pgx.Rows) ([]*topic.Topic, error) {
	topics := make([]*topic.Topic, 0)
	for rows.Next() {
		t, err := scanTopicRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		topics = append(topics, t)
	}
	return topics, rows.Err()
}

func scanTopicRow(scan func(dest ...any) error) (*topic.Topic, error) {
	var t topic.Topic
	var topicType, status string
	var tags []string
	var matchedAt, discussionStartedAt, reportReadyAt, completedAt *time.Time
	var discussionID *string

	err := scan(
		&t.ID, &t.SubmitterUserID, &t.SubmitterAgentID, &topicType,
		&t.Title, &t.Description, &t.Background, &tags, &status,
		&t.SubmittedAt, &matchedAt, &discussionStartedAt, &reportReadyAt, &completedAt,
		&t.Notified1h, &t.Notified12h, &t.Notified48h,
		&discussionID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan topic: %w", err)
	}

	t.TopicType = topic.TopicType(topicType)
	t.Status = topic.Status(status)
	t.Tags = tags
	t.MatchedAt = matchedAt
	t.DiscussionStartedAt = discussionStartedAt
	t.ReportReadyAt = reportReadyAt
	t.CompletedAt = completedAt
	t.DiscussionID = discussionID
	return &t, nil
}

// =============================================================================
// SchedulerRepository implements scheduler.TopicRepository
// =============================================================================

// SchedulerRepository implements scheduler.TopicRepository backed by PostgreSQL.
type SchedulerRepository struct {
	db *pgxpool.Pool
}

// NewSchedulerRepository constructs a SchedulerRepository.
func NewSchedulerRepository(db *pgxpool.Pool) *SchedulerRepository {
	return &SchedulerRepository{db: db}
}

// FindPendingMatching returns lightweight topic refs waiting for matching.
func (s *SchedulerRepository) FindPendingMatching(ctx context.Context, limit int) ([]*scheduler.TopicRef, error) {
	const q = `
		SELECT id, status::text, submitted_at, matched_at, notified_1h, notified_12h, notified_48h
		FROM topics
		WHERE status = 'pending_matching'::topic_status
		ORDER BY submitted_at ASC
		LIMIT $1`
	return s.queryRefs(ctx, q, limit)
}

// FindReadyFor1hNotification returns topics matched >= 1h ago without 1h notification.
func (s *SchedulerRepository) FindReadyFor1hNotification(ctx context.Context) ([]*scheduler.TopicRef, error) {
	const q = `
		SELECT id, status::text, submitted_at, matched_at, notified_1h, notified_12h, notified_48h
		FROM topics
		WHERE notified_1h = false
		  AND matched_at IS NOT NULL
		  AND matched_at <= NOW() - INTERVAL '1 hour'
		  AND status NOT IN ('cancelled', 'failed')
		ORDER BY matched_at ASC
		LIMIT 100`
	return s.queryRefs(ctx, q)
}

// FindReadyFor12hNotification returns topics matched >= 12h ago without 12h notification.
func (s *SchedulerRepository) FindReadyFor12hNotification(ctx context.Context) ([]*scheduler.TopicRef, error) {
	const q = `
		SELECT id, status::text, submitted_at, matched_at, notified_1h, notified_12h, notified_48h
		FROM topics
		WHERE notified_12h = false
		  AND matched_at IS NOT NULL
		  AND matched_at <= NOW() - INTERVAL '12 hours'
		  AND status NOT IN ('cancelled', 'failed')
		ORDER BY matched_at ASC
		LIMIT 100`
	return s.queryRefs(ctx, q)
}

// FindReadyFor48hNotification returns topics matched >= 48h ago without 48h notification.
func (s *SchedulerRepository) FindReadyFor48hNotification(ctx context.Context) ([]*scheduler.TopicRef, error) {
	const q = `
		SELECT id, status::text, submitted_at, matched_at, notified_1h, notified_12h, notified_48h
		FROM topics
		WHERE notified_48h = false
		  AND matched_at IS NOT NULL
		  AND matched_at <= NOW() - INTERVAL '48 hours'
		  AND status NOT IN ('cancelled', 'failed')
		ORDER BY matched_at ASC
		LIMIT 100`
	return s.queryRefs(ctx, q)
}

// FindActiveDiscussions returns topics that are currently in discussion.
func (s *SchedulerRepository) FindActiveDiscussions(ctx context.Context) ([]*scheduler.TopicRef, error) {
	const q = `
		SELECT id, status::text, submitted_at, matched_at, notified_1h, notified_12h, notified_48h
		FROM topics
		WHERE status = 'discussion_active'::topic_status
		ORDER BY matched_at ASC
		LIMIT 100`
	return s.queryRefs(ctx, q)
}

func (s *SchedulerRepository) queryRefs(ctx context.Context, q string, args ...interface{}) ([]*scheduler.TopicRef, error) {
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []*scheduler.TopicRef
	for rows.Next() {
		var ref scheduler.TopicRef
		var matchedAt *time.Time
		if err := rows.Scan(
			&ref.ID, &ref.Status, &ref.SubmittedAt, &matchedAt,
			&ref.Notified1h, &ref.Notified12h, &ref.Notified48h,
		); err != nil {
			return nil, fmt.Errorf("scan topic ref: %w", err)
		}
		ref.MatchedAt = matchedAt
		refs = append(refs, &ref)
	}
	return refs, rows.Err()
}
