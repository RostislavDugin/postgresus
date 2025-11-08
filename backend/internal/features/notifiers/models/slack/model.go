package slack_notifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SlackNotifier struct {
	NotifierID   uuid.UUID `json:"notifierId"   gorm:"primaryKey;column:notifier_id"`
	BotToken     string    `json:"botToken"     gorm:"not null;column:bot_token"`
	TargetChatID string    `json:"targetChatId" gorm:"not null;column:target_chat_id"`
}

func (s *SlackNotifier) TableName() string { return "slack_notifiers" }

func (s *SlackNotifier) Validate() error {
	if s.BotToken == "" {
		return errors.New("bot token is required")
	}

	if s.TargetChatID == "" {
		return errors.New("target channel ID is required")
	}

	if !strings.HasPrefix(s.TargetChatID, "C") && !strings.HasPrefix(s.TargetChatID, "G") &&
		!strings.HasPrefix(s.TargetChatID, "D") &&
		!strings.HasPrefix(s.TargetChatID, "U") {
		return errors.New(
			"target channel ID must be a valid Slack channel ID (starts with C, G, D) or User ID (starts with U)",
		)
	}

	return nil
}

func (s *SlackNotifier) Send(logger *slog.Logger, heading, message string) error {
	full := fmt.Sprintf("*%s*", heading)

	if message != "" {
		full = fmt.Sprintf("%s\n\n%s", full, message)
	}

	payload, _ := json.Marshal(map[string]any{
		"channel": s.TargetChatID,
		"text":    full,
		"mrkdwn":  true,
	})

	const (
		maxAttempts       = 5
		defaultBackoff    = 2 * time.Second // when Retry-After header missing
		backoffMultiplier = 1.5             // use exponential growth
	)

	var (
		backoff  = defaultBackoff
		attempts = 0
	)

	for {
		attempts++

		req, err := http.NewRequest(
			"POST",
			"https://slack.com/api/chat.postMessage",
			bytes.NewReader(payload),
		)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Authorization", "Bearer "+s.BotToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("send slack message: %w", err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				logger.Warn("Failed to close response body", "error", err)
			}
		}()

		if resp.StatusCode == http.StatusTooManyRequests { // 429
			retryAfter := backoff
			if h := resp.Header.Get("Retry-After"); h != "" {
				if seconds, _ := strconv.Atoi(h); seconds > 0 {
					retryAfter = time.Duration(seconds) * time.Second
				}
			}

			if attempts >= maxAttempts {
				return fmt.Errorf("rate-limited after %d attempts, giving up", attempts)
			}

			logger.Warn("Slack rate-limited, retrying", "after", retryAfter, "attempt", attempts)
			time.Sleep(retryAfter)
			backoff = time.Duration(float64(backoff) * backoffMultiplier)

			continue
		}

		// Slack always returns 200 for logical errors, so decode body
		var respBody struct {
			OK    bool   `json:"ok"`
			Error string `json:"error,omitempty"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			raw, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("decode response: %v – raw: %s", err, raw)
		}

		if !respBody.OK {
			return fmt.Errorf("slack API error: %s", respBody.Error)
		}

		logger.Info("Slack message sent", "channel", s.TargetChatID, "attempts", attempts)

		return nil
	}
}

func (s *SlackNotifier) HideSensitiveData() {
	s.BotToken = ""
}

func (s *SlackNotifier) Update(incoming *SlackNotifier) {
	s.TargetChatID = incoming.TargetChatID

	if incoming.BotToken != "" {
		s.BotToken = incoming.BotToken
	}
}
