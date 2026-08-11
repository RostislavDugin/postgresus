package mattermost_notifier

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifier_models "databasus-backend/internal/features/notifiers/models"
)

func send(t *testing.T, notifier *MattermostNotifier, notification notifier_models.Notification) error {
	t.Helper()

	return notifier.Send(
		notifier_models.PassthroughEncryptor{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		notification,
	)
}

func backupFailedNotification() notifier_models.Notification {
	return notifier_models.Notification{
		Type:    notifier_models.NotificationTypeBackupFailed,
		Heading: "Backup failed",
		Message: "mydb: connection refused",
	}
}

func decodeBody(t *testing.T, body string) map[string]any {
	t.Helper()

	var decoded map[string]any

	require.NoError(t, json.Unmarshal([]byte(body), &decoded))

	return decoded
}

func Test_Send_WithWebhookMode_PostsMarkdownTextToHookURL(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK, Body: "ok"},
	)
	notifier := &MattermostNotifier{DeliveryMode: DeliveryModeWebhook, WebhookURL: webhookURL}

	require.NoError(t, send(t, notifier, backupFailedNotification()))

	require.Equal(t, 1, recorder.GetRequestCount())
	request := recorder.GetLastRequest()
	assert.Equal(t, http.MethodPost, request.Method)
	assert.Equal(t, "application/json", request.Headers.Get("Content-Type"))
	assert.Empty(t, request.Headers.Get("Authorization"))

	body := decodeBody(t, request.Body)
	assert.Equal(t, "**Backup failed**\n\nmydb: connection refused", body["text"])
	assert.NotContains(t, body, "channel")
	assert.NotContains(t, body, "username")
	assert.NotContains(t, body, "icon_url")
}

func Test_Send_WithWebhookModeAndOverrides_PostsChannelAndIdentityFields(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK, Body: "ok"},
	)
	notifier := &MattermostNotifier{
		DeliveryMode:      DeliveryModeWebhook,
		WebhookURL:        webhookURL,
		TargetChannelName: "town-square",
		OverrideUsername:  "Databasus",
		OverrideIconURL:   "https://databasus.com/icon.png",
	}

	require.NoError(t, send(t, notifier, backupFailedNotification()))

	body := decodeBody(t, recorder.GetLastRequest().Body)
	assert.Equal(t, "town-square", body["channel"])
	assert.Equal(t, "Databasus", body["username"])
	assert.Equal(t, "https://databasus.com/icon.png", body["icon_url"])
}

func Test_Send_WithHeadingOnly_PostsBoldHeadingWithoutTrailingNewlines(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK, Body: "ok"},
	)
	notifier := &MattermostNotifier{DeliveryMode: DeliveryModeWebhook, WebhookURL: webhookURL}

	require.NoError(t, send(t, notifier, notifier_models.Notification{
		Type:    notifier_models.NotificationTypeBackupSuccess,
		Heading: "Backup completed",
	}))

	body := decodeBody(t, recorder.GetLastRequest().Body)
	assert.Equal(t, "**Backup completed**", body["text"])
}

func Test_Send_WhenPostExceedsMattermostLimit_TruncatesText(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK, Body: "ok"},
	)
	notifier := &MattermostNotifier{DeliveryMode: DeliveryModeWebhook, WebhookURL: webhookURL}

	require.NoError(t, send(t, notifier, notifier_models.Notification{
		Type:    notifier_models.NotificationTypeBackupFailed,
		Heading: "Backup failed",
		Message: strings.Repeat("é", maxPostLength),
	}))

	body := decodeBody(t, recorder.GetLastRequest().Body)
	assert.Equal(t, maxPostLength, len([]rune(body["text"].(string))))
}

func Test_Send_WithBotMode_PostsToCreatePostEndpointWithBearerToken(t *testing.T) {
	serverURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusCreated, Body: `{"id":"post123"}`},
	)
	notifier := &MattermostNotifier{
		DeliveryMode:    DeliveryModeBot,
		ServerURL:       serverURL + "/",
		BotToken:        "bot-token",
		TargetChannelID: strings.Repeat("c", channelIDLength),
	}

	require.NoError(t, send(t, notifier, backupFailedNotification()))

	request := recorder.GetLastRequest()
	assert.Equal(t, http.MethodPost, request.Method)
	assert.Equal(t, createPostPath, request.Path)
	assert.Equal(t, "Bearer bot-token", request.Headers.Get("Authorization"))

	body := decodeBody(t, request.Body)
	assert.Equal(t, strings.Repeat("c", channelIDLength), body["channel_id"])
	assert.Equal(t, "**Backup failed**\n\nmydb: connection refused", body["message"])
	assert.NotContains(t, body, "props")
}

func Test_Send_WithBotModeAndOverrides_PostsIdentityFieldsInProps(t *testing.T) {
	serverURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusCreated, Body: `{"id":"post123"}`},
	)
	notifier := &MattermostNotifier{
		DeliveryMode:     DeliveryModeBot,
		ServerURL:        serverURL,
		BotToken:         "bot-token",
		TargetChannelID:  strings.Repeat("c", channelIDLength),
		OverrideUsername: "Databasus",
		OverrideIconURL:  "https://databasus.com/icon.png",
	}

	require.NoError(t, send(t, notifier, backupFailedNotification()))

	props := decodeBody(t, recorder.GetLastRequest().Body)["props"].(map[string]any)
	assert.Equal(t, "Databasus", props["override_username"])
	assert.Equal(t, "https://databasus.com/icon.png", props["override_icon_url"])
}

func Test_Send_WhenServerReturnsNon2xx_ReturnsError(t *testing.T) {
	webhookURL, _ := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusInternalServerError, Body: "boom"},
	)
	notifier := &MattermostNotifier{DeliveryMode: DeliveryModeWebhook, WebhookURL: webhookURL}

	err := send(t, notifier, backupFailedNotification())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func Test_Send_WhenServerReturnsMattermostErrorBody_ReturnsItsMessage(t *testing.T) {
	serverURL, _ := notifier_models.StartRecordingServer(t, notifier_models.StubResponse{
		StatusCode: http.StatusForbidden,
		Body:       `{"id":"api.context.permissions.app_error","message":"You do not have the appropriate permissions."}`,
	})
	notifier := &MattermostNotifier{
		DeliveryMode:    DeliveryModeBot,
		ServerURL:       serverURL,
		BotToken:        "bot-token",
		TargetChannelID: strings.Repeat("c", channelIDLength),
	}

	err := send(t, notifier, backupFailedNotification())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "You do not have the appropriate permissions.")
}

func Test_Send_WhenServerUnreachable_ReturnsErrorWithoutTheWebhookURL(t *testing.T) {
	webhookURL, _ := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK},
	)
	unreachableURL := webhookURL + "-gone/hooks/secret-key"
	notifier := &MattermostNotifier{DeliveryMode: DeliveryModeWebhook, WebhookURL: unreachableURL}

	err := send(t, notifier, backupFailedNotification())

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "secret-key")
}

func Test_Send_WithBotModeWhenServerUnreachable_KeepsServerURLInError(t *testing.T) {
	serverURL, _ := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK},
	)
	notifier := &MattermostNotifier{
		DeliveryMode:    DeliveryModeBot,
		ServerURL:       serverURL + "-gone",
		BotToken:        "bot-token",
		TargetChannelID: strings.Repeat("c", channelIDLength),
	}

	err := send(t, notifier, backupFailedNotification())

	require.Error(t, err)
	assert.Contains(t, err.Error(), serverURL+"-gone")
}

func Test_Send_WhenErrorBodyExceedsTheReadLimit_TruncatesIt(t *testing.T) {
	serverURL, _ := notifier_models.StartRecordingServer(t, notifier_models.StubResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       strings.Repeat("x", 10*maxErrorBodyBytes),
	})
	notifier := &MattermostNotifier{DeliveryMode: DeliveryModeWebhook, WebhookURL: serverURL}

	err := send(t, notifier, backupFailedNotification())

	require.Error(t, err)
	assert.NotContains(t, err.Error(), strings.Repeat("x", maxErrorBodyBytes+1))
}

func Test_Validate_WithUnknownDeliveryMode_ReturnsError(t *testing.T) {
	notifier := &MattermostNotifier{DeliveryMode: "CARRIER_PIGEON", WebhookURL: "https://mm.example.com/hooks/key"}

	require.ErrorContains(t, notifier.Validate(notifier_models.PassthroughEncryptor{}), "delivery mode")
}

func Test_Validate_WithWebhookModeAndValidURL_ReturnsNoError(t *testing.T) {
	notifier := &MattermostNotifier{
		DeliveryMode: DeliveryModeWebhook,
		WebhookURL:   "https://mm.example.com/hooks/abc123",
	}

	require.NoError(t, notifier.Validate(notifier_models.PassthroughEncryptor{}))
}

func Test_Validate_WithWebhookModeAndMissingURL_ReturnsError(t *testing.T) {
	notifier := &MattermostNotifier{DeliveryMode: DeliveryModeWebhook}

	require.ErrorContains(t,
		notifier.Validate(notifier_models.PassthroughEncryptor{}),
		"incoming webhook URL is required",
	)
}

func Test_Validate_WithWebhookModeAndNonHTTPURL_ReturnsError(t *testing.T) {
	notifier := &MattermostNotifier{DeliveryMode: DeliveryModeWebhook, WebhookURL: "mm.example.com/hooks/abc"}

	require.ErrorContains(t,
		notifier.Validate(notifier_models.PassthroughEncryptor{}),
		"http or https",
	)
}

func Test_Validate_WithBotModeAndFullCredentials_ReturnsNoError(t *testing.T) {
	notifier := &MattermostNotifier{
		DeliveryMode:    DeliveryModeBot,
		ServerURL:       "https://mm.example.com",
		BotToken:        "bot-token",
		TargetChannelID: strings.Repeat("c", channelIDLength),
	}

	require.NoError(t, notifier.Validate(notifier_models.PassthroughEncryptor{}))
}

func Test_Validate_WithBotModeAndMissingToken_ReturnsError(t *testing.T) {
	notifier := &MattermostNotifier{
		DeliveryMode:    DeliveryModeBot,
		ServerURL:       "https://mm.example.com",
		TargetChannelID: strings.Repeat("c", channelIDLength),
	}

	require.ErrorContains(t,
		notifier.Validate(notifier_models.PassthroughEncryptor{}),
		"bot token is required",
	)
}

func Test_Validate_WhenChannelIDIsAChannelName_ReturnsError(t *testing.T) {
	notifier := &MattermostNotifier{
		DeliveryMode:    DeliveryModeBot,
		ServerURL:       "https://mm.example.com",
		BotToken:        "bot-token",
		TargetChannelID: "town-square",
	}

	require.ErrorContains(t,
		notifier.Validate(notifier_models.PassthroughEncryptor{}),
		"not the channel name",
	)
}

func Test_Validate_WithNonHTTPOverrideIconURL_ReturnsError(t *testing.T) {
	notifier := &MattermostNotifier{
		DeliveryMode:    DeliveryModeWebhook,
		WebhookURL:      "https://mm.example.com/hooks/abc123",
		OverrideIconURL: "javascript:alert(1)",
	}

	require.ErrorContains(t,
		notifier.Validate(notifier_models.PassthroughEncryptor{}),
		"override icon URL",
	)
}

func Test_HideSensitiveData_BlanksBothCredentials(t *testing.T) {
	notifier := &MattermostNotifier{
		DeliveryMode:    DeliveryModeBot,
		WebhookURL:      "https://mm.example.com/hooks/abc123",
		BotToken:        "bot-token",
		ServerURL:       "https://mm.example.com",
		TargetChannelID: strings.Repeat("c", channelIDLength),
	}

	notifier.HideSensitiveData()

	assert.Empty(t, notifier.WebhookURL)
	assert.Empty(t, notifier.BotToken)
	assert.Equal(t, "https://mm.example.com", notifier.ServerURL)
}

func Test_Update_WithBlankCredentials_KeepsStoredOnes(t *testing.T) {
	storedNotifier := &MattermostNotifier{
		DeliveryMode:    DeliveryModeBot,
		ServerURL:       "https://mm.example.com",
		BotToken:        "stored-token",
		WebhookURL:      "https://mm.example.com/hooks/stored",
		TargetChannelID: strings.Repeat("c", channelIDLength),
	}

	storedNotifier.Update(&MattermostNotifier{
		DeliveryMode:     DeliveryModeBot,
		ServerURL:        "https://mm.internal.example.com",
		TargetChannelID:  strings.Repeat("d", channelIDLength),
		OverrideUsername: "Databasus",
	})

	assert.Equal(t, "stored-token", storedNotifier.BotToken)
	assert.Equal(t, "https://mm.example.com/hooks/stored", storedNotifier.WebhookURL)
	assert.Equal(t, "https://mm.internal.example.com", storedNotifier.ServerURL)
	assert.Equal(t, strings.Repeat("d", channelIDLength), storedNotifier.TargetChannelID)
	assert.Equal(t, "Databasus", storedNotifier.OverrideUsername)
}
