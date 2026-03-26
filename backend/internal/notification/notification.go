// Package notification handles multi-channel push notifications
// via Firebase Cloud Messaging and SendGrid email.
package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Type mirrors the DB enum.
type Type string

const (
	TypeMatchPreview      Type = "match_preview"       // T+1h 匹配预告
	TypeDiscussionUpdate  Type = "discussion_update"   // T+12h 讨论快报
	TypeReportReady       Type = "report_ready"        // T+48h 完整报告
	TypeConnectionRequest Type = "connection_request"  // 连接申请
	TypeConnectionAccepted Type = "connection_accepted" // 连接接受
)

// Channel identifies the delivery channel.
type Channel string

const (
	ChannelFCM    Channel = "fcm"
	ChannelEmail  Channel = "email"
	ChannelInApp  Channel = "in_app"
)

// Notification represents a notification to be sent.
type Notification struct {
	ID       string
	UserID   string
	TopicID  string
	Type     Type
	Channel  Channel
	Title    string
	Body     string
	Data     map[string]string
	ScheduledAt time.Time
}

// FCMSender sends push notifications via Firebase Cloud Messaging.
type FCMSender interface {
	SendToDevice(ctx context.Context, fcmToken, title, body string, data map[string]string) (string, error)
	SendToTopic(ctx context.Context, topic, title, body string, data map[string]string) (string, error)
}

// EmailSender sends emails via SendGrid.
type EmailSender interface {
	SendEmail(ctx context.Context, toEmail, toName, subject, htmlContent string) (string, error)
}

// UserRepository gets user notification preferences and FCM tokens.
type UserRepository interface {
	GetFCMToken(ctx context.Context, userID string) (string, error)
	GetEmail(ctx context.Context, userID string) (email, displayName string, err error)
}

// NotificationRepository persists notification state.
type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	UpdateStatus(ctx context.Context, id, status, externalID, errorMsg string) error
}

// Service handles notification delivery.
type Service struct {
	fcm      FCMSender
	email    EmailSender
	userRepo UserRepository
	repo     NotificationRepository
	logger   *zap.Logger
}

// NewService constructs a notification Service.
func NewService(
	fcm FCMSender,
	email EmailSender,
	userRepo UserRepository,
	repo NotificationRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		fcm:      fcm,
		email:    email,
		userRepo: userRepo,
		repo:     repo,
		logger:   logger,
	}
}

// Send delivers a notification via the appropriate channel.
func (s *Service) Send(ctx context.Context, n *Notification) error {
	var (
		externalID string
		sendErr    error
	)

	switch n.Channel {
	case ChannelFCM:
		fcmToken, err := s.userRepo.GetFCMToken(ctx, n.UserID)
		if err != nil || fcmToken == "" {
			// FCM token not available, skip silently
			s.logger.Debug("no FCM token for user, skipping push", zap.String("user_id", n.UserID))
			return s.repo.UpdateStatus(ctx, n.ID, "skipped", "", "no fcm token")
		}
		externalID, sendErr = s.fcm.SendToDevice(ctx, fcmToken, n.Title, n.Body, n.Data)

	case ChannelEmail:
		email, name, err := s.userRepo.GetEmail(ctx, n.UserID)
		if err != nil || email == "" {
			s.logger.Debug("no email for user, skipping", zap.String("user_id", n.UserID))
			return s.repo.UpdateStatus(ctx, n.ID, "skipped", "", "no email")
		}
		htmlBody := buildEmailHTML(n)
		externalID, sendErr = s.email.SendEmail(ctx, email, name, n.Title, htmlBody)

	case ChannelInApp:
		// In-app notifications are stored and polled by the client
		return s.repo.UpdateStatus(ctx, n.ID, "sent", "in_app", "")
	}

	if sendErr != nil {
		s.logger.Warn("notification send failed",
			zap.String("user_id", n.UserID),
			zap.String("type", string(n.Type)),
			zap.String("channel", string(n.Channel)),
			zap.Error(sendErr),
		)
		return s.repo.UpdateStatus(ctx, n.ID, "failed", "", sendErr.Error())
	}

	s.logger.Info("notification sent",
		zap.String("user_id", n.UserID),
		zap.String("type", string(n.Type)),
		zap.String("channel", string(n.Channel)),
		zap.String("external_id", externalID),
	)
	return s.repo.UpdateStatus(ctx, n.ID, "sent", externalID, "")
}

// SendMatchPreview sends the T+1h "match preview" notification.
func (s *Service) SendMatchPreview(ctx context.Context, userID, topicID, topicTitle string) error {
	n := &Notification{
		UserID:  userID,
		TopicID: topicID,
		Type:    TypeMatchPreview,
		Title:   "你的想法匹配成功了！",
		Body:    fmt.Sprintf("「%s」已匹配到4位数字分身，讨论即将开始～", truncate(topicTitle, 20)),
		Data: map[string]string{
			"topic_id": topicID,
			"screen":   "discussion_preview",
		},
		ScheduledAt: time.Now(),
	}
	return s.deliverMultiChannel(ctx, n, userID)
}

// SendDiscussionUpdate sends the T+12h "discussion update" notification.
func (s *Service) SendDiscussionUpdate(ctx context.Context, userID, topicID, topicTitle string) error {
	n := &Notification{
		UserID:  userID,
		TopicID: topicID,
		Type:    TypeDiscussionUpdate,
		Title:   "讨论进行中，快来看看！",
		Body:    fmt.Sprintf("「%s」的讨论正在火热进行中，已完成3轮精彩交锋", truncate(topicTitle, 20)),
		Data: map[string]string{
			"topic_id": topicID,
			"screen":   "discussion",
		},
		ScheduledAt: time.Now(),
	}
	return s.deliverMultiChannel(ctx, n, userID)
}

// SendReportReady sends the T+48h "full report ready" notification.
func (s *Service) SendReportReady(ctx context.Context, userID, topicID, reportID, topicTitle string) error {
	n := &Notification{
		UserID:  userID,
		TopicID: topicID,
		Type:    TypeReportReady,
		Title:   "你的讨论报告已生成！",
		Body:    fmt.Sprintf("「%s」的完整分析报告已出炉，点击查看深度洞见", truncate(topicTitle, 20)),
		Data: map[string]string{
			"topic_id":  topicID,
			"report_id": reportID,
			"screen":    "report",
		},
		ScheduledAt: time.Now(),
	}
	return s.deliverMultiChannel(ctx, n, userID)
}

// deliverMultiChannel sends via both FCM (push) and in-app.
func (s *Service) deliverMultiChannel(ctx context.Context, n *Notification, userID string) error {
	// FCM push
	fcmN := *n
	fcmN.Channel = ChannelFCM
	if err := s.Send(ctx, &fcmN); err != nil {
		s.logger.Warn("FCM delivery failed", zap.Error(err))
	}

	// In-app
	inAppN := *n
	inAppN.Channel = ChannelInApp
	return s.Send(ctx, &inAppN)
}

func buildEmailHTML(n *Notification) string {
	return fmt.Sprintf(`<html><body>
<h2>%s</h2>
<p>%s</p>
<br>
<p>前往 <a href="https://digital-twin.community">数字分身社区</a> 查看详情</p>
</body></html>`, escapeHTML(n.Title), escapeHTML(n.Body))
}

// escapeHTML escapes HTML special characters to prevent XSS in email templates.
func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
