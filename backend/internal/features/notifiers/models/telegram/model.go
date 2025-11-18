package telegram_notifier

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"postgresus-backend/internal/util/encryption"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type TelegramNotifier struct {
	NotifierID   uuid.UUID `json:"notifierId"   gorm:"primaryKey;column:notifier_id"`
	BotToken     string    `json:"botToken"     gorm:"not null;column:bot_token"`
	TargetChatID string    `json:"targetChatId" gorm:"not null;column:target_chat_id"`
	ThreadID     *int64    `json:"threadId"     gorm:"column:thread_id"`
}

func (t *TelegramNotifier) TableName() string {
	return "telegram_notifiers"
}

func (t *TelegramNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if t.BotToken == "" {
		return errors.New("bot token is required")
	}

	if t.TargetChatID == "" {
		return errors.New("target chat ID is required")
	}

	return nil
}

func (t *TelegramNotifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	heading string,
	message string,
) error {
	botToken, err := encryptor.Decrypt(t.NotifierID, t.BotToken)
	if err != nil {
		return fmt.Errorf("failed to decrypt bot token: %w", err)
	}

	fullMessage := heading
	if message != "" {
		fullMessage = fmt.Sprintf("%s\n\n%s", heading, message)
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	data := url.Values{}
	data.Set("chat_id", t.TargetChatID)
	data.Set("text", fullMessage)
	data.Set("parse_mode", "HTML")

	if t.ThreadID != nil && *t.ThreadID != 0 {
		data.Set("message_thread_id", strconv.FormatInt(*t.ThreadID, 10))
	}

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"telegram API returned non-OK status: %s. Error: %s",
			resp.Status,
			string(bodyBytes),
		)
	}

	return nil
}

func (t *TelegramNotifier) HideSensitiveData() {
	t.BotToken = ""
}

func (t *TelegramNotifier) Update(incoming *TelegramNotifier) {
	t.TargetChatID = incoming.TargetChatID
	t.ThreadID = incoming.ThreadID

	if incoming.BotToken != "" {
		t.BotToken = incoming.BotToken
	}
}

func (t *TelegramNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if t.BotToken != "" {
		encrypted, err := encryptor.Encrypt(t.NotifierID, t.BotToken)
		if err != nil {
			return fmt.Errorf("failed to encrypt bot token: %w", err)
		}
		t.BotToken = encrypted
	}
	return nil
}
