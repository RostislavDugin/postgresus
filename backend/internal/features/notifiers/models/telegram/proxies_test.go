package telegram_notifier

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_BuildHTTPClient_WhenProxyEnabled_UsesConfiguredProxy(t *testing.T) {
	for _, proxyURL := range []string{
		"http://user:password@proxy.example.com:3128",
		"https://proxy.example.com:8443",
		"socks5://user:password@proxy.example.com:1080",
		"socks5h://proxy.example.com:1080",
	} {
		notifier := &TelegramNotifier{
			IsProxyEnabled: true,
			ProxyURL:       proxyURL,
		}

		client, err := notifier.buildHTTPClient(passthroughEncryptor{})
		require.NoError(t, err)

		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, transport.Proxy)

		req, err := http.NewRequest(http.MethodGet, "https://api.telegram.org", nil)
		require.NoError(t, err)

		resolvedProxyURL, err := transport.Proxy(req)
		require.NoError(t, err)
		require.NotNil(t, resolvedProxyURL)
		assert.Equal(t, proxyURL, resolvedProxyURL.String())
	}
}

func Test_BuildHTTPClient_WhenProxyDisabled_UsesDefaultTransport(t *testing.T) {
	notifier := &TelegramNotifier{}

	client, err := notifier.buildHTTPClient(passthroughEncryptor{})

	require.NoError(t, err)
	assert.Nil(t, client.Transport)
}
