// Package notificationdb implements notification.NotificationRepository and
// notification.UserRepository using pgx.
package notificationdb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/digital-twin-community/backend/internal/notification"
)

// Repository implements both notification.NotificationRepository and
// notification.UserRepository backed by PostgreSQL.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository constructs a notification Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create persists a new notification and populates n.ID.
func (r *Repository) Create(ctx context.Context, n *notification.Notification) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}

	dataJSON, err := json.Marshal(n.Data)
	if err != nil {
		dataJSON = []byte("{}")
	}

	var topicIDPtr *string
	if n.TopicID != "" {
		topicIDPtr = &n.TopicID
	}

	const q = `
		INSERT INTO notifications
			(id, user_id, topic_id, notification_type, channel, title, body, data, scheduled_at)
		VALUES ($1, $2, $3, $4::notification_type, $5::notification_channel, $6, $7, $8, $9)`
	_, err = r.db.Exec(ctx, q,
		n.ID, n.UserID, topicIDPtr,
		string(n.Type), string(n.Channel),
		n.Title, n.Body, dataJSON, n.ScheduledAt,
	)
	return err
}

// UpdateStatus updates the delivery outcome of a notification.
func (r *Repository) UpdateStatus(ctx context.Context, id, status, externalID, errorMsg string) error {
	if id == "" {
		return nil // notification was not persisted, skip
	}
	const q = `
		UPDATE notifications
		SET status        = $2::notification_status,
		    external_id   = NULLIF($3, ''),
		    error_message = NULLIF($4, ''),
		    sent_at       = CASE WHEN $2 = 'sent' THEN NOW() ELSE sent_at END,
		    updated_at    = NOW()
		WHERE id = $1`
	// notifications table may not have updated_at; omit it gracefully
	const qNoUpdatedAt = `
		UPDATE notifications
		SET status        = $2::notification_status,
		    external_id   = NULLIF($3, ''),
		    error_message = NULLIF($4, ''),
		    sent_at       = CASE WHEN $2 = 'sent' THEN NOW() ELSE sent_at END
		WHERE id = $1`
	_, err := r.db.Exec(ctx, qNoUpdatedAt, id, status, externalID, errorMsg)
	return err
}

// GetFCMToken returns the FCM device token for a user.
func (r *Repository) GetFCMToken(ctx context.Context, userID string) (string, error) {
	const q = `SELECT COALESCE(fcm_token, '') FROM users WHERE id = $1 AND deleted_at IS NULL`
	var token string
	if err := r.db.QueryRow(ctx, q, userID).Scan(&token); err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("get fcm token: %w", err)
	}
	return token, nil
}

// GetEmail returns the email address and display name for a user.
func (r *Repository) GetEmail(ctx context.Context, userID string) (email, displayName string, err error) {
	const q = `
		SELECT COALESCE(email, ''), display_name
		FROM users WHERE id = $1 AND deleted_at IS NULL`
	if err = r.db.QueryRow(ctx, q, userID).Scan(&email, &displayName); err != nil {
		if err == pgx.ErrNoRows {
			return "", "", nil
		}
		return "", "", fmt.Errorf("get email: %w", err)
	}
	return email, displayName, nil
}
