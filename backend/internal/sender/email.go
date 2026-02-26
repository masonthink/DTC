package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const sendGridEndpoint = "https://api.sendgrid.com/v3/mail/send"

// SendGridClient sends transactional email via the SendGrid Web API v3.
type SendGridClient struct {
	apiKey     string
	fromEmail  string
	fromName   string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewSendGridClient constructs a SendGridClient.
// apiKey may be empty; calls will return an error in that case.
func NewSendGridClient(apiKey, fromEmail, fromName string, logger *zap.Logger) *SendGridClient {
	return &SendGridClient{
		apiKey:     apiKey,
		fromEmail:  fromEmail,
		fromName:   fromName,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		logger:     logger,
	}
}

// SendEmail sends an HTML email and returns the SendGrid X-Message-Id on success.
func (c *SendGridClient) SendEmail(ctx context.Context, toEmail, toName, subject, htmlContent string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("sendgrid: api key not configured")
	}

	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": []map[string]string{
					{"email": toEmail, "name": toName},
				},
				"subject": subject,
			},
		},
		"from": map[string]string{
			"email": c.fromEmail,
			"name":  c.fromName,
		},
		"content": []map[string]string{
			{"type": "text/html", "value": htmlContent},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal email payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendGridEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build sendgrid request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sendgrid HTTP request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	// SendGrid returns HTTP 202 with no body on success.
	if resp.StatusCode == http.StatusAccepted {
		msgID := resp.Header.Get("X-Message-Id")
		c.logger.Debug("email sent via sendgrid",
			zap.String("to", toEmail),
			zap.String("subject", subject),
			zap.String("message_id", msgID),
		)
		return msgID, nil
	}

	return "", fmt.Errorf("sendgrid error %d: %s", resp.StatusCode, string(respBody))
}
