package discord_notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/util/encryption"
)

type DiscordNotifier struct {
	NotifierID        uuid.UUID `json:"notifierId"        gorm:"primaryKey;column:notifier_id"`
	ChannelWebhookURL string    `json:"channelWebhookUrl" gorm:"not null;column:channel_webhook_url"`
}

func (d *DiscordNotifier) TableName() string {
	return "discord_notifiers"
}

func (d *DiscordNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if d.ChannelWebhookURL == "" {
		return errors.New("webhook URL is required")
	}

	return nil
}

func (d *DiscordNotifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	notification notifier_models.Notification,
) error {
	webhookURL, err := encryptor.Decrypt(d.ChannelWebhookURL)
	if err != nil {
		return fmt.Errorf("failed to decrypt webhook URL: %w", err)
	}

	fullMessage := notification.Heading
	if notification.Message != "" {
		fullMessage = fmt.Sprintf("%s\n\n%s", notification.Heading, notification.Message)
	}

	payload := map[string]any{
		"content": fullMessage,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", webhookURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Discord message: %w", notifier_models.ErrorWithoutWebhookURLCredentials(err))
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"discord API returned non-OK status: %s. Error: %s",
			resp.Status,
			string(bodyBytes),
		)
	}

	return nil
}

func (d *DiscordNotifier) HideSensitiveData() {
	d.ChannelWebhookURL = ""
}

func (d *DiscordNotifier) Update(incoming *DiscordNotifier) {
	if incoming.ChannelWebhookURL != "" {
		d.ChannelWebhookURL = incoming.ChannelWebhookURL
	}
}

func (d *DiscordNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if d.ChannelWebhookURL != "" {
		encrypted, err := encryptor.Encrypt(d.ChannelWebhookURL)
		if err != nil {
			return fmt.Errorf("failed to encrypt webhook URL: %w", err)
		}
		d.ChannelWebhookURL = encrypted
	}
	return nil
}
