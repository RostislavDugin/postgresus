package notifier_models

import (
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_StripWebhookURLCredentials_WithTokenBearingUrls_RemovesCredential(t *testing.T) {
	cases := []struct {
		name      string
		rawURL    string
		secret    string
		sanitized string
	}{
		{
			name:      "telegram fuses the bot prefix and the token into one segment",
			rawURL:    "https://api.telegram.org/bot123456:AAH-REALTOKEN/sendMessage",
			secret:    "AAH-REALTOKEN",
			sanitized: "https://api.telegram.org/botredacted/redacted",
		},
		{
			name:      "discord carries the token as a trailing path segment",
			rawURL:    "https://discord.com/api/webhooks/999/SECRETWEBHOOKTOKEN",
			secret:    "SECRETWEBHOOKTOKEN",
			sanitized: "https://discord.com/api/redacted/redacted/redacted",
		},
		{
			name:      "mattermost incoming webhook",
			rawURL:    "https://chat.example.com/hooks/xxxSECRETHOOKxxx",
			secret:    "xxxSECRETHOOKxxx",
			sanitized: "https://chat.example.com/hooks/redacted",
		},
		{
			name:      "query string can carry a token",
			rawURL:    "https://hooks.example.com/notify?token=SECRETQUERYTOKEN",
			secret:    "SECRETQUERYTOKEN",
			sanitized: "https://hooks.example.com/notify",
		},
		{
			name:      "userinfo credentials",
			rawURL:    "https://user:SECRETPASSWORD@hooks.example.com/notify",
			secret:    "SECRETPASSWORD",
			sanitized: "https://redacted@hooks.example.com/notify",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			sanitizedURL := StripWebhookURLCredentials(testCase.rawURL)

			assert.NotContains(t, sanitizedURL, testCase.secret)
			assert.Equal(t, testCase.sanitized, sanitizedURL)
		})
	}
}

func Test_StripWebhookURLCredentials_WithUnparseableUrl_ReturnsRedactedMarker(t *testing.T) {
	assert.Equal(t, redactedURLSegment, StripWebhookURLCredentials("://not a url\x7f"))
}

func Test_StripWebhookURLCredentials_KeepsHostReadable(t *testing.T) {
	assert.Contains(t, StripWebhookURLCredentials("https://api.telegram.org/bot1:X/sendMessage"), "api.telegram.org")
}

func Test_ErrorWithoutWebhookURLCredentials_WithTelegramTransportError_ErrorContainsNoBotToken(t *testing.T) {
	transportError := &url.Error{
		Op:  "Post",
		URL: "https://api.telegram.org/bot123456:AAH-REALBOTTOKEN/sendMessage",
		Err: errors.New("dial tcp: i/o timeout"),
	}

	sanitizedError := ErrorWithoutWebhookURLCredentials(transportError)

	assert.NotContains(t, sanitizedError.Error(), "AAH-REALBOTTOKEN")
	assert.Contains(t, sanitizedError.Error(), "api.telegram.org")
	assert.Contains(t, sanitizedError.Error(), "i/o timeout")
}

func Test_ErrorWithoutWebhookURLCredentials_WithWebhookTransportError_ErrorContainsNoUrlCredentials(t *testing.T) {
	transportError := &url.Error{
		Op:  "Post",
		URL: "https://discord.com/api/webhooks/999/SECRETWEBHOOKTOKEN",
		Err: errors.New("connection refused"),
	}

	sanitizedError := ErrorWithoutWebhookURLCredentials(transportError)

	assert.NotContains(t, sanitizedError.Error(), "SECRETWEBHOOKTOKEN")
	assert.Contains(t, sanitizedError.Error(), "discord.com")
	assert.Contains(t, sanitizedError.Error(), "connection refused")
}

func Test_ErrorWithoutWebhookURLCredentials_WithNonTransportError_ReturnsItUnchanged(t *testing.T) {
	plainError := errors.New("failed to decrypt bot token")

	assert.Equal(t, plainError, ErrorWithoutWebhookURLCredentials(plainError))
}
