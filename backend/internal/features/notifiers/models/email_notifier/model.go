package email_notifier

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"postgresus-backend/internal/util/encryption"
	"time"

	"github.com/google/uuid"
)

const (
	ImplicitTLSPort  = 465
	DefaultTimeout   = 5 * time.Second
	DefaultHelloName = "localhost"
	MIMETypeHTML     = "text/html"
	MIMECharsetUTF8  = "UTF-8"
)

type EmailNotifier struct {
	NotifierID   uuid.UUID `json:"notifierId"   gorm:"primaryKey;type:uuid;column:notifier_id"`
	TargetEmail  string    `json:"targetEmail"  gorm:"not null;type:varchar(255);column:target_email"`
	SMTPHost     string    `json:"smtpHost"     gorm:"not null;type:varchar(255);column:smtp_host"`
	SMTPPort     int       `json:"smtpPort"     gorm:"not null;column:smtp_port"`
	SMTPUser     string    `json:"smtpUser"     gorm:"type:varchar(255);column:smtp_user"`
	SMTPPassword string    `json:"smtpPassword" gorm:"type:varchar(255);column:smtp_password"`
	From         string    `json:"from"         gorm:"type:varchar(255);column:from_email"`
}

func (e *EmailNotifier) TableName() string {
	return "email_notifiers"
}

func (e *EmailNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if e.TargetEmail == "" {
		return errors.New("target email is required")
	}

	if e.SMTPHost == "" {
		return errors.New("SMTP host is required")
	}

	if e.SMTPPort == 0 {
		return errors.New("SMTP port is required")
	}

	// Authentication is optional - both user and password must be provided together or both empty
	if (e.SMTPUser == "") != (e.SMTPPassword == "") {
		return errors.New("SMTP user and password must both be provided or both be empty")
	}

	return nil
}

func (e *EmailNotifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	heading string,
	message string,
) error {
	// Decrypt SMTP password if provided
	var smtpPassword string
	if e.SMTPPassword != "" {
		decrypted, err := encryptor.Decrypt(e.NotifierID, e.SMTPPassword)
		if err != nil {
			return fmt.Errorf("failed to decrypt SMTP password: %w", err)
		}
		smtpPassword = decrypted
	}

	// Compose email
	from := e.From
	if from == "" {
		from = e.SMTPUser
		if from == "" {
			from = "noreply@" + e.SMTPHost
		}
	}

	to := []string{e.TargetEmail}

	// Format the email content
	subject := fmt.Sprintf("Subject: %s\r\n", heading)
	mime := fmt.Sprintf(
		"MIME-version: 1.0;\nContent-Type: %s; charset=\"%s\";\n\n",
		MIMETypeHTML,
		MIMECharsetUTF8,
	)
	body := message
	fromHeader := fmt.Sprintf("From: %s\r\n", from)
	toHeader := fmt.Sprintf("To: %s\r\n", e.TargetEmail)

	// Combine all parts of the email
	emailContent := []byte(fromHeader + toHeader + subject + mime + body)

	addr := net.JoinHostPort(e.SMTPHost, fmt.Sprintf("%d", e.SMTPPort))
	timeout := DefaultTimeout

	// Determine if authentication is required
	isAuthRequired := e.SMTPUser != "" && smtpPassword != ""

	// Handle different port scenarios
	if e.SMTPPort == ImplicitTLSPort {
		// Implicit TLS (port 465)
		// Set up TLS config
		tlsConfig := &tls.Config{
			ServerName: e.SMTPHost,
		}

		// Dial with timeout
		dialer := &net.Dialer{Timeout: timeout}
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		defer func() {
			_ = conn.Close()
		}()

		// Create SMTP client
		client, err := smtp.NewClient(conn, e.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer func() {
			_ = client.Quit()
		}()

		// Set up authentication only if credentials are provided
		if isAuthRequired {
			if err := e.authenticate(client, smtpPassword); err != nil {
				return err
			}
		}

		// Set sender and recipients
		if err := client.Mail(from); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}
		for _, recipient := range to {
			if err := client.Rcpt(recipient); err != nil {
				return fmt.Errorf("failed to set recipient: %w", err)
			}
		}

		// Send the email body
		writer, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %w", err)
		}
		_, err = writer.Write(emailContent)
		if err != nil {
			return fmt.Errorf("failed to write email content: %w", err)
		}
		err = writer.Close()
		if err != nil {
			return fmt.Errorf("failed to close data writer: %w", err)
		}

		return nil
	} else {
		// STARTTLS (port 587) or other ports
		// Create a custom dialer with timeout
		dialer := &net.Dialer{Timeout: timeout}
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}

		// Create client from connection
		client, err := smtp.NewClient(conn, e.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer func() {
			_ = client.Quit()
		}()

		// Send email using the client
		if err := client.Hello(DefaultHelloName); err != nil {
			return fmt.Errorf("SMTP hello failed: %w", err)
		}

		// Start TLS if available
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: e.SMTPHost}); err != nil {
				return fmt.Errorf("STARTTLS failed: %w", err)
			}
		}

		// Authenticate only if credentials are provided
		if isAuthRequired {
			if err := e.authenticate(client, smtpPassword); err != nil {
				return err
			}
		}

		if err := client.Mail(from); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}

		for _, recipient := range to {
			if err := client.Rcpt(recipient); err != nil {
				return fmt.Errorf("failed to set recipient: %w", err)
			}
		}

		writer, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %w", err)
		}

		_, err = writer.Write(emailContent)
		if err != nil {
			return fmt.Errorf("failed to write email content: %w", err)
		}

		err = writer.Close()
		if err != nil {
			return fmt.Errorf("failed to close data writer: %w", err)
		}

		return client.Quit()
	}
}

func (e *EmailNotifier) HideSensitiveData() {
	e.SMTPPassword = ""
}

func (e *EmailNotifier) Update(incoming *EmailNotifier) {
	e.TargetEmail = incoming.TargetEmail
	e.SMTPHost = incoming.SMTPHost
	e.SMTPPort = incoming.SMTPPort
	e.SMTPUser = incoming.SMTPUser
	e.From = incoming.From

	if incoming.SMTPPassword != "" {
		e.SMTPPassword = incoming.SMTPPassword
	}
}

func (e *EmailNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if e.SMTPPassword != "" {
		encrypted, err := encryptor.Encrypt(e.NotifierID, e.SMTPPassword)
		if err != nil {
			return fmt.Errorf("failed to encrypt SMTP password: %w", err)
		}
		e.SMTPPassword = encrypted
	}
	return nil
}

func (e *EmailNotifier) authenticate(client *smtp.Client, password string) error {
	// Try PLAIN auth first (most common)
	plainAuth := smtp.PlainAuth("", e.SMTPUser, password, e.SMTPHost)
	if err := client.Auth(plainAuth); err == nil {
		return nil
	}

	// If PLAIN fails, try LOGIN auth (required by Office 365 and some providers)
	loginAuth := &loginAuth{e.SMTPUser, password}
	if err := client.Auth(loginAuth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	return nil
}
