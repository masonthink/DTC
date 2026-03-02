// Package discussiondb implements discussion.Repository using pgx.
package discussiondb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/digital-twin-community/backend/internal/discussion"
)

// Repository implements discussion.Repository backed by PostgreSQL.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository constructs a discussion Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create inserts a new Discussion record.
func (r *Repository) Create(ctx context.Context, d *discussion.Discussion) error {
	participantsJSON, err := json.Marshal(d.Participants)
	if err != nil {
		return fmt.Errorf("marshal participants: %w", err)
	}
	const q = `
		INSERT INTO discussions
			(id, topic_id, status, current_round, error_count, participants, is_degraded, degraded_reason)
		VALUES ($1, $2, $3::discussion_status, $4, $5, $6, $7, $8)`
	_, err = r.db.Exec(ctx, q,
		d.ID, d.TopicID, dbStatus(d.Status),
		d.CurrentRound, d.ErrorCount, participantsJSON,
		d.IsDegraded, d.DegradedReason,
	)
	return err
}

// FindByID retrieves a Discussion by primary key. Messages are not populated.
func (r *Repository) FindByID(ctx context.Context, id string) (*discussion.Discussion, error) {
	const q = `
		SELECT id, topic_id, status::text, current_round, error_count,
		       participants, is_degraded, COALESCE(degraded_reason, '')
		FROM discussions WHERE id = $1`
	return r.scanOne(r.db.QueryRow(ctx, q, id))
}

// FindByTopicID retrieves the Discussion for a given topic.
func (r *Repository) FindByTopicID(ctx context.Context, topicID string) (*discussion.Discussion, error) {
	const q = `
		SELECT id, topic_id, status::text, current_round, error_count,
		       participants, is_degraded, COALESCE(degraded_reason, '')
		FROM discussions WHERE topic_id = $1`
	return r.scanOne(r.db.QueryRow(ctx, q, topicID))
}

// UpdateStatus updates the discussion status and infers current_round from status name.
func (r *Repository) UpdateStatus(ctx context.Context, id string, status discussion.DiscussionStatus) error {
	const q = `
		UPDATE discussions SET
			status = $2::discussion_status,
			current_round = CASE
				WHEN $3 ~ '^round_[1-4]_' THEN CAST(SUBSTRING($3 FROM '^round_([1-4])_') AS integer)
				ELSE current_round
			END,
			updated_at = NOW()
		WHERE id = $1`
	s := dbStatus(status)
	_, err := r.db.Exec(ctx, q, id, s, s)
	return err
}

// SaveMessage persists one RoundMessage to discussion_messages.
func (r *Repository) SaveMessage(ctx context.Context, discussionID string, roundNum int, msg *discussion.RoundMessage) error {
	idempotencyKey := fmt.Sprintf("%s:round%d:%s:%s", discussionID, roundNum, msg.AgentID, uuid.NewString()[:8])
	const q = `
		INSERT INTO discussion_messages
			(discussion_id, agent_id, round_number, role, content, key_point,
			 addressed_to, confidence, similarity_to_prev, was_rewritten,
			 model_used, prompt_tokens, completion_tokens, idempotency_key)
		VALUES ($1, $2, $3, $4::discussion_role, $5, $6,
		        $7::discussion_role, $8, $9, $10,
		        $11, $12, $13, $14)
		ON CONFLICT (idempotency_key) DO NOTHING`
	_, err := r.db.Exec(ctx, q,
		discussionID, msg.AgentID, roundNum, string(msg.Role),
		msg.Content, msg.KeyPoint,
		string(msg.AddressedTo), msg.Confidence, msg.SimilarityToPrev, msg.WasRewritten,
		msg.ModelUsed, msg.PromptTokens, msg.CompletionTokens, idempotencyKey,
	)
	return err
}

// FindMessages returns all messages for a discussion in round/creation order.
func (r *Repository) FindMessages(ctx context.Context, discussionID string) ([]*discussion.RoundMessage, error) {
	const q = `
		SELECT agent_id::text, role::text, content, key_point,
		       addressed_to::text, confidence, similarity_to_prev, was_rewritten,
		       COALESCE(model_used, ''), prompt_tokens, completion_tokens, round_number
		FROM discussion_messages
		WHERE discussion_id = $1
		ORDER BY round_number ASC, created_at ASC`
	rows, err := r.db.Query(ctx, q, discussionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*discussion.RoundMessage
	for rows.Next() {
		var msg discussion.RoundMessage
		var role, addressedTo string
		if err := rows.Scan(
			&msg.AgentID, &role, &msg.Content, &msg.KeyPoint,
			&addressedTo, &msg.Confidence, &msg.SimilarityToPrev, &msg.WasRewritten,
			&msg.ModelUsed, &msg.PromptTokens, &msg.CompletionTokens, &msg.RoundNum,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msg.Role = discussion.Role(role)
		msg.AddressedTo = discussion.Role(addressedTo)
		msgs = append(msgs, &msg)
	}
	return msgs, rows.Err()
}

// SaveAnonMappings inserts anon_id → agent_id records into the anon schema for privacy audit.
func (r *Repository) SaveAnonMappings(ctx context.Context, discussionID string, participants []discussion.Participant) error {
	for _, p := range participants {
		const q = `
			INSERT INTO anon.anon_id_mappings (anon_id, agent_id, discussion_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (anon_id, discussion_id) DO NOTHING`
		if _, err := r.db.Exec(ctx, q, p.AnonID, p.AgentID, discussionID); err != nil {
			return fmt.Errorf("save anon mapping for %s: %w", p.AnonID, err)
		}
	}
	return nil
}

// scanOne reads a single Discussion from a pgx.Row (messages not included).
func (r *Repository) scanOne(row pgx.Row) (*discussion.Discussion, error) {
	var d discussion.Discussion
	var statusStr string
	var participantsJSON []byte

	err := row.Scan(
		&d.ID, &d.TopicID, &statusStr, &d.CurrentRound, &d.ErrorCount,
		&participantsJSON, &d.IsDegraded, &d.DegradedReason,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan discussion: %w", err)
	}

	d.Status = discussion.DiscussionStatus(strings.ToUpper(statusStr))

	if err := json.Unmarshal(participantsJSON, &d.Participants); err != nil {
		return nil, fmt.Errorf("unmarshal participants: %w", err)
	}
	return &d, nil
}

// dbStatus converts a Go discussion status (uppercase) to its DB enum value (lowercase).
func dbStatus(s discussion.DiscussionStatus) string {
	return strings.ToLower(string(s))
}
