package webhook_notifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"postgresus-backend/internal/util/encryption"

	"github.com/google/uuid"
)

type WebhookNotifier struct {
	NotifierID    uuid.UUID     `json:"notifierId"    gorm:"primaryKey;column:notifier_id"`
	WebhookURL    string        `json:"webhookUrl"    gorm:"not null;column:webhook_url"`
	WebhookMethod WebhookMethod `json:"webhookMethod" gorm:"not null;column:webhook_method"`
}

func (t *WebhookNotifier) TableName() string {
	return "webhook_notifiers"
}

func (t *WebhookNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if t.WebhookURL == "" {
		return errors.New("webhook URL is required")
	}

	if t.WebhookMethod == "" {
		return errors.New("webhook method is required")
	}

	return nil
}

func (t *WebhookNotifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	heading string,
	message string,
) error {
	webhookURL, err := encryptor.Decrypt(t.NotifierID, t.WebhookURL)
	if err != nil {
		return fmt.Errorf("failed to decrypt webhook URL: %w", err)
	}

	switch t.WebhookMethod {
	case WebhookMethodGET:
		reqURL := fmt.Sprintf("%s?heading=%s&message=%s",
			webhookURL,
			url.QueryEscape(heading),
			url.QueryEscape(message),
		)

		resp, err := http.Get(reqURL)
		if err != nil {
			return fmt.Errorf("failed to send GET webhook: %w", err)
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil {
				logger.Error("failed to close response body", "error", cerr)
			}
		}()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf(
				"webhook GET returned status: %s, body: %s",
				resp.Status,
				string(body),
			)
		}

		return nil

	case WebhookMethodPOST:
		payload := map[string]string{
			"heading": heading,
			"message": message,
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal webhook payload: %w", err)
		}

		resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to send POST webhook: %w", err)
		}

		defer func() {
			if cerr := resp.Body.Close(); cerr != nil {
				logger.Error("failed to close response body", "error", cerr)
			}
		}()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf(
				"webhook POST returned status: %s, body: %s",
				resp.Status,
				string(body),
			)
		}

		return nil

	default:
		return fmt.Errorf("unsupported webhook method: %s", t.WebhookMethod)
	}
}

func (t *WebhookNotifier) HideSensitiveData() {
}

func (t *WebhookNotifier) Update(incoming *WebhookNotifier) {
	t.WebhookURL = incoming.WebhookURL
	t.WebhookMethod = incoming.WebhookMethod
}

func (t *WebhookNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if t.WebhookURL != "" {
		encrypted, err := encryptor.Encrypt(t.NotifierID, t.WebhookURL)
		if err != nil {
			return fmt.Errorf("failed to encrypt webhook URL: %w", err)
		}
		t.WebhookURL = encrypted
	}
	return nil
}
