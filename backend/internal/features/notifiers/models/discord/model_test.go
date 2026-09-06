package discord_notifier

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

func Test_Send_WhenContentExceedsDiscordLimit_TruncatesContentByRunes(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusNoContent},
	)
	notifier := &DiscordNotifier{ChannelWebhookURL: webhookURL}
	heading := "Backup failed"

	require.NoError(t, notifier.Send(
		notifier_models.PassthroughEncryptor{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		notifier_models.Notification{
			Type:    notifier_models.NotificationTypeBackupFailed,
			Heading: heading,
			Message: strings.Repeat("界", maxContentRunes),
		},
	))

	require.Equal(t, 1, recorder.GetRequestCount())
	request := recorder.GetLastRequest()
	assert.Equal(t, http.MethodPost, request.Method)
	assert.Equal(t, "application/json", request.Headers.Get("Content-Type"))

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(request.Body), &payload))

	messagePrefixLength := maxContentRunes - len([]rune(heading)) - len([]rune("\n\n"))
	truncatedContent := heading + "\n\n" + strings.Repeat("界", messagePrefixLength)
	assert.Equal(t, truncatedContent, payload["content"])
}
