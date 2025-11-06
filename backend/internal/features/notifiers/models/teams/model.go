package teams_notifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"
)

type TeamsNotifier struct {
	NotifierID uuid.UUID `gorm:"type:uuid;primaryKey;column:notifier_id"      json:"notifierId"`
	WebhookURL string    `gorm:"type:text;not null;column:power_automate_url" json:"powerAutomateUrl"`
}

func (TeamsNotifier) TableName() string {
	return "teams_notifiers"
}

func (n *TeamsNotifier) Validate() error {
	if n.WebhookURL == "" {
		return errors.New("webhook_url is required")
	}
	u, err := url.Parse(n.WebhookURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("invalid webhook_url")
	}
	return nil
}

type cardAttachment struct {
	ContentType string      `json:"contentType"`
	Content     interface{} `json:"content"`
}

type payload struct {
	Title       string           `json:"title"`
	Text        string           `json:"text"`
	Attachments []cardAttachment `json:"attachments,omitempty"`
}

func (n *TeamsNotifier) Send(logger *slog.Logger, vars map[string]string) error {
	// Get heading and message for backward compatibility
	heading := vars["heading"]
	message := vars["message"]

	if err := n.Validate(); err != nil {
		return err
	}

	card := map[string]any{
		"type":    "AdaptiveCard",
		"version": "1.4",
		"body": []any{
			map[string]any{
				"type":   "TextBlock",
				"size":   "Medium",
				"weight": "Bolder",
				"text":   heading,
			},
			map[string]any{"type": "TextBlock", "wrap": true, "text": message},
		},
	}

	p := payload{
		Title: heading,
		Text:  message,
		Attachments: []cardAttachment{
			{ContentType: "application/vnd.microsoft.card.adaptive", Content: card},
		},
	}

	body, _ := json.Marshal(p)
	req, err := http.NewRequest(http.MethodPost, n.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
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
