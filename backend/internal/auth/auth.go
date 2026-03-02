// Package auth handles user registration, login, and JWT token management.
package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/digital-twin-community/backend/internal/config"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenInvalid       = errors.New("token invalid")
)

// Claims represents JWT claims.
type Claims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

// TokenPair holds access and refresh tokens.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// LoginRequest for phone/email + password login.
type LoginRequest struct {
	Phone    string `json:"phone,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password"`
}

// RegisterRequest for new user registration.
type RegisterRequest struct {
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// User is a lightweight user DTO (no sensitive fields).
type User struct {
	ID          string    `json:"id"`
	Phone       string    `json:"phone,omitempty"`
	Email       string    `json:"email,omitempty"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserRepository abstracts database operations for auth.
type UserRepository interface {
	FindByPhone(ctx context.Context, phone string) (*UserRecord, error)
	FindByEmail(ctx context.Context, email string) (*UserRecord, error)
	FindByID(ctx context.Context, id string) (*UserRecord, error)
	Create(ctx context.Context, req RegisterRequest, passwordHash string) (*UserRecord, error)
	SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ValidateRefreshToken(ctx context.Context, userID, tokenHash string) (bool, error)
	RevokeRefreshToken(ctx context.Context, userID, tokenHash string) error
	UpdateFCMToken(ctx context.Context, userID, token string) error
}

// UserRecord is the full DB record including password hash.
type UserRecord struct {
	ID           string
	Phone        string
	Email        string
	DisplayName  string
	PasswordHash string
	Status       string
	CreatedAt    time.Time
}

// Service handles auth business logic.
type Service struct {
	repo   UserRepository
	cfg    *config.JWTConfig
	logger *zap.Logger
}

// NewService constructs an auth Service.
func NewService(repo UserRepository, cfg *config.JWTConfig, logger *zap.Logger) *Service {
	return &Service{repo: repo, cfg: cfg, logger: logger}
}

// Register creates a new user account.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*User, *TokenPair, error) {
	if req.Phone == "" && req.Email == "" {
		return nil, nil, fmt.Errorf("phone or email required")
	}
	if len(req.Password) < 8 {
		return nil, nil, fmt.Errorf("password must be at least 8 characters")
	}
	if !hasLetter(req.Password) || !hasDigit(req.Password) {
		return nil, nil, fmt.Errorf("password must contain at least one letter and one digit")
	}
	if req.DisplayName == "" {
		return nil, nil, fmt.Errorf("display_name required")
	}
	if len(req.DisplayName) > 100 {
		return nil, nil, fmt.Errorf("display_name too long (max 100 chars)")
	}

	// Check for existing user
	if req.Phone != "" {
		if existing, _ := s.repo.FindByPhone(ctx, req.Phone); existing != nil {
			return nil, nil, ErrUserAlreadyExists
		}
	}
	if req.Email != "" {
		if existing, _ := s.repo.FindByEmail(ctx, req.Email); existing != nil {
			return nil, nil, ErrUserAlreadyExists
		}
	}

	cost := s.cfg.BcryptCost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), cost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	record, err := s.repo.Create(ctx, req, string(hash))
	if err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	tokens, err := s.generateTokenPair(ctx, record.ID)
	if err != nil {
		return nil, nil, err
	}

	s.logger.Info("user registered", zap.String("user_id", record.ID))
	return recordToUser(record), tokens, nil
}

// Login authenticates a user and returns token pair.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*User, *TokenPair, error) {
	var record *UserRecord
	var err error

	if req.Phone != "" {
		record, err = s.repo.FindByPhone(ctx, req.Phone)
	} else if req.Email != "" {
		record, err = s.repo.FindByEmail(ctx, req.Email)
	} else {
		return nil, nil, fmt.Errorf("phone or email required")
	}

	if err != nil || record == nil {
		return nil, nil, ErrInvalidCredentials
	}
	if record.Status != "active" {
		return nil, nil, fmt.Errorf("account suspended")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte(req.Password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.generateTokenPair(ctx, record.ID)
	if err != nil {
		return nil, nil, err
	}

	s.logger.Info("user logged in", zap.String("user_id", record.ID))
	return recordToUser(record), tokens, nil
}

// RefreshTokens issues a new token pair given a valid refresh token.
func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	tokenHash := hashToken(refreshToken)
	valid, err := s.repo.ValidateRefreshToken(ctx, claims.UserID, tokenHash)
	if err != nil || !valid {
		return nil, ErrTokenInvalid
	}

	// Rotate: revoke old, issue new
	_ = s.repo.RevokeRefreshToken(ctx, claims.UserID, tokenHash)
	return s.generateTokenPair(ctx, claims.UserID)
}

// GetMe returns the profile of the authenticated user.
func (s *Service) GetMe(ctx context.Context, userID string) (*User, error) {
	record, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if record == nil {
		return nil, ErrUserNotFound
	}
	return recordToUser(record), nil
}

// UpdateFCMToken saves a device FCM registration token for push notifications.
func (s *Service) UpdateFCMToken(ctx context.Context, userID, token string) error {
	return s.repo.UpdateFCMToken(ctx, userID, token)
}

// ValidateAccessToken parses and validates an access token, returning the user ID.
func (s *Service) ValidateAccessToken(tokenStr string) (string, error) {
	claims, err := s.parseToken(tokenStr)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// generateTokenPair creates a new access + refresh token pair.
func (s *Service) generateTokenPair(ctx context.Context, userID string) (*TokenPair, error) {
	expiresAt := time.Now().Add(s.cfg.AccessTokenTTL)

	accessClaims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshClaims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	// Persist refresh token hash
	tokenHash := hashToken(refreshToken)
	if err := s.repo.SaveRefreshToken(ctx, userID, tokenHash, time.Now().Add(s.cfg.RefreshTokenTTL)); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Service) parseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func recordToUser(r *UserRecord) *User {
	return &User{
		ID:          r.ID,
		Phone:       r.Phone,
		Email:       r.Email,
		DisplayName: r.DisplayName,
		CreatedAt:   r.CreatedAt,
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", sum[:])
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func hasDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
