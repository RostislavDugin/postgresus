package teams_notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/util/encryption"
)

type TeamsNotifier struct {
	NotifierID uuid.UUID `gorm:"type:uuid;primaryKey;column:notifier_id"      json:"notifierId"`
	WebhookURL string    `gorm:"type:text;not null;column:power_automate_url" json:"powerAutomateUrl"`
}

func (TeamsNotifier) TableName() string {
	return "teams_notifiers"
}

func (n *TeamsNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if n.WebhookURL == "" {
		return errors.New("webhook_url is required")
	}

	webhookURL, err := encryptor.Decrypt(n.WebhookURL)
	if err != nil {
		return fmt.Errorf("failed to decrypt webhook URL: %w", err)
	}

	u, err := url.Parse(webhookURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("invalid webhook_url")
	}
	return nil
}

type cardAttachment struct {
	ContentType string `json:"contentType"`
	Content     any    `json:"content"`
}

type payload struct {
	Title       string           `json:"title"`
	Text        string           `json:"text"`
	Attachments []cardAttachment `json:"attachments,omitzero"`
}

func (n *TeamsNotifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	notification notifier_models.Notification,
) error {
	if err := n.Validate(encryptor); err != nil {
		return err
	}

	webhookURL, err := encryptor.Decrypt(n.WebhookURL)
	if err != nil {
		return fmt.Errorf("failed to decrypt webhook URL: %w", err)
	}

	card := map[string]any{
		"type":    "AdaptiveCard",
		"version": "1.4",
		"body": []any{
			map[string]any{
				"type":   "TextBlock",
				"size":   "Medium",
				"weight": "Bolder",
				"text":   notification.Heading,
			},
			map[string]any{"type": "TextBlock", "wrap": true, "text": notification.Message},
		},
	}

	p := payload{
		Title: notification.Heading,
		Text:  notification.Message,
		Attachments: []cardAttachment{
			{ContentType: "application/vnd.microsoft.card.adaptive", Content: card},
		},
	}

	body, _ := json.Marshal(p)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return notifier_models.ErrorWithoutWebhookURLCredentials(err)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Error("failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("teams webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (n *TeamsNotifier) HideSensitiveData() {
	n.WebhookURL = ""
}

func (n *TeamsNotifier) Update(incoming *TeamsNotifier) {
	if incoming.WebhookURL != "" {
		n.WebhookURL = incoming.WebhookURL
	}
}

func (n *TeamsNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if n.WebhookURL != "" {
		encrypted, err := encryptor.Encrypt(n.WebhookURL)
		if err != nil {
			return fmt.Errorf("failed to encrypt webhook URL: %w", err)
		}
		n.WebhookURL = encrypted
	}
	return nil
}
