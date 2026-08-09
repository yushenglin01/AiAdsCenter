package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"html"
	"io"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/example/adnova/internal/config"
	"go.uber.org/zap"
)

type Sender interface {
	SendVerification(ctx context.Context, recipient, displayName, verificationURL string) error
}

type LogSender struct{ logger *zap.Logger }

func NewLogSender(logger *zap.Logger) *LogSender { return &LogSender{logger: logger} }

func (s *LogSender) SendVerification(_ context.Context, recipient, _ string, verificationURL string) error {
	s.logger.Warn("development email verification link", zap.String("recipient", recipient), zap.String("verification_url", verificationURL))
	return nil
}

type SMTPSender struct {
	config  config.MailConfig
	timeout time.Duration
}

func NewSMTPSender(cfg config.MailConfig) *SMTPSender {
	return &SMTPSender{config: cfg, timeout: 10 * time.Second}
}

func (s *SMTPSender) SendVerification(ctx context.Context, recipient, displayName, verificationURL string) error {
	if containsHeaderBreak(recipient) || containsHeaderBreak(displayName) || containsHeaderBreak(verificationURL) {
		return fmt.Errorf("email input contains an invalid line break")
	}
	host, _, err := net.SplitHostPort(s.config.SMTPAddress)
	if err != nil {
		return fmt.Errorf("parse SMTP address: %w", err)
	}
	dialer := net.Dialer{Timeout: s.timeout}
	connection, err := dialer.DialContext(ctx, "tcp", s.config.SMTPAddress)
	if err != nil {
		return fmt.Errorf("connect SMTP: %w", err)
	}
	client, err := smtp.NewClient(connection, host)
	if err != nil {
		_ = connection.Close()
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server does not support required STARTTLS")
	}
	if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
		return fmt.Errorf("start SMTP TLS: %w", err)
	}
	if s.config.SMTPUsername != "" {
		if err := client.Auth(smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, host)); err != nil {
			return fmt.Errorf("authenticate SMTP: %w", err)
		}
	}
	if err := client.Mail(s.config.FromAddress); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP message: %w", err)
	}
	if _, err := io.WriteString(w, s.message(recipient, displayName, verificationURL)); err != nil {
		_ = w.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close SMTP message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("finish SMTP session: %w", err)
	}
	return nil
}

func (s *SMTPSender) message(recipient, displayName, verificationURL string) string {
	fromName := s.config.FromName
	if fromName == "" {
		fromName = "AdNova"
	}
	encodedName := mime.QEncoding.Encode("UTF-8", fromName)
	encodedSubject := mime.QEncoding.Encode("UTF-8", "确认你的 AdNova 企业成员申请")
	name := html.EscapeString(displayName)
	link := html.EscapeString(verificationURL)
	body := fmt.Sprintf("<!doctype html><html><body><p>%s，你好：</p><p>请点击下面的链接确认公司邮箱。邮箱确认后，还需要管理员授权才能登录。</p><p><a href=\"%s\">确认邮箱</a></p><p>如果你没有发起申请，请忽略此邮件。</p></body></html>", name, link)
	return strings.Join([]string{
		"From: " + encodedName + " <" + s.config.FromAddress + ">",
		"To: " + recipient,
		"Subject: " + encodedSubject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")
}

func containsHeaderBreak(value string) bool { return strings.ContainsAny(value, "\r\n") }
