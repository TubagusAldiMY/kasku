package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/TubagusAldiMY/kasku/notification-service/configs"
)

// Sender mendefinisikan kontrak untuk pengiriman email.
type Sender interface {
	Send(to, subject, htmlBody string) error
}

// SMTPSender mengimplementasikan Sender via net/smtp.
type SMTPSender struct {
	cfg *configs.SMTPConfig
}

func NewSMTPSender(cfg *configs.SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Send(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	var auth smtp.Auth
	if s.cfg.User != "" {
		auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
	}

	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg := []byte(
		"To: " + to + "\r\n" +
			"From: KasKu <" + s.cfg.From + ">\r\n" +
			"Subject: " + subject + "\r\n" +
			mime +
			htmlBody,
	)

	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, msg)
}

// NoOpSender adalah implementasi no-op untuk development (log saja, tidak kirim).
type NoOpSender struct{}

func NewNoOpSender() *NoOpSender { return &NoOpSender{} }

func (s *NoOpSender) Send(to, subject, htmlBody string) error {
	// Di development, email tidak dikirim — hanya di-log oleh caller
	return nil
}

// RenderTemplate merender HTML template dengan data yang diberikan.
func RenderTemplate(tmpl *template.Template, name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("gagal render template %s: %w", name, err)
	}
	return buf.String(), nil
}
