// Package connectiondb implements connection.Repository and connection.AgentRepository using pgx.
package connectiondb

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/digital-twin-community/backend/internal/connection"
)

// Repository implements connection.Repository backed by PostgreSQL.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository constructs a connection Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create inserts a new connection request with the requester's encrypted contact.
func (r *Repository) Create(
	ctx context.Context,
	conn *connection.Connection,
	requesterEnc, targetEnc *connection.EncryptedContact,
) error {
	var reqCiphertext, reqIV, tgtCiphertext, tgtIV []byte
	if requesterEnc != nil {
		reqCiphertext = requesterEnc.Ciphertext
		reqIV = requesterEnc.IV
	}
	if targetEnc != nil {
		tgtCiphertext = targetEnc.Ciphertext
		tgtIV = targetEnc.IV
	}
	const q = `
		INSERT INTO connections
			(id, requester_user_id, target_user_id,
			 requester_agent_id, target_agent_id, topic_id,
			 status, request_message,
			 requester_contact_enc, requester_contact_iv,
			 target_contact_enc, target_contact_iv,
			 requested_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7::connection_status, $8,
		        $9, $10, $11, $12, $13, $14)`
	_, err := r.db.Exec(ctx, q,
		conn.ID, conn.RequesterUserID, conn.TargetUserID,
		conn.RequesterAgentID, conn.TargetAgentID, conn.TopicID,
		string(conn.Status), conn.RequestMessage,
		reqCiphertext, reqIV,
		tgtCiphertext, tgtIV,
		conn.RequestedAt, conn.ExpiresAt,
	)
	return err
}

// FindByID retrieves a connection by primary key.
func (r *Repository) FindByID(ctx context.Context, id string) (*connection.Connection, error) {
	const q = connSelectCols + ` WHERE id = $1`
	return r.scanOne(r.db.QueryRow(ctx, q, id))
}

// FindByRequester returns connections initiated by a user.
func (r *Repository) FindByRequester(ctx context.Context, userID string, limit, offset int) ([]*connection.Connection, error) {
	const q = connSelectCols + `
		WHERE requester_user_id = $1
		ORDER BY requested_at DESC
		LIMIT $2 OFFSET $3`
	return r.queryMany(ctx, q, userID, limit, offset)
}

// FindByTarget returns connection requests targeting a user.
func (r *Repository) FindByTarget(ctx context.Context, userID string, limit, offset int) ([]*connection.Connection, error) {
	const q = connSelectCols + `
		WHERE target_user_id = $1
		ORDER BY requested_at DESC
		LIMIT $2 OFFSET $3`
	return r.queryMany(ctx, q, userID, limit, offset)
}

// UpdateStatus updates the connection status and optionally sets respondedAt.
func (r *Repository) UpdateStatus(ctx context.Context, id string, status connection.Status, respondedAt *time.Time) error {
	const q = `
		UPDATE connections
		SET status = $2::connection_status, responded_at = $3, updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, string(status), respondedAt)
	return err
}

// StoreTargetContact persists the target's encrypted contact information.
func (r *Repository) StoreTargetContact(ctx context.Context, id string, enc *connection.EncryptedContact) error {
	const q = `
		UPDATE connections
		SET target_contact_enc = $2, target_contact_iv = $3, updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, enc.Ciphertext, enc.IV)
	return err
}

// GetEncryptedContacts retrieves both parties' encrypted contact bytes.
func (r *Repository) GetEncryptedContacts(ctx context.Context, id string) (requester, target *connection.EncryptedContact, err error) {
	const q = `
		SELECT requester_contact_enc, requester_contact_iv,
		       target_contact_enc, target_contact_iv
		FROM connections WHERE id = $1`
	var reqCipher, reqIV, tgtCipher, tgtIV []byte
	if err = r.db.QueryRow(ctx, q, id).Scan(&reqCipher, &reqIV, &tgtCipher, &tgtIV); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, connection.ErrConnectionNotFound
		}
		return nil, nil, fmt.Errorf("get encrypted contacts: %w", err)
	}
	if reqCipher != nil {
		requester = &connection.EncryptedContact{Ciphertext: reqCipher, IV: reqIV}
	}
	if tgtCipher != nil {
		target = &connection.EncryptedContact{Ciphertext: tgtCipher, IV: tgtIV}
	}
	return requester, target, nil
}

// ExpirePending bulk-expires all pending connections whose expires_at < NOW().
func (r *Repository) ExpirePending(ctx context.Context) (int64, error) {
	const q = `
		UPDATE connections
		SET status = 'expired'::connection_status, updated_at = NOW()
		WHERE status = 'pending' AND expires_at < NOW()`
	tag, err := r.db.Exec(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("expire pending connections: %w", err)
	}
	return tag.RowsAffected(), nil
}

// LogAnonAccess inserts a row into the anonymized audit log.
func (r *Repository) LogAnonAccess(ctx context.Context, action, anonID string, connID string, accessedBy string) error {
	var anonIDPtr *string
	if anonID != "" {
		anonIDPtr = &anonID
	}
	var connIDPtr *string
	if connID != "" {
		connIDPtr = &connID
	}
	const q = `
		INSERT INTO anon.access_audit_log (accessed_by, action, anon_id, connection_id)
		VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(ctx, q, accessedBy, action, anonIDPtr, connIDPtr)
	return err
}

// connSelectCols is the base SELECT clause for connections.
const connSelectCols = `
	SELECT id,
	       requester_user_id::text, target_user_id::text,
	       requester_agent_id::text, target_agent_id::text,
	       COALESCE(topic_id::text, '') AS topic_id,
	       status::text, request_message,
	       requested_at, responded_at, expires_at
	FROM connections`

func (r *Repository) scanOne(row pgx.Row) (*connection.Connection, error) {
	conn, err := scanConnRow(row.Scan)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (r *Repository) queryMany(ctx context.Context, q string, args ...interface{}) ([]*connection.Connection, error) {
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []*connection.Connection
	for rows.Next() {
		conn, err := scanConnRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		conns = append(conns, conn)
	}
	return conns, rows.Err()
}

func scanConnRow(scan func(dest ...any) error) (*connection.Connection, error) {
	var conn connection.Connection
	var status string
	var respondedAt *time.Time

	err := scan(
		&conn.ID,
		&conn.RequesterUserID, &conn.TargetUserID,
		&conn.RequesterAgentID, &conn.TargetAgentID,
		&conn.TopicID,
		&status, &conn.RequestMessage,
		&conn.RequestedAt, &respondedAt, &conn.ExpiresAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan connection: %w", err)
	}
	conn.Status = connection.Status(status)
	conn.RespondedAt = respondedAt
	return &conn, nil
}

// =============================================================================
// AgentRepository implements connection.AgentRepository
// =============================================================================

// AgentRepository provides agent lookups for the connection service.
type AgentRepository struct {
	db *pgxpool.Pool
}

// NewAgentRepository constructs a connection AgentRepository.
func NewAgentRepository(db *pgxpool.Pool) *AgentRepository {
	return &AgentRepository{db: db}
}

// FindByID returns the owning user ID and anonymous ID for an agent.
func (r *AgentRepository) FindByID(ctx context.Context, id string) (userID, anonID string, err error) {
	const q = `SELECT user_id::text, anon_id FROM agents WHERE id = $1`
	if err = r.db.QueryRow(ctx, q, id).Scan(&userID, &anonID); err != nil {
		if err == pgx.ErrNoRows {
			return "", "", fmt.Errorf("agent not found: %s", id)
		}
		return "", "", fmt.Errorf("find agent: %w", err)
	}
	return userID, anonID, nil
}
