package httpclient

import (
	"net/http"
	"time"
)

const DefaultTimeout = 30 * time.Second

// NewClient creates an HTTP client with proxy support.
// It reads proxy configuration from HTTP_PROXY, HTTPS_PROXY, and NO_PROXY
// environment variables automatically via http.ProxyFromEnvironment.
func NewClient() *http.Client {
	return NewClientWithTimeout(DefaultTimeout)
}

// NewClientWithTimeout creates an HTTP client with a custom timeout and proxy support.
func NewClientWithTimeout(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
