package email_notifier

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/testing/mailpit"
)

func Test_SanitizeHeaderValue_StripsCRLFAndNUL(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"plain", "plain"},
		{"with\rcr", "withcr"},
		{"with\nlf", "withlf"},
		{"with\r\ncrlf", "withcrlf"},
		{"with\x00nul", "withnul"},
		{"a\r\nBcc: attacker@evil.com\r\n", "aBcc: attacker@evil.com"},
	}

	for _, c := range cases {
		got := sanitizeHeaderValue(c.input)
		if got != c.expected {
			t.Errorf("sanitizeHeaderValue(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}

func Test_BuildEmailContent_DropsInjectedHeadersFromTargetEmail(t *testing.T) {
	notifier := &EmailNotifier{
		NotifierID:  uuid.New(),
		TargetEmail: "user@example.com\r\nBcc: attacker@evil.com",
		SMTPHost:    "smtp.example.com",
		SMTPPort:    587,
	}

	content := string(notifier.buildEmailContent("subject", "<p>body</p>", "from@example.com"))

	if strings.Contains(content, "\r\nBcc:") || strings.Contains(content, "\nBcc:") {
		t.Errorf("Bcc header line was injected via TargetEmail: %q", content)
	}

	if !strings.Contains(content, "To: user@example.comBcc: attacker@evil.com\r\n") {
		t.Errorf("expected sanitized To header without CRLF, got: %q", content)
	}
}

func Test_EmailNotifierSend_WhenSmtpServerAccepts_DeliversMessageToRecipient(t *testing.T) {
	mailpitEndpoint := containers.StartMailpit(t)
	mailpitClient := mailpit.NewClient(
		fmt.Sprintf("%s:%d", mailpitEndpoint.HTTP.Host, mailpitEndpoint.HTTP.Port),
	)

	notifier := &EmailNotifier{
		NotifierID:  uuid.New(),
		TargetEmail: "recipient@databasus.local",
		SMTPHost:    mailpitEndpoint.SMTP.Host,
		SMTPPort:    mailpitEndpoint.SMTP.Port,
		From:        "sender@databasus.local",
	}

	err := notifier.Send(
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
		"Backup completed",
		"<b>All good</b>",
	)
	require.NoError(t, err)

	var delivered []mailpit.Message
	require.Eventually(t, func() bool {
		messages, fetchErr := mailpitClient.FetchMessages()
		if fetchErr != nil {
			return false
		}

		delivered = messages

		return len(messages) == 1
	}, 5*time.Second, 100*time.Millisecond, "Mailpit should receive exactly one message")

	assert.Equal(t, "Backup completed", delivered[0].Subject)
	require.Len(t, delivered[0].To, 1)
	assert.Equal(t, "recipient@databasus.local", delivered[0].To[0].Address)
}

func Test_BuildEmailContent_DropsInjectedHeadersFromSMTPHost(t *testing.T) {
	notifier := &EmailNotifier{
		NotifierID:  uuid.New(),
		TargetEmail: "user@example.com",
		SMTPHost:    "smtp.example.com>\r\nX-Injected: 1",
		SMTPPort:    587,
	}

	content := string(notifier.buildEmailContent("subject", "<p>body</p>", "from@example.com"))

	if strings.Contains(content, "\r\nX-Injected:") || strings.Contains(content, "\nX-Injected:") {
		t.Errorf("injected header line leaked via SMTPHost: %q", content)
	}
}
