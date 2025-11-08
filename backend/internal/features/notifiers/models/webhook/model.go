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

func (t *WebhookNotifier) Validate() error {
	if t.WebhookURL == "" {
		return errors.New("webhook URL is required")
	}

	if t.WebhookMethod == "" {
		return errors.New("webhook method is required")
	}

	return nil
}

func (t *WebhookNotifier) Send(logger *slog.Logger, heading string, message string) error {
	switch t.WebhookMethod {
	case WebhookMethodGET:
		reqURL := fmt.Sprintf("%s?heading=%s&message=%s",
			t.WebhookURL,
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

		resp, err := http.Post(t.WebhookURL, "application/json", bytes.NewReader(body))
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
