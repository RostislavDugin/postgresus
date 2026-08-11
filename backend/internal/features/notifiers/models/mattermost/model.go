package mattermost_notifier

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"

	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/util/encryption"
)

type MattermostNotifier struct {
	NotifierID   uuid.UUID    `json:"notifierId"   gorm:"type:uuid;primaryKey;column:notifier_id"`
	DeliveryMode DeliveryMode `json:"deliveryMode" gorm:"type:text;not null;column:delivery_mode"`

	WebhookURL string `json:"webhookUrl" gorm:"type:text;not null;default:'';column:webhook_url"`

	ServerURL string `json:"serverUrl" gorm:"type:text;not null;default:'';column:server_url"`
	BotToken  string `json:"botToken"  gorm:"type:text;not null;default:'';column:bot_token"`

	TargetChannelName string `json:"targetChannelName" gorm:"type:text;not null;default:'';column:target_channel_name"`
	TargetChannelID   string `json:"targetChannelId"   gorm:"type:text;not null;default:'';column:target_channel_id"`

	OverrideUsername string `json:"overrideUsername" gorm:"type:text;not null;default:'';column:override_username"`
	OverrideIconURL  string `json:"overrideIconUrl"  gorm:"type:text;not null;default:'';column:override_icon_url"`

	IsInsecureSkipVerify bool `json:"isInsecureSkipVerify" gorm:"not null;default:false;column:is_insecure_skip_verify"`
}

type mattermostRequest struct {
	URL         string
	IsURLSecret bool
	BotToken    string
	Payload     map[string]any
}

func (m *MattermostNotifier) TableName() string {
	return "mattermost_notifiers"
}

func (m *MattermostNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if m.OverrideIconURL != "" && !isHTTPURL(m.OverrideIconURL) {
		return errors.New("override icon URL must be an http or https URL")
	}

	switch m.DeliveryMode {
	case DeliveryModeWebhook:
		return m.validateWebhookMode(encryptor)
	case DeliveryModeBot:
		return m.validateBotMode()
	default:
		return fmt.Errorf("delivery mode must be %s or %s", DeliveryModeWebhook, DeliveryModeBot)
	}
}

func (m *MattermostNotifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	notification notifier_models.Notification,
) error {
	postText := buildPost(notification)

	switch m.DeliveryMode {
	case DeliveryModeWebhook:
		return m.sendViaIncomingWebhook(encryptor, logger, postText)
	case DeliveryModeBot:
		return m.sendViaBotAPI(encryptor, logger, postText)
	default:
		return fmt.Errorf("unknown delivery mode: %s", m.DeliveryMode)
	}
}

func (m *MattermostNotifier) HideSensitiveData() {
	m.WebhookURL = ""
	m.BotToken = ""
}

func (m *MattermostNotifier) Update(incoming *MattermostNotifier) {
	m.DeliveryMode = incoming.DeliveryMode
	m.ServerURL = incoming.ServerURL
	m.TargetChannelName = incoming.TargetChannelName
	m.TargetChannelID = incoming.TargetChannelID
	m.OverrideUsername = incoming.OverrideUsername
	m.OverrideIconURL = incoming.OverrideIconURL
	m.IsInsecureSkipVerify = incoming.IsInsecureSkipVerify

	if incoming.WebhookURL != "" {
		m.WebhookURL = incoming.WebhookURL
	}

	if incoming.BotToken != "" {
		m.BotToken = incoming.BotToken
	}
}

func (m *MattermostNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if m.WebhookURL != "" {
		encrypted, err := encryptor.Encrypt(m.WebhookURL)
		if err != nil {
			return fmt.Errorf("failed to encrypt incoming webhook URL: %w", err)
		}

		m.WebhookURL = encrypted
	}

	if m.BotToken != "" {
		encrypted, err := encryptor.Encrypt(m.BotToken)
		if err != nil {
			return fmt.Errorf("failed to encrypt bot token: %w", err)
		}

		m.BotToken = encrypted
	}

	return nil
}

func (m *MattermostNotifier) validateWebhookMode(encryptor encryption.FieldEncryptor) error {
	if m.WebhookURL == "" {
		return errors.New("incoming webhook URL is required")
	}

	webhookURL, err := encryptor.Decrypt(m.WebhookURL)
	if err != nil {
		return fmt.Errorf("failed to decrypt incoming webhook URL: %w", err)
	}

	if !isHTTPURL(webhookURL) {
		return errors.New("incoming webhook URL must be an http or https URL")
	}

	return nil
}

func (m *MattermostNotifier) validateBotMode() error {
	if m.ServerURL == "" {
		return errors.New("server URL is required")
	}

	if !isHTTPURL(m.ServerURL) {
		return errors.New("server URL must be an http or https URL")
	}

	if m.BotToken == "" {
		return errors.New("bot token is required")
	}

	if m.TargetChannelID == "" {
		return errors.New("target channel ID is required")
	}

	if len([]rune(m.TargetChannelID)) != channelIDLength {
		return fmt.Errorf(
			"target channel ID must be the %d-character Mattermost channel ID, not the channel name",
			channelIDLength,
		)
	}

	return nil
}

func (m *MattermostNotifier) sendViaIncomingWebhook(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	postText string,
) error {
	webhookURL, err := encryptor.Decrypt(m.WebhookURL)
	if err != nil {
		return fmt.Errorf("failed to decrypt incoming webhook URL: %w", err)
	}

	payload := map[string]any{"text": postText}

	if m.TargetChannelName != "" {
		payload["channel"] = m.TargetChannelName
	}

	if m.OverrideUsername != "" {
		payload["username"] = m.OverrideUsername
	}

	if m.OverrideIconURL != "" {
		payload["icon_url"] = m.OverrideIconURL
	}

	request := mattermostRequest{URL: webhookURL, IsURLSecret: true, Payload: payload}

	if err := m.sendRequest(request, logger); err != nil {
		return err
	}

	logger.Info("mattermost message sent via incoming webhook")

	return nil
}

func (m *MattermostNotifier) sendViaBotAPI(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	postText string,
) error {
	botToken, err := encryptor.Decrypt(m.BotToken)
	if err != nil {
		return fmt.Errorf("failed to decrypt bot token: %w", err)
	}

	payload := map[string]any{
		"channel_id": m.TargetChannelID,
		"message":    postText,
	}

	props := map[string]any{}

	if m.OverrideUsername != "" {
		props["override_username"] = m.OverrideUsername
	}

	if m.OverrideIconURL != "" {
		props["override_icon_url"] = m.OverrideIconURL
	}

	if len(props) > 0 {
		payload["props"] = props
	}

	createPostURL := strings.TrimSuffix(m.ServerURL, "/") + createPostPath

	request := mattermostRequest{URL: createPostURL, BotToken: botToken, Payload: payload}

	if err := m.sendRequest(request, logger); err != nil {
		return err
	}

	logger.Info("mattermost message sent via bot account", "channel_id", m.TargetChannelID)

	return nil
}

func (m *MattermostNotifier) sendRequest(request mattermostRequest, logger *slog.Logger) error {
	jsonPayload, err := json.Marshal(request.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Mattermost payload: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		request.URL,
		bytes.NewReader(jsonPayload),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", request.withoutSecretURL(err))
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	if request.BotToken != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+request.BotToken)
	}

	response, err := m.newHTTPClient().Do(httpRequest)
	if err != nil {
		return fmt.Errorf("failed to send Mattermost message: %w", request.withoutSecretURL(err))
	}

	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			logger.Warn("failed to close response body", "error", closeErr)
		}
	}()

	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf(
			"mattermost returned non-OK status: %s. Error: %s",
			response.Status,
			describeError(responseBody),
		)
	}

	return nil
}

func (m *MattermostNotifier) newHTTPClient() *http.Client {
	client := &http.Client{Timeout: sendTimeout}

	if m.IsInsecureSkipVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true,
			},
		}
	}

	return client
}

// An incoming webhook URL is itself a credential, and net/http wraps the request URL into every
// transport error, which the test-send endpoints (POST /notifiers/{id}/test and
// POST /notifiers/direct-test) return verbatim to the caller. A bot mode server URL is not a
// secret, so it stays in the error to keep failures diagnosable.
func (r mattermostRequest) withoutSecretURL(err error) error {
	if !r.IsURLSecret {
		return err
	}

	var urlError *url.Error
	if errors.As(err, &urlError) {
		return urlError.Err
	}

	return err
}

func isHTTPURL(candidate string) bool {
	parsed, err := url.Parse(candidate)
	if err != nil {
		return false
	}

	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func buildPost(notification notifier_models.Notification) string {
	postText := fmt.Sprintf("**%s**", notification.Heading)

	if notification.Message != "" {
		postText = fmt.Sprintf("%s\n\n%s", postText, notification.Message)
	}

	runes := []rune(postText)
	if len(runes) > maxPostLength {
		return string(runes[:maxPostLength])
	}

	return postText
}

func describeError(responseBody []byte) string {
	var mattermostError struct {
		Message string `json:"message"`
	}

	if err := json.Unmarshal(responseBody, &mattermostError); err == nil &&
		mattermostError.Message != "" {
		return mattermostError.Message
	}

	return string(responseBody)
}
