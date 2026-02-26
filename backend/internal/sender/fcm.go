// Package sender provides concrete implementations of notification.FCMSender
// and notification.EmailSender.
package sender

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

const (
	fcmScope    = "https://www.googleapis.com/auth/firebase.messaging"
	fcmEndpointFmt = "https://fcm.googleapis.com/v1/projects/%s/messages:send"
	defaultTokenURI = "https://oauth2.googleapis.com/token"
)

// serviceAccountKey mirrors the fields needed from a GCP service account JSON file.
type serviceAccountKey struct {
	Type         string `json:"type"`
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	TokenURI     string `json:"token_uri"`
}

// FCMClient sends push notifications via the Firebase Cloud Messaging HTTP v1 API.
// Auth is handled via service account JWT → OAuth2 token exchange (no Firebase SDK needed).
type FCMClient struct {
	sa          serviceAccountKey
	privateKey  *rsa.PrivateKey
	httpClient  *http.Client
	logger      *zap.Logger
	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

// NewFCMClient loads a service account JSON file and constructs an FCMClient.
// Returns an error if the file is missing or malformed.
func NewFCMClient(credentialsFile string, logger *zap.Logger) (*FCMClient, error) {
	if credentialsFile == "" {
		return nil, fmt.Errorf("FCM credentials file path is empty")
	}
	data, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("read FCM credentials file: %w", err)
	}

	var sa serviceAccountKey
	if err := json.Unmarshal(data, &sa); err != nil {
		return nil, fmt.Errorf("parse FCM credentials JSON: %w", err)
	}
	if sa.Type != "service_account" {
		return nil, fmt.Errorf("invalid credentials type %q (expected \"service_account\")", sa.Type)
	}
	if sa.TokenURI == "" {
		sa.TokenURI = defaultTokenURI
	}

	privateKey, err := parseRSAPrivateKey(sa.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse FCM private key: %w", err)
	}

	return &FCMClient{
		sa:         sa,
		privateKey: privateKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}, nil
}

// accessToken returns a valid OAuth2 access token, refreshing if necessary.
func (c *FCMClient) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.cachedToken, nil
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   c.sa.ClientEmail,
		"scope": fcmScope,
		"aud":   c.sa.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = c.sa.PrivateKeyID

	signedJWT, err := tok.SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign service account JWT: %w", err)
	}

	resp, err := http.PostForm(c.sa.TokenURI, url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {signedJWT},
	})
	if err != nil {
		return "", fmt.Errorf("exchange JWT for token: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("token exchange: %s – %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	// Cache with 60-second safety margin
	c.cachedToken = tokenResp.AccessToken
	c.tokenExpiry = now.Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	return c.cachedToken, nil
}

// sendMessage posts a single FCM message and returns the server-assigned message name.
func (c *FCMClient) sendMessage(ctx context.Context, msg map[string]interface{}) (string, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("get FCM access token: %w", err)
	}

	payload, err := json.Marshal(map[string]interface{}{"message": msg})
	if err != nil {
		return "", fmt.Errorf("marshal FCM payload: %w", err)
	}

	endpoint := fmt.Sprintf(fcmEndpointFmt, c.sa.ProjectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build FCM request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("FCM HTTP request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Name  string `json:"name"` // projects/{project}/messages/{id}
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse FCM response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FCM error %d (%s): %s",
			result.Error.Code, result.Error.Status, result.Error.Message)
	}
	return result.Name, nil
}

// SendToDevice sends a push notification to a specific device token.
func (c *FCMClient) SendToDevice(ctx context.Context, fcmToken, title, body string, data map[string]string) (string, error) {
	strData := make(map[string]interface{}, len(data))
	for k, v := range data {
		strData[k] = v
	}
	msg := map[string]interface{}{
		"token": fcmToken,
		"notification": map[string]string{
			"title": title,
			"body":  body,
		},
		"data": strData,
	}
	return c.sendMessage(ctx, msg)
}

// SendToTopic sends a push notification to an FCM topic.
func (c *FCMClient) SendToTopic(ctx context.Context, topic, title, body string, data map[string]string) (string, error) {
	topicName := topic
	if !strings.HasPrefix(topicName, "/topics/") {
		topicName = "/topics/" + topicName
	}
	strData := make(map[string]interface{}, len(data))
	for k, v := range data {
		strData[k] = v
	}
	msg := map[string]interface{}{
		"topic": topicName,
		"notification": map[string]string{
			"title": title,
			"body":  body,
		},
		"data": strData,
	}
	return c.sendMessage(ctx, msg)
}

// parseRSAPrivateKey parses a PEM-encoded RSA private key (PKCS8 or PKCS1).
func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from private key")
	}

	// Prefer PKCS8 (GCP service accounts use PKCS8)
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS8 key is not RSA")
		}
		return rsaKey, nil
	}

	// Fallback to PKCS1
	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS1 private key: %w", err)
	}
	return rsaKey, nil
}

// ---------------------------------------------------------------------------
// No-op sender (used when FCM is not configured in dev)
// ---------------------------------------------------------------------------

// NoOpFCMSender satisfies notification.FCMSender but never sends anything.
type NoOpFCMSender struct {
	logger *zap.Logger
}

// NewNoOpFCMSender returns a sender that logs and discards all push notifications.
func NewNoOpFCMSender(logger *zap.Logger) *NoOpFCMSender {
	return &NoOpFCMSender{logger: logger}
}

func (n *NoOpFCMSender) SendToDevice(_ context.Context, fcmToken, title, _ string, _ map[string]string) (string, error) {
	n.logger.Debug("FCM noop: SendToDevice",
		zap.String("title", title),
		zap.String("token_prefix", safePrefix(fcmToken, 8)),
	)
	return "noop", nil
}

func (n *NoOpFCMSender) SendToTopic(_ context.Context, topic, title, _ string, _ map[string]string) (string, error) {
	n.logger.Debug("FCM noop: SendToTopic",
		zap.String("topic", topic),
		zap.String("title", title),
	)
	return "noop", nil
}

func safePrefix(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
