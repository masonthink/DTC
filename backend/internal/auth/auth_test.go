package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/digital-twin-community/backend/internal/auth"
	"github.com/digital-twin-community/backend/internal/config"
)

// ── mock repository ───────────────────────────────────────────────────────────

type mockRepo struct {
	users   map[string]*auth.UserRecord // keyed by ID
	byPhone map[string]*auth.UserRecord
	byEmail map[string]*auth.UserRecord
	tokens  map[string]mockToken // keyed by tokenHash
}

type mockToken struct {
	userID    string
	expiresAt time.Time
	revoked   bool
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		users:   make(map[string]*auth.UserRecord),
		byPhone: make(map[string]*auth.UserRecord),
		byEmail: make(map[string]*auth.UserRecord),
		tokens:  make(map[string]mockToken),
	}
}

func (m *mockRepo) FindByPhone(_ context.Context, phone string) (*auth.UserRecord, error) {
	return m.byPhone[phone], nil
}
func (m *mockRepo) FindByEmail(_ context.Context, email string) (*auth.UserRecord, error) {
	return m.byEmail[email], nil
}
func (m *mockRepo) FindByID(_ context.Context, id string) (*auth.UserRecord, error) {
	return m.users[id], nil
}
func (m *mockRepo) Create(_ context.Context, req auth.RegisterRequest, hash string) (*auth.UserRecord, error) {
	rec := &auth.UserRecord{
		ID:           uuid.NewString(),
		Phone:        req.Phone,
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		PasswordHash: hash,
		Status:       "active",
		CreatedAt:    time.Now(),
	}
	m.users[rec.ID] = rec
	if req.Phone != "" {
		m.byPhone[req.Phone] = rec
	}
	if req.Email != "" {
		m.byEmail[req.Email] = rec
	}
	return rec, nil
}
func (m *mockRepo) SaveRefreshToken(_ context.Context, userID, tokenHash string, expiresAt time.Time) error {
	m.tokens[tokenHash] = mockToken{userID: userID, expiresAt: expiresAt}
	return nil
}
func (m *mockRepo) ValidateRefreshToken(_ context.Context, userID, tokenHash string) (bool, error) {
	t, ok := m.tokens[tokenHash]
	if !ok || t.revoked || t.expiresAt.Before(time.Now()) {
		return false, nil
	}
	return t.userID == userID, nil
}
func (m *mockRepo) RevokeRefreshToken(_ context.Context, _, tokenHash string) error {
	if t, ok := m.tokens[tokenHash]; ok {
		t.revoked = true
		m.tokens[tokenHash] = t
	}
	return nil
}
func (m *mockRepo) UpdateFCMToken(_ context.Context, _, _ string) error { return nil }

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestService() (*auth.Service, *mockRepo) {
	repo := newMockRepo()
	cfg := &config.JWTConfig{
		Secret:          "test-secret-must-be-at-least-32-chars!",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		BcryptCost:      bcrypt.MinCost,
	}
	svc := auth.NewService(repo, cfg, zap.NewNop())
	return svc, repo
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	user, tokens, err := svc.Register(ctx, auth.RegisterRequest{
		Phone:       "+8613800138000",
		Password:    "password123",
		DisplayName: "测试用户",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
	if user.Phone != "+8613800138000" {
		t.Errorf("got phone %q, want +8613800138000", user.Phone)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Error("expected non-empty tokens")
	}
}

func TestRegister_DuplicatePhone(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	req := auth.RegisterRequest{
		Phone:       "+8613800138000",
		Password:    "password123",
		DisplayName: "用户A",
	}
	if _, _, err := svc.Register(ctx, req); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	req.DisplayName = "用户B"
	_, _, err := svc.Register(ctx, req)
	if err != auth.ErrUserAlreadyExists {
		t.Errorf("got %v, want ErrUserAlreadyExists", err)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	svc, _ := newTestService()
	_, _, err := svc.Register(context.Background(), auth.RegisterRequest{
		Email:       "a@example.com",
		Password:    "short",
		DisplayName: "用户",
	})
	if err == nil {
		t.Error("expected error for short password")
	}
}

func TestRegister_MissingContact(t *testing.T) {
	svc, _ := newTestService()
	_, _, err := svc.Register(context.Background(), auth.RegisterRequest{
		Password:    "password123",
		DisplayName: "用户",
	})
	if err == nil {
		t.Error("expected error when phone and email both empty")
	}
}

func TestLogin_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	if _, _, err := svc.Register(ctx, auth.RegisterRequest{
		Phone:       "+8613800138000",
		Password:    "password123",
		DisplayName: "用户",
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	user, tokens, err := svc.Login(ctx, auth.LoginRequest{
		Phone:    "+8613800138000",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if user == nil || tokens == nil {
		t.Fatal("expected non-nil user and tokens")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	if _, _, err := svc.Register(ctx, auth.RegisterRequest{
		Phone:       "+8613800138000",
		Password:    "correctpassword",
		DisplayName: "用户",
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, _, err := svc.Login(ctx, auth.LoginRequest{
		Phone:    "+8613800138000",
		Password: "wrongpassword",
	})
	if err != auth.ErrInvalidCredentials {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_UnknownPhone(t *testing.T) {
	svc, _ := newTestService()
	_, _, err := svc.Login(context.Background(), auth.LoginRequest{
		Phone:    "+8600000000000",
		Password: "password123",
	})
	if err != auth.ErrInvalidCredentials {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestRefreshTokens_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, tokens, err := svc.Register(ctx, auth.RegisterRequest{
		Phone:       "+8613800138000",
		Password:    "password123",
		DisplayName: "用户",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	newTokens, err := svc.RefreshTokens(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshTokens failed: %v", err)
	}
	if newTokens.AccessToken == tokens.AccessToken {
		t.Error("expected new access token to differ from old one")
	}
}

func TestRefreshTokens_Replay(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, tokens, _ := svc.Register(ctx, auth.RegisterRequest{
		Phone:       "+8613800138000",
		Password:    "password123",
		DisplayName: "用户",
	})

	// First refresh succeeds
	if _, err := svc.RefreshTokens(ctx, tokens.RefreshToken); err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	// Replay the same refresh token — must fail (revoked)
	_, err := svc.RefreshTokens(ctx, tokens.RefreshToken)
	if err == nil {
		t.Error("expected error on refresh token replay")
	}
}

func TestValidateAccessToken(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	user, tokens, _ := svc.Register(ctx, auth.RegisterRequest{
		Phone:       "+8613800138000",
		Password:    "password123",
		DisplayName: "用户",
	})

	userID, err := svc.ValidateAccessToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if userID != user.ID {
		t.Errorf("got userID %q, want %q", userID, user.ID)
	}
}

func TestGetMe_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	user, _, _ := svc.Register(ctx, auth.RegisterRequest{
		Email:       "a@example.com",
		Password:    "password123",
		DisplayName: "测试",
	})

	me, err := svc.GetMe(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	if me.ID != user.ID {
		t.Errorf("got ID %q, want %q", me.ID, user.ID)
	}
	if me.Email != "a@example.com" {
		t.Errorf("got email %q, want a@example.com", me.Email)
	}
}

func TestGetMe_NotFound(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.GetMe(context.Background(), "nonexistent-id")
	if err != auth.ErrUserNotFound {
		t.Errorf("got %v, want ErrUserNotFound", err)
	}
}
