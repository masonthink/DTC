// Package reportdb implements report.Repository using pgx.
package reportdb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/digital-twin-community/backend/internal/report"
)

// Repository implements report.Repository backed by PostgreSQL.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository constructs a report Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Save persists a new report. It populates r.ID if it was empty.
func (r *Repository) Save(ctx context.Context, rep *report.Report) error {
	if rep.ID == "" {
		rep.ID = uuid.NewString()
	}

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

	const q = `
		INSERT INTO reports
			(id, discussion_id, topic_id, summary,
			 consensus_points, divergence_points, key_questions, action_items, blind_spots,
			 recommended_agents, quality_score, model_used, total_tokens,
			 generation_attempts, generated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	_, err = r.db.Exec(ctx, q,
		rep.ID, rep.DiscussionID, rep.TopicID, rep.Summary,
		consensusJSON, divergenceJSON, questionsJSON, actionsJSON, blindSpotsJSON,
		recommendedJSON, rep.QualityScore, rep.ModelUsed, rep.TotalTokens,
		rep.GenerationAttempts, rep.GeneratedAt,
	)
	return err
}

// FindByID retrieves a report by primary key.
func (r *Repository) FindByID(ctx context.Context, id string) (*report.Report, error) {
	const q = reportSelectCols + ` WHERE id = $1`
	return r.scanOne(r.db.QueryRow(ctx, q, id))
}

// FindByDiscussionID retrieves the report for a discussion.
func (r *Repository) FindByDiscussionID(ctx context.Context, discussionID string) (*report.Report, error) {
	const q = reportSelectCols + ` WHERE discussion_id = $1`
	return r.scanOne(r.db.QueryRow(ctx, q, discussionID))
}

// UpdateUserRating stores a user's star rating (1-5) and optional feedback.
func (r *Repository) UpdateUserRating(ctx context.Context, id string, rating int, feedback string) error {
	const q = `
		UPDATE reports
		SET user_rating = $2, user_feedback = NULLIF($3, ''), updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, rating, feedback)
	return err
}

const reportSelectCols = `
	SELECT id::text, discussion_id::text, topic_id::text, summary,
	       consensus_points, divergence_points, key_questions, action_items, blind_spots,
	       recommended_agents, quality_score,
	       COALESCE(user_rating, 0), COALESCE(user_feedback, ''),
	       COALESCE(model_used, ''), COALESCE(total_tokens, 0),
	       COALESCE(generation_attempts, 1), generated_at
	FROM reports`

func (r *Repository) scanOne(row pgx.Row) (*report.Report, error) {
	var rep report.Report
	var consensusJSON, divergenceJSON, questionsJSON, actionsJSON, blindSpotsJSON []byte
	var recommendedJSON []byte

	err := row.Scan(
		&rep.ID, &rep.DiscussionID, &rep.TopicID, &rep.Summary,
		&consensusJSON, &divergenceJSON, &questionsJSON, &actionsJSON, &blindSpotsJSON,
		&recommendedJSON, &rep.QualityScore,
		&rep.UserRating, &rep.UserFeedback,
		&rep.ModelUsed, &rep.TotalTokens,
		&rep.GenerationAttempts, &rep.GeneratedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan report: %w", err)
	}

	if err := json.Unmarshal(consensusJSON, &rep.OpinionMatrix.ConsensusPoints); err != nil {
		return nil, fmt.Errorf("unmarshal consensus_points: %w", err)
	}
	if err := json.Unmarshal(divergenceJSON, &rep.OpinionMatrix.DivergencePoints); err != nil {
		return nil, fmt.Errorf("unmarshal divergence_points: %w", err)
	}
	if err := json.Unmarshal(questionsJSON, &rep.OpinionMatrix.KeyQuestions); err != nil {
		return nil, fmt.Errorf("unmarshal key_questions: %w", err)
	}
	if err := json.Unmarshal(actionsJSON, &rep.OpinionMatrix.ActionItems); err != nil {
		return nil, fmt.Errorf("unmarshal action_items: %w", err)
	}
	if err := json.Unmarshal(blindSpotsJSON, &rep.OpinionMatrix.BlindSpots); err != nil {
		return nil, fmt.Errorf("unmarshal blind_spots: %w", err)
	}
	if err := json.Unmarshal(recommendedJSON, &rep.RecommendedAgents); err != nil {
		return nil, fmt.Errorf("unmarshal recommended_agents: %w", err)
	}
	return &rep, nil
}
