package services

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"

	"github.com/tehilla-22/b2b-api/internal/config"
)

type EmailService struct {
	cfg *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

func (s *EmailService) SendVerificationEmail(to, token string) error {
	verifyURL := s.cfg.FrontendURL + "/verify-email?token=" + token
	body := fmt.Sprintf(`
		<h2>Welcome to Joenize</h2>
		<p>Click the link below to verify your email address:</p>
		<a href="%s">Verify Email</a>
		<p>If you did not create an account, please ignore this email.</p>
	`, verifyURL)

	return s.send(to, "Verify your Joenize account", body)
}

func (s *EmailService) SendPasswordReset(to, token string) error {
	resetURL := s.cfg.FrontendURL + "/reset-password?token=" + token
	body := fmt.Sprintf(`
		<h2>Password Reset Request</h2>
		<p>Click the link below to reset your password:</p>
		<a href="%s">Reset Password</a>
		<p>If you did not request this, please ignore this email.</p>
	`, resetURL)

	return s.send(to, "Reset your Joenize password", body)
}

func (s *EmailService) SendOnboardingConfirmation(to string) error {
	body := `
		<h2>Seller Onboarding Complete</h2>
		<p>Congratulations! Your seller onboarding payment has been confirmed.</p>
		<p>You can now set up your storefront and publish products.</p>
	`
	return s.send(to, "Welcome to Joenize Selling", body)
}

func (s *EmailService) send(to, subject, htmlBody string) error {
	if s.cfg.SMTPHost == "" {
		fmt.Printf("Email not sent (no SMTP configured): to=%s subject=%s\n", to, subject)
		return nil
	}

	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)

	var buf bytes.Buffer
	tpl := template.Must(template.New("email").Parse(s.smtpTemplate()))
	tpl.Execute(&buf, struct {
		From    string
		To      string
		Subject string
		Body    template.HTML
	}{
		From:    s.cfg.SMTPFrom,
		To:      to,
		Subject: subject,
		Body:    template.HTML(htmlBody),
	})

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	err := smtp.SendMail(addr, auth, s.cfg.SMTPFrom, []string{to}, buf.Bytes())
	if err != nil {
		log.Printf("SMTP send error (to=%s): %v", to, err)
	}
	return err
}

func (s *EmailService) smtpTemplate() string {
	return "From: {{.From}}\r\nTo: {{.To}}\r\nSubject: {{.Subject}}\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=utf-8\r\n\r\n{{.Body}}"
}
