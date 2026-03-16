package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"strconv"
)

type Mailer interface {
	SendOTP(to, name, otp string, emailType OTPEmailType) error
	SendRaw(to, subject, body string) error
}

type OTPEmailType string

const (
	OTPEmailVerification   OTPEmailType = "email_verification"
	OTPEmailChangePassword OTPEmailType = "change_password"
)

type smtpMailer struct {
	host        string
	port        int
	username    string
	password    string
	fromAddress string // envelope sender — must match authenticated account
	fromHeader  string
}

func NewMailer() Mailer {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}

	username := os.Getenv("SMTP_USERNAME")

	fromHeader := os.Getenv("SMTP_FROM")
	if fromHeader == "" {
		fromHeader = username
	}

	return &smtpMailer{
		host:        os.Getenv("SMTP_HOST"),
		port:        port,
		username:    username,
		password:    os.Getenv("SMTP_PASSWORD"),
		fromAddress: username,
		fromHeader:  fromHeader,
	}
}

func (m *smtpMailer) SendOTP(to, name, otp string, emailType OTPEmailType) error {
	subject, body, err := buildOTPEmail(name, otp, emailType)
	if err != nil {
		return fmt.Errorf("failed to build email: %w", err)
	}

	return m.send(to, subject, body)
}

func (m *smtpMailer) send(to, subject, body string) error {
	auth := smtp.PlainAuth("", m.username, m.password, m.host)

	// From:   display address shown to recipient e.g. "No Reply <no-reply@yourdomain.com>"
	// Sender: actual authenticated account — tells providers who is really sending
	//         without this, providers rewrite From to match the authenticated account
	headers := fmt.Sprintf(
		"From: %s\r\nSender: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		m.fromHeader, m.fromAddress, to, subject,
	)

	msg := []byte(headers + body)
	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	return smtp.SendMail(addr, auth, m.fromAddress, []string{to}, msg)
}

func (m *smtpMailer) SendRaw(to, subject, body string) error {
	return m.send(to, subject, body)
}

// --- Templates ---

type otpTemplateData struct {
	Name    string
	OTP     string
	Minutes int
}

func buildOTPEmail(name, otp string, emailType OTPEmailType) (subject, body string, err error) {
	data := otpTemplateData{
		Name:    name,
		OTP:     otp,
		Minutes: 15,
	}

	switch emailType {
	case OTPEmailVerification:
		subject = "Verify your email address"
		body, err = renderTemplate(verificationTemplate, data)
	case OTPEmailChangePassword:
		subject = "Confirm your password change"
		body, err = renderTemplate(changePasswordTemplate, data)
	default:
		return "", "", fmt.Errorf("unknown email type: %s", emailType)
	}

	return subject, body, err
}

func renderTemplate(tmpl string, data otpTemplateData) (string, error) {
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
