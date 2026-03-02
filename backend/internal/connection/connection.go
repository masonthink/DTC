// Package connection handles connection requests, confirmations, and
// the privacy-preserving contact exchange mechanism.
package connection

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Status mirrors the DB enum.
type Status string

const (
	StatusPending   Status = "pending"
	StatusAccepted  Status = "accepted"
	StatusRejected  Status = "rejected"
	StatusCancelled Status = "cancelled"
	StatusExpired   Status = "expired"
)

var (
	ErrAlreadyConnected    = errors.New("connection already exists")
	ErrConnectionNotFound  = errors.New("connection not found")
	ErrNotAuthorized       = errors.New("not authorized for this action")
	ErrConnectionExpired   = errors.New("connection request expired")
	ErrContactNotAvailable = errors.New("contact information not available: connection not accepted")
)

// Connection is the core domain object.
type Connection struct {
	ID               string     `json:"id"`
	RequesterUserID  string     `json:"requester_user_id"`
	TargetUserID     string     `json:"target_user_id"`
	RequesterAgentID string     `json:"requester_agent_id"`
	TargetAgentID    string     `json:"target_agent_id"`
	TopicID          string     `json:"topic_id"`
	Status           Status     `json:"status"`
	RequestMessage   string     `json:"request_message"`
	RequesterContact string     `json:"requester_contact,omitempty"` // decrypted, only available after accepted
	TargetContact    string     `json:"target_contact,omitempty"`    // decrypted, only available after accepted
	RequestedAt      time.Time  `json:"requested_at"`
	RespondedAt      *time.Time `json:"responded_at,omitempty"`
	ExpiresAt        time.Time  `json:"expires_at"`
}

// EncryptedContact holds AES-GCM encrypted contact bytes.
type EncryptedContact struct {
	Ciphertext []byte
	IV         []byte
}

// RequestInput is the input for creating a connection request.
type RequestInput struct {
	RequesterUserID  string `json:"-"`
	RequesterAgentID string `json:"requester_agent_id"`
	TargetAgentID    string `json:"target_agent_id"` // 目标 Agent（通过 AnonID 或报告推荐）
	TopicID          string `json:"topic_id,omitempty"`
	RequestMessage   string `json:"request_message,omitempty"`
	RequesterContact string `json:"requester_contact"` // 申请方联系方式（将加密存储）
}

// RespondInput is the input for accepting/rejecting a connection.
type RespondInput struct {
	ConnectionID  string `json:"-"`
	TargetUserID  string `json:"-"`
	Accept        bool   `json:"accept"`
	TargetContact string `json:"target_contact,omitempty"` // 目标方联系方式（仅 Accept 时需要，将加密存储）
}

// Repository abstracts data access for connections.
type Repository interface {
	Create(ctx context.Context, conn *Connection, requesterEnc, targetEnc *EncryptedContact) error
	FindByID(ctx context.Context, id string) (*Connection, error)
	FindByRequester(ctx context.Context, userID string, limit, offset int) ([]*Connection, error)
	FindByTarget(ctx context.Context, userID string, limit, offset int) ([]*Connection, error)
	UpdateStatus(ctx context.Context, id string, status Status, respondedAt *time.Time) error
	StoreTargetContact(ctx context.Context, id string, enc *EncryptedContact) error
	GetEncryptedContacts(ctx context.Context, id string) (requester, target *EncryptedContact, err error)
	LogAnonAccess(ctx context.Context, action, anonID string, connID string, accessedBy string) error
	// ExpirePending bulk-updates all pending connections whose expires_at is in the past.
	// Returns the number of rows updated.
	ExpirePending(ctx context.Context) (int64, error)
}

// AgentRepository is used to look up agent ownership.
type AgentRepository interface {
	FindByID(ctx context.Context, id string) (userID, anonID string, err error)
}

// Service handles connection business logic.
type Service struct {
	repo      Repository
	agentRepo AgentRepository
	encKey    []byte // AES-256 key (32 bytes), managed by Cloud KMS in production
	logger    *zap.Logger
}

// NewService constructs a connection Service.
// encKey must be 32 bytes (AES-256).
func NewService(repo Repository, agentRepo AgentRepository, encKey []byte, logger *zap.Logger) (*Service, error) {
	if len(encKey) != 32 {
		return nil, fmt.Errorf("encKey must be 32 bytes for AES-256, got %d", len(encKey))
	}
	return &Service{
		repo:      repo,
		agentRepo: agentRepo,
		encKey:    encKey,
		logger:    logger,
	}, nil
}

// Request creates a new connection request.
func (s *Service) Request(ctx context.Context, input RequestInput) (*Connection, error) {
	if input.RequesterContact == "" {
		return nil, fmt.Errorf("requester_contact required")
	}

	// Look up target agent's user
	targetUserID, _, err := s.agentRepo.FindByID(ctx, input.TargetAgentID)
	if err != nil {
		return nil, fmt.Errorf("target agent not found: %w", err)
	}
	if targetUserID == input.RequesterUserID {
		return nil, fmt.Errorf("cannot connect to your own agent")
	}

	// Encrypt requester's contact
	requesterEnc, err := s.encrypt(input.RequesterContact)
	if err != nil {
		return nil, fmt.Errorf("encrypt contact: %w", err)
	}

	conn := &Connection{
		ID:               uuid.NewString(),
		RequesterUserID:  input.RequesterUserID,
		TargetUserID:     targetUserID,
		RequesterAgentID: input.RequesterAgentID,
		TargetAgentID:    input.TargetAgentID,
		TopicID:          input.TopicID,
		Status:           StatusPending,
		RequestMessage:   input.RequestMessage,
		RequestedAt:      time.Now(),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.repo.Create(ctx, conn, requesterEnc, nil); err != nil {
		if errors.Is(err, ErrAlreadyConnected) {
			return nil, ErrAlreadyConnected
		}
		return nil, fmt.Errorf("create connection: %w", err)
	}

	s.logger.Info("connection requested",
		zap.String("connection_id", conn.ID),
		zap.String("requester", conn.RequesterUserID),
		zap.String("target", conn.TargetUserID),
	)
	return conn, nil
}

// Respond accepts or rejects a connection request.
func (s *Service) Respond(ctx context.Context, input RespondInput) (*Connection, error) {
	conn, err := s.repo.FindByID(ctx, input.ConnectionID)
	if err != nil || conn == nil {
		return nil, ErrConnectionNotFound
	}
	if conn.TargetUserID != input.TargetUserID {
		return nil, ErrNotAuthorized
	}
	if conn.Status != StatusPending {
		return nil, fmt.Errorf("connection is not in pending status")
	}
	if time.Now().After(conn.ExpiresAt) {
		_ = s.repo.UpdateStatus(ctx, conn.ID, StatusExpired, nil)
		return nil, ErrConnectionExpired
	}

	if !input.Accept {
		now := time.Now()
		if err := s.repo.UpdateStatus(ctx, conn.ID, StatusRejected, &now); err != nil {
			return nil, err
		}
		conn.Status = StatusRejected
		return conn, nil
	}

	// Accept: encrypt target's contact and store, then mark accepted
	if input.TargetContact == "" {
		return nil, fmt.Errorf("target_contact required when accepting")
	}
	targetEnc, err := s.encrypt(input.TargetContact)
	if err != nil {
		return nil, fmt.Errorf("encrypt contact: %w", err)
	}

	if err := s.repo.StoreTargetContact(ctx, conn.ID, targetEnc); err != nil {
		return nil, fmt.Errorf("store target contact: %w", err)
	}

	now := time.Now()
	if err := s.repo.UpdateStatus(ctx, conn.ID, StatusAccepted, &now); err != nil {
		return nil, err
	}
	conn.Status = StatusAccepted
	conn.RespondedAt = &now

	s.logger.Info("connection accepted",
		zap.String("connection_id", conn.ID),
	)

	// Audit log the contact access event
	_ = s.repo.LogAnonAccess(ctx, "connection_accepted", "", conn.ID, input.TargetUserID)

	return conn, nil
}

// GetContacts returns decrypted contact information for both parties.
// Only available when connection status is 'accepted'.
// The requestingUserID must be one of the two parties.
func (s *Service) GetContacts(ctx context.Context, connectionID, requestingUserID string) (requesterContact, targetContact string, err error) {
	conn, err := s.repo.FindByID(ctx, connectionID)
	if err != nil || conn == nil {
		return "", "", ErrConnectionNotFound
	}

	if conn.RequesterUserID != requestingUserID && conn.TargetUserID != requestingUserID {
		return "", "", ErrNotAuthorized
	}
	if conn.Status != StatusAccepted {
		return "", "", ErrContactNotAvailable
	}

	requesterEnc, targetEnc, err := s.repo.GetEncryptedContacts(ctx, connectionID)
	if err != nil {
		return "", "", fmt.Errorf("get encrypted contacts: %w", err)
	}

	// Audit every contact reveal
	_ = s.repo.LogAnonAccess(ctx, "decrypt_contact", "", connectionID, requestingUserID)

	if requesterEnc != nil {
		requesterContact, err = s.decrypt(requesterEnc)
		if err != nil {
			return "", "", fmt.Errorf("decrypt requester contact: %w", err)
		}
	}
	if targetEnc != nil {
		targetContact, err = s.decrypt(targetEnc)
		if err != nil {
			return "", "", fmt.Errorf("decrypt target contact: %w", err)
		}
	}
	return requesterContact, targetContact, nil
}

// =============================================================================
// AES-256-GCM encryption helpers
// =============================================================================

func (s *Service) encrypt(plaintext string) (*EncryptedContact, error) {
	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, iv, []byte(plaintext), nil)
	return &EncryptedContact{
		Ciphertext: ciphertext,
		IV:         iv,
	}, nil
}

func (s *Service) decrypt(enc *EncryptedContact) (string, error) {
	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, enc.IV, enc.Ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}
	return string(plaintext), nil
}

// EncryptToBase64 is a utility for storing encrypted contacts as base64 strings.
func EncryptToBase64(key []byte, plaintext string) (ciphertextB64, ivB64 string, err error) {
	svc := &Service{encKey: key}
	enc, err := svc.encrypt(plaintext)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(enc.Ciphertext),
		base64.StdEncoding.EncodeToString(enc.IV), nil
}
