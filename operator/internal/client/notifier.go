package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type NotifierRequest struct {
	ID               string           `json:"id,omitempty"`
	WorkspaceID      string           `json:"workspaceId"`
	Name             string           `json:"name"`
	NotifierType     string           `json:"notifierType"`
	DiscordNotifier  *DiscordRequest  `json:"discordNotifier,omitempty"`
	TelegramNotifier *TelegramRequest `json:"telegramNotifier,omitempty"`
	SlackNotifier    *SlackRequest    `json:"slackNotifier,omitempty"`
	EmailNotifier    *EmailRequest    `json:"emailNotifier,omitempty"`
	WebhookNotifier  *WebhookRequest  `json:"webhookNotifier,omitempty"`
	TeamsNotifier    *TeamsRequest    `json:"teamsNotifier,omitempty"`
}

type DiscordRequest struct {
	ChannelWebhookURL string `json:"channelWebhookUrl"`
}

type TelegramRequest struct {
	BotToken     string `json:"botToken"`
	TargetChatID string `json:"targetChatId"`
	ThreadID     *int64 `json:"threadId,omitempty"`
}

type SlackRequest struct {
	BotToken     string `json:"botToken"`
	TargetChatID string `json:"targetChatId"`
}

type EmailRequest struct {
	TargetEmail          string `json:"targetEmail"`
	SMTPHost             string `json:"smtpHost"`
	SMTPPort             int    `json:"smtpPort"`
	SMTPUser             string `json:"smtpUser,omitempty"`
	SMTPPassword         string `json:"smtpPassword,omitempty"`
	From                 string `json:"from,omitempty"`
	IsInsecureSkipVerify bool   `json:"isInsecureSkipVerify"`
}

type WebhookRequest struct {
	WebhookURL    string                  `json:"webhookUrl"`
	WebhookMethod string                  `json:"webhookMethod"`
	BodyTemplate  string                  `json:"bodyTemplate,omitempty"`
	Headers       []WebhookHeaderRequest  `json:"headers,omitempty"`
}

type WebhookHeaderRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type TeamsRequest struct {
	ChannelWebhookURL string `json:"channelWebhookUrl"`
}

type NotifierResponse struct {
	ID           string  `json:"id"`
	WorkspaceID  string  `json:"workspaceId"`
	Name         string  `json:"name"`
	NotifierType string  `json:"notifierType"`
	LastSendError *string `json:"lastSendError"`
}

func (c *DatabasusClient) SaveNotifier(ctx context.Context, req *NotifierRequest) (*NotifierResponse, error) {
	if req.WorkspaceID == "" {
		req.WorkspaceID = c.workspaceID
	}

	body, statusCode, err := c.do(ctx, http.MethodPost, "/api/v1/notifiers", req)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, parseError(statusCode, body)
	}

	var resp NotifierResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notifier response: %w", err)
	}

	return &resp, nil
}

func (c *DatabasusClient) GetNotifier(ctx context.Context, notifierID string) (*NotifierResponse, error) {
	body, statusCode, err := c.do(ctx, http.MethodGet, "/api/v1/notifiers/"+notifierID, nil)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusNotFound {
		return nil, nil
	}

	if statusCode != http.StatusOK {
		return nil, parseError(statusCode, body)
	}

	var resp NotifierResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notifier response: %w", err)
	}

	return &resp, nil
}

func (c *DatabasusClient) DeleteNotifier(ctx context.Context, notifierID string) error {
	body, statusCode, err := c.do(ctx, http.MethodDelete, "/api/v1/notifiers/"+notifierID, nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
		return parseError(statusCode, body)
	}

	return nil
}
