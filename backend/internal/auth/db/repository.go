// Package authdb implements the auth.UserRepository interface using pgx.
package authdb

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/digital-twin-community/backend/internal/auth"
)

// Repository implements auth.UserRepository backed by PostgreSQL.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository constructs an auth Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// FindByPhone looks up a user by phone number.
func (r *Repository) FindByPhone(ctx context.Context, phone string) (*auth.UserRecord, error) {
	const q = `
		SELECT id, phone, email, display_name, password_hash, status, created_at
		FROM users
		WHERE phone = $1 AND deleted_at IS NULL`
	return r.scanUser(r.db.QueryRow(ctx, q, phone))
}

// FindByEmail looks up a user by email address.
func (r *Repository) FindByEmail(ctx context.Context, email string) (*auth.UserRecord, error) {
	const q = `
		SELECT id, phone, email, display_name, password_hash, status, created_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`
	return r.scanUser(r.db.QueryRow(ctx, q, email))
}

// FindByID looks up a user by primary key.
func (r *Repository) FindByID(ctx context.Context, id string) (*auth.UserRecord, error) {
	const q = `
		SELECT id, phone, email, display_name, password_hash, status, created_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`
	return r.scanUser(r.db.QueryRow(ctx, q, id))
}

// Create inserts a new user and returns the full record.
func (r *Repository) Create(ctx context.Context, req auth.RegisterRequest, passwordHash string) (*auth.UserRecord, error) {
	var phone, email *string
	if req.Phone != "" {
		phone = &req.Phone
	}
	if req.Email != "" {
		email = &req.Email
	}
	const q = `
		INSERT INTO users (phone, email, display_name, password_hash, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id, phone, email, display_name, password_hash, status, created_at`
	return r.scanUser(r.db.QueryRow(ctx, q, phone, email, req.DisplayName, passwordHash))
}

// SaveRefreshToken persists a hashed refresh token for the given user.
func (r *Repository) SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`
	_, err := r.db.Exec(ctx, q, userID, tokenHash, expiresAt)
	return err
}

// ValidateRefreshToken checks whether a token hash is valid (not revoked, not expired).
func (r *Repository) ValidateRefreshToken(ctx context.Context, userID, tokenHash string) (bool, error) {
	const q = `
		SELECT COUNT(1) FROM refresh_tokens
		WHERE user_id = $1
		  AND token_hash = $2
		  AND revoked_at IS NULL
		  AND expires_at > NOW()`
	var count int
	if err := r.db.QueryRow(ctx, q, userID, tokenHash).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// RevokeRefreshToken marks a token hash as revoked.
func (r *Repository) RevokeRefreshToken(ctx context.Context, userID, tokenHash string) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND token_hash = $2 AND revoked_at IS NULL`
	_, err := r.db.Exec(ctx, q, userID, tokenHash)
	return err
}

// UpdateFCMToken saves the Firebase Cloud Messaging device token for the given user.
func (r *Repository) UpdateFCMToken(ctx context.Context, userID, token string) error {
	const q = `UPDATE users SET fcm_token = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, q, token, userID)
	return err
}

// scanUser reads a single UserRecord from a pgx.Row.
func (r *Repository) scanUser(row pgx.Row) (*auth.UserRecord, error) {
	var rec auth.UserRecord
	var phone, email *string
	err := row.Scan(
		&rec.ID,
		&phone,
		&email,
		&rec.DisplayName,
		&rec.PasswordHash,
		&rec.Status,
		&rec.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	if phone != nil {
		rec.Phone = *phone
	}
	if email != nil {
		rec.Email = *email
	}
	return &rec, nil
}
