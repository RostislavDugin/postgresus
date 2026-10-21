package telegram_notifier

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifier_models "databasus-backend/internal/features/notifiers/models"
)

func Test_Validate_WhenProxyEnabledWithoutURL_ReturnsError(t *testing.T) {
	notifier := &TelegramNotifier{
		BotToken:       "token",
		TargetChatID:   "123456",
		IsProxyEnabled: true,
	}

	err := notifier.Validate(notifier_models.PassthroughEncryptor{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "proxy URL is required")
}

func Test_Validate_WhenProxyURLSchemeUnsupported_ReturnsError(t *testing.T) {
	notifier := &TelegramNotifier{
		BotToken:       "token",
		TargetChatID:   "123456",
		IsProxyEnabled: true,
		ProxyURL:       "ftp://proxy.example.com:3128",
	}

	err := notifier.Validate(notifier_models.PassthroughEncryptor{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "http/https/socks5/socks5h")
}

func Test_Validate_WhenProxyURLIsSocks5_ReturnsNoError(t *testing.T) {
	notifier := &TelegramNotifier{
		BotToken:       "token",
		TargetChatID:   "123456",
		IsProxyEnabled: true,
		ProxyURL:       "socks5://user:password@proxy.example.com:1080",
	}

	err := notifier.Validate(notifier_models.PassthroughEncryptor{})

	require.NoError(t, err)
}

func Test_Validate_WithBotAPIURL_AcceptsPlainHTTPAddressesOnly(t *testing.T) {
	testCases := []struct {
		botAPIURL     string
		expectedError string
	}{
		{botAPIURL: ""},
		{botAPIURL: "http://localhost:8081"},
		{botAPIURL: "http://telegram-bot-api:8081"},
		{botAPIURL: "https://gateway.example.com"},
		{botAPIURL: "https://gateway.example.com/telegram/"},
		{botAPIURL: "ftp://telegram.example.com", expectedError: "valid http/https URL with a host"},
		{botAPIURL: "telegram-bot-api:8081", expectedError: "valid http/https URL with a host"},
		{botAPIURL: "https://", expectedError: "valid http/https URL with a host"},
		{botAPIURL: "gateway.example.com/telegram", expectedError: "valid http/https URL with a host"},
		{botAPIURL: "https://user:password@gateway.example.com", expectedError: "username or password"},
		{botAPIURL: "https://user@gateway.example.com", expectedError: "username or password"},
		{botAPIURL: "https://gateway.example.com/telegram?key=1", expectedError: "query or fragment"},
		{botAPIURL: "https://gateway.example.com/telegram?", expectedError: "query or fragment"},
		{botAPIURL: "https://gateway.example.com/#bot", expectedError: "query or fragment"},
		{botAPIURL: "https://gateway.example.com/#", expectedError: "query or fragment"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.botAPIURL, func(t *testing.T) {
			notifier := &TelegramNotifier{
				BotToken:     "token",
				TargetChatID: "123456",
				BotAPIURL:    testCase.botAPIURL,
			}

			err := notifier.Validate(notifier_models.PassthroughEncryptor{})

			if testCase.expectedError == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedError)
		})
	}
}

func Test_GetSendMessageURL_WithoutBotAPIURL_UsesOfficialTelegramAPI(t *testing.T) {
	notifier := &TelegramNotifier{}

	sendMessageURL, err := notifier.getSendMessageURL("123456:AAH-TOKEN")

	require.NoError(t, err)
	assert.Equal(t, "https://api.telegram.org/bot123456:AAH-TOKEN/sendMessage", sendMessageURL)
}

func Test_Send_WhenTextExceedsTelegramLimit_TruncatesTextByCharacters(t *testing.T) {
	heading := "Backup failed"

	sentForm := sendNotificationAndCaptureForm(t, notifier_models.Notification{
		Type:    notifier_models.NotificationTypeBackupFailed,
		Heading: heading,
		Message: strings.Repeat("界", maxTextUTF16Units),
	})

	messagePrefixLength := maxTextUTF16Units - len(heading) - len("\n\n")
	assert.Equal(
		t,
		heading+"\n\n"+strings.Repeat("界", messagePrefixLength),
		sentForm.Get("text"),
	)
}

func Test_Send_WhenTextWithEmojiExceedsTelegramLimit_CountsEmojiAsTwoUnits(t *testing.T) {
	heading := "Backup failed"

	sentForm := sendNotificationAndCaptureForm(t, notifier_models.Notification{
		Type:    notifier_models.NotificationTypeBackupFailed,
		Heading: heading,
		Message: strings.Repeat("🔥", maxTextUTF16Units),
	})

	emojiThatFit := (maxTextUTF16Units - len(heading) - len("\n\n")) / 2
	assert.Equal(
		t,
		heading+"\n\n"+strings.Repeat("🔥", emojiThatFit),
		sentForm.Get("text"),
	)
}

func Test_Send_WhenMessageContainsHTMLSpecialCharacters_SendsPlainTextWithoutParseMode(t *testing.T) {
	message := "mysqldump: Got error: 1146: Table 'db.<orders>' doesn't exist & aborted"

	sentForm := sendNotificationAndCaptureForm(t, notifier_models.Notification{
		Type:    notifier_models.NotificationTypeBackupFailed,
		Heading: "Backup failed",
		Message: message,
	})

	assert.Equal(t, "Backup failed\n\n"+message, sentForm.Get("text"))
	assert.False(t, sentForm.Has("parse_mode"))
}

func sendNotificationAndCaptureForm(t *testing.T, notification notifier_models.Notification) url.Values {
	t.Helper()

	botAPIURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK, Body: `{"ok":true}`},
	)
	notifier := &TelegramNotifier{
		BotToken:     "123456:AAH-TOKEN",
		TargetChatID: "123456",
		BotAPIURL:    botAPIURL,
	}

	require.NoError(t, notifier.Send(
		notifier_models.PassthroughEncryptor{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		notification,
	))
	require.Equal(t, 1, recorder.GetRequestCount())

	sentForm, err := url.ParseQuery(recorder.GetLastRequest().Body)
	require.NoError(t, err)

	return sentForm
}
