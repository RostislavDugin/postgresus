package telegram_notifier

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"databasus-backend/internal/util/encryption"
)

func parseProxyURL(rawProxyURL string) (*url.URL, error) {
	parsedProxyURL, err := url.Parse(rawProxyURL)
	if err != nil || !ProxyScheme(parsedProxyURL.Scheme).IsAllowed() || parsedProxyURL.Host == "" {
		return nil, errors.New("proxy URL must be a valid http/https/socks5/socks5h URL")
	}

	return parsedProxyURL, nil
}

func (t *TelegramNotifier) buildHTTPClient(
	encryptor encryption.FieldEncryptor,
) (*http.Client, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if !t.IsProxyEnabled {
		return client, nil
	}

	proxyURL, err := encryptor.Decrypt(t.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt proxy URL: %w", err)
	}

	parsedProxyURL, err := parseProxyURL(proxyURL)
	if err != nil {
		return nil, err
	}

	defaultTransport, isTransport := http.DefaultTransport.(*http.Transport)
	if !isTransport {
		return nil, errors.New("unexpected default HTTP transport type")
	}

	transport := defaultTransport.Clone()
	transport.Proxy = http.ProxyURL(parsedProxyURL)
	client.Transport = transport

	return client, nil
}
