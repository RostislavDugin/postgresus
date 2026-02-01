package email

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"time"
)

const (
	ImplicitTLSPort  = 465
	DefaultTimeout   = 5 * time.Second
	DefaultHelloName = "localhost"
	MIMETypeHTML     = "text/html"
	MIMECharsetUTF8  = "UTF-8"
)

type EmailSMTPSender struct {
	logger       *slog.Logger
	smtpHost     string
	smtpPort     int
	smtpUser     string
	smtpPassword string
	isConfigured bool
}

func (s *EmailSMTPSender) SendEmail(to, subject, body string) error {
	if !s.isConfigured {
		s.logger.Warn("Skipping email send, SMTP not initialized", "to", to, "subject", subject)
		return nil
	}

	from := s.smtpUser
	if from == "" {
		from = "noreply@" + s.smtpHost
	}

	emailContent := s.buildEmailContent(to, subject, body, from)
	isAuthRequired := s.smtpUser != "" && s.smtpPassword != ""

	if s.smtpPort == ImplicitTLSPort {
		return s.sendImplicitTLS(to, from, emailContent, isAuthRequired)
	}

	return s.sendStartTLS(to, from, emailContent, isAuthRequired)
}

func (s *EmailSMTPSender) buildEmailContent(to, subject, body, from string) []byte {
	// Encode Subject header using RFC 2047 to avoid SMTPUTF8 requirement
	encodedSubject := encodeRFC2047(subject)
	subjectHeader := fmt.Sprintf("Subject: %s\r\n", encodedSubject)
	// Get the current time
	now := time.Now()
	// Generate Date header that is compliant with RFC 5322 using the format from RFC 1123Z
	dateHeader := fmt.Sprintf("Date: %s\r\n", now.Format(time.RFC1123Z))

	mimeHeaders := fmt.Sprintf(
		"MIME-version: 1.0;\nContent-Type: %s; charset=\"%s\";\n\n",
		MIMETypeHTML,
		MIMECharsetUTF8,
	)

	// Encode From header display name if it contains non-ASCII
	encodedFrom := encodeRFC2047(from)
	fromHeader := fmt.Sprintf("From: %s\r\n", encodedFrom)

	toHeader := fmt.Sprintf("To: %s\r\n", to)

	return []byte(fromHeader + toHeader + subjectHeader + dateHeader + mimeHeaders + body)
}

func (s *EmailSMTPSender) sendImplicitTLS(
	to, from string,
	emailContent []byte,
	isAuthRequired bool,
) error {
	createClient := func() (*smtp.Client, func(), error) {
		return s.createImplicitTLSClient()
	}

	client, cleanup, err := s.authenticateWithRetry(createClient, isAuthRequired)
	if err != nil {
		return err
	}
	defer cleanup()

	return s.sendEmail(client, to, from, emailContent)
}

func (s *EmailSMTPSender) sendStartTLS(
	to, from string,
	emailContent []byte,
	isAuthRequired bool,
) error {
	createClient := func() (*smtp.Client, func(), error) {
		return s.createStartTLSClient()
	}

	client, cleanup, err := s.authenticateWithRetry(createClient, isAuthRequired)
	if err != nil {
		return err
	}
	defer cleanup()

	return s.sendEmail(client, to, from, emailContent)
}

func (s *EmailSMTPSender) createImplicitTLSClient() (*smtp.Client, func(), error) {
	addr := net.JoinHostPort(s.smtpHost, fmt.Sprintf("%d", s.smtpPort))
	tlsConfig := &tls.Config{ServerName: s.smtpHost}
	dialer := &net.Dialer{Timeout: DefaultTimeout}

	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, s.smtpHost)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	return client, func() { _ = client.Quit() }, nil
}

func (s *EmailSMTPSender) createStartTLSClient() (*smtp.Client, func(), error) {
	addr := net.JoinHostPort(s.smtpHost, fmt.Sprintf("%d", s.smtpPort))
	dialer := &net.Dialer{Timeout: DefaultTimeout}

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, s.smtpHost)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if err := client.Hello(DefaultHelloName); err != nil {
		_ = client.Quit()
		_ = conn.Close()
		return nil, nil, fmt.Errorf("SMTP hello failed: %w", err)
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.smtpHost}); err != nil {
			_ = client.Quit()
			_ = conn.Close()
			return nil, nil, fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	return client, func() { _ = client.Quit() }, nil
}

func (s *EmailSMTPSender) authenticateWithRetry(
	createClient func() (*smtp.Client, func(), error),
	isAuthRequired bool,
) (*smtp.Client, func(), error) {
	client, cleanup, err := createClient()
	if err != nil {
		return nil, nil, err
	}

	if !isAuthRequired {
		return client, cleanup, nil
	}

	// Try PLAIN auth first
	plainAuth := smtp.PlainAuth("", s.smtpUser, s.smtpPassword, s.smtpHost)
	if err := client.Auth(plainAuth); err == nil {
		return client, cleanup, nil
	}

	// PLAIN auth failed, connection may be closed - recreate and try LOGIN auth
	cleanup()

	client, cleanup, err = createClient()
	if err != nil {
		return nil, nil, err
	}

	loginAuth := &loginAuth{username: s.smtpUser, password: s.smtpPassword}
	if err := client.Auth(loginAuth); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("SMTP authentication failed: %w", err)
	}

	return client, cleanup, nil
}

func (s *EmailSMTPSender) sendEmail(client *smtp.Client, to, from string, content []byte) error {
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err = writer.Write(content); err != nil {
		return fmt.Errorf("failed to write email content: %w", err)
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}

func encodeRFC2047(s string) string {
	return mime.QEncoding.Encode("UTF-8", s)
}

type loginAuth struct {
	username string
	password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:", "User Name\x00":
			return []byte(a.username), nil
		case "Password:", "Password\x00":
			return []byte(a.password), nil
		default:
			return []byte(a.username), nil
		}
	}
	return nil, nil
}
