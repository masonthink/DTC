// Package agentdb implements agent.Repository using pgx.
package agentdb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/digital-twin-community/backend/internal/agent"
)

// Repository implements agent.Repository backed by PostgreSQL.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository constructs an agent Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create inserts a new agent.
func (r *Repository) Create(ctx context.Context, a *agent.Agent) error {
	questJSON, err := json.Marshal(a.Questionnaire)
	if err != nil {
		return fmt.Errorf("marshal questionnaire: %w", err)
	}
	tsJSON, err := json.Marshal(a.ThinkingStyle)
	if err != nil {
		return fmt.Errorf("marshal thinking_style: %w", err)
	}
	const q = `
		INSERT INTO agents
			(id, user_id, agent_type, display_name, questionnaire, industries, skills,
			 thinking_style, experience_years, anon_id, quality_score, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	_, err = r.db.Exec(ctx, q,
		a.ID, a.UserID, string(a.AgentType), a.DisplayName,
		questJSON, a.Industries, a.Skills,
		tsJSON, a.ExperienceYears, a.AnonID,
		a.QualityScore, a.IsActive, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

// FindByID retrieves an agent by primary key.
func (r *Repository) FindByID(ctx context.Context, id string) (*agent.Agent, error) {
	const q = agentSelectColumns + ` WHERE id = $1`
	return r.scanOne(r.db.QueryRow(ctx, q, id))
}

// FindByUserID retrieves all agents for a user.
func (r *Repository) FindByUserID(ctx context.Context, userID string) ([]*agent.Agent, error) {
	const q = agentSelectColumns + ` WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.collectRows(rows)
}

// FindByAnonID retrieves an agent by anonymous ID.
func (r *Repository) FindByAnonID(ctx context.Context, anonID string) (*agent.Agent, error) {
	const q = agentSelectColumns + ` WHERE anon_id = $1`
	return r.scanOne(r.db.QueryRow(ctx, q, anonID))
}

// Update applies partial updates to an agent.
func (r *Repository) Update(ctx context.Context, id string, req agent.UpdateRequest) (*agent.Agent, error) {
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{id}
	argIdx := 2

	if req.DisplayName != nil {
		setClauses = append(setClauses, fmt.Sprintf("display_name = $%d", argIdx))
		args = append(args, *req.DisplayName)
		argIdx++
	}
	if req.AgentType != nil {
		setClauses = append(setClauses, fmt.Sprintf("agent_type = $%d", argIdx))
		args = append(args, string(*req.AgentType))
		argIdx++
	}
	if req.Questionnaire != nil {
		qJSON, err := json.Marshal(*req.Questionnaire)
		if err != nil {
			return nil, fmt.Errorf("marshal questionnaire: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("questionnaire = $%d", argIdx))
		args = append(args, qJSON)
		argIdx++
	}
	_ = argIdx
	q := fmt.Sprintf("UPDATE agents SET %s WHERE id = $1", strings.Join(setClauses, ", "))
	if _, err := r.db.Exec(ctx, q, args...); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

// UpdateEmbedding stores the Qdrant point ID and refreshes the embedding timestamp.
func (r *Repository) UpdateEmbedding(ctx context.Context, id, qdrantPointID string) error {
	const q = `
		UPDATE agents
		SET qdrant_point_id = $2, embedding_updated_at = NOW(), updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, qdrantPointID)
	return err
}

// UpdateQualityScore updates the quality score of an agent.
func (r *Repository) UpdateQualityScore(ctx context.Context, id string, score float64) error {
	const q = `UPDATE agents SET quality_score = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, score)
	return err
}

// IncrementDiscussionCount atomically increments the discussion count.
func (r *Repository) IncrementDiscussionCount(ctx context.Context, id string) error {
	const q = `UPDATE agents SET discussion_count = discussion_count + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

// SetActive enables or disables an agent.
func (r *Repository) SetActive(ctx context.Context, id string, active bool) error {
	const q = `UPDATE agents SET is_active = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, active)
	return err
}

// FindActiveAgents returns paginated active agents sorted by last activity.
func (r *Repository) FindActiveAgents(ctx context.Context, limit, offset int) ([]*agent.Agent, error) {
	const q = agentSelectColumns + `
		WHERE is_active = true
		ORDER BY last_active_at DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.collectRows(rows)
}

// FindAgentsForEmbedding returns agents whose embeddings are missing or stale.
func (r *Repository) FindAgentsForEmbedding(ctx context.Context, limit int) ([]*agent.Agent, error) {
	const q = agentSelectColumns + `
		WHERE qdrant_point_id IS NULL OR embedding_updated_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1`
	rows, err := r.db.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.collectRows(rows)
}

// agentSelectColumns is the base SELECT for all agent queries.
const agentSelectColumns = `
	SELECT id, user_id, agent_type, display_name, questionnaire, industries, skills,
	       thinking_style, experience_years, anon_id,
	       COALESCE(qdrant_point_id::text, '') AS qdrant_point_id,
	       quality_score, discussion_count, connection_count,
	       is_active, last_active_at, created_at, updated_at
	FROM agents`

func (r *Repository) scanOne(row pgx.Row) (*agent.Agent, error) {
	a, err := scanAgentRow(row.Scan)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *Repository) collectRows(rows pgx.Rows) ([]*agent.Agent, error) {
	var agents []*agent.Agent
	for rows.Next() {
		a, err := scanAgentRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// scanAgentRow scans one row using the provided Scan function (supports both pgx.Row and pgx.Rows).
func scanAgentRow(scan func(dest ...any) error) (*agent.Agent, error) {
	var a agent.Agent
	var agentType string
	var questJSON, tsJSON []byte
	var qdrantID string
	var lastActiveAt *time.Time

	err := scan(
		&a.ID, &a.UserID, &agentType, &a.DisplayName,
		&questJSON, &a.Industries, &a.Skills,
		&tsJSON, &a.ExperienceYears, &a.AnonID, &qdrantID,
		&a.QualityScore, &a.DiscussionCount, &a.ConnectionCount,
		&a.IsActive, &lastActiveAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan agent: %w", err)
	}

	a.AgentType = agent.AgentType(agentType)
	a.QdrantPointID = qdrantID
	if lastActiveAt != nil {
		a.LastActiveAt = *lastActiveAt
	}

	if err := json.Unmarshal(questJSON, &a.Questionnaire); err != nil {
		return nil, fmt.Errorf("unmarshal questionnaire: %w", err)
	}
	if err := json.Unmarshal(tsJSON, &a.ThinkingStyle); err != nil {
		return nil, fmt.Errorf("unmarshal thinking_style: %w", err)
	}
	return &a, nil
}
