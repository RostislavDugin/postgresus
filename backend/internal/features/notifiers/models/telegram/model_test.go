package telegram_notifier

import (
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
