package email

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"mime"
	"net/smtp"
	"strings"
	"time"

	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/platform/config"
)

// Sender gửi email qua SMTP hoặc fallback sang log khi chưa cấu hình SMTP.
type Sender struct {
	cfg config.EmailConfig
}

// NewSender tạo email sender mới.
func NewSender(cfg config.EmailConfig) domain.EmailSender {
	return &Sender{cfg: cfg}
}

func (s *Sender) SendVerificationEmail(ctx context.Context, user *domain.User, token string, expiresAt time.Time) error {
	subject := "Xac thuc email tai khoan"
	body := fmt.Sprintf(
		"Xin chao %s,\n\nSu dung token sau de xac thuc email cua ban:\n\n%s\n\nToken het han luc: %s (UTC)\n\nNeu ban khong yeu cau, vui long bo qua email nay.\n",
		s.displayName(user),
		token,
		expiresAt.UTC().Format(time.RFC3339),
	)
	return s.send(ctx, []string{user.Email}, subject, body)
}

func (s *Sender) SendPasswordResetEmail(ctx context.Context, user *domain.User, token string, expiresAt time.Time) error {
	subject := "Dat lai mat khau tai khoan"
	body := fmt.Sprintf(
		"Xin chao %s,\n\nSu dung token sau de dat lai mat khau:\n\n%s\n\nToken het han luc: %s (UTC)\n\nNeu ban khong yeu cau, vui long bo qua email nay.\n",
		s.displayName(user),
		token,
		expiresAt.UTC().Format(time.RFC3339),
	)
	return s.send(ctx, []string{user.Email}, subject, body)
}

func (s *Sender) send(ctx context.Context, recipients []string, subject, body string) error {
	if strings.TrimSpace(s.cfg.SMTPHost) == "" {
		slog.InfoContext(ctx, "smtp not configured, fallback to log email", "to", recipients, "subject", subject, "body", body)
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	msg, err := s.buildMessage(recipients, subject, body)
	if err != nil {
		return err
	}

	var auth smtp.Auth
	if strings.TrimSpace(s.cfg.Username) != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, s.cfg.FromEmail, recipients, msg); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

func (s *Sender) buildMessage(recipients []string, subject, body string) ([]byte, error) {
	from := s.cfg.FromEmail
	if strings.TrimSpace(s.cfg.FromName) != "" {
		from = mime.QEncoding.Encode("utf-8", s.cfg.FromName) + " <" + s.cfg.FromEmail + ">"
	}

	var buf bytes.Buffer
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + strings.Join(recipients, ", ") + "\r\n")
	buf.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(body)
	return buf.Bytes(), nil
}

func (s *Sender) displayName(user *domain.User) string {
	if user.FullName != nil && strings.TrimSpace(*user.FullName) != "" {
		return strings.TrimSpace(*user.FullName)
	}
	return user.Email
}
