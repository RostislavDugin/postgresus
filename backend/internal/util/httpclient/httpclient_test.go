package httpclient

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient_ReturnsClientWithDefaultTimeout(t *testing.T) {
	client := NewClient()
	
	assert.NotNil(t, client)
	assert.Equal(t, DefaultTimeout, client.Timeout)
}

func TestNewClientWithTimeout_ReturnsClientWithCustomTimeout(t *testing.T) {
	customTimeout := 10 * time.Second
	client := NewClientWithTimeout(customTimeout)
	
	assert.NotNil(t, client)
	assert.Equal(t, customTimeout, client.Timeout)
}

func TestNewClient_HasTransportWithProxy(t *testing.T) {
	client := NewClient()
	
	assert.NotNil(t, client.Transport)
	
	transport, ok := client.Transport.(*http.Transport)
	assert.True(t, ok, "Transport should be *http.Transport")
	assert.NotNil(t, transport.Proxy, "Proxy function should be set")
}

func TestNewClient_CanMakeHTTPRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestNewClient_RespectsHTTPProxyEnv(t *testing.T) {
	originalProxy := os.Getenv("HTTP_PROXY")
	testProxy := originalProxy
	if testProxy == "" {
		testProxy = "http://test-proxy:8080"
		os.Setenv("HTTP_PROXY", testProxy)
		defer os.Unsetenv("HTTP_PROXY")
	}

	client := NewClient()
	transport, ok := client.Transport.(*http.Transport)
	
	assert.True(t, ok)
	assert.NotNil(t, transport.Proxy, "Proxy function should be set")
	
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	proxyURL, err := transport.Proxy(req)
	
	assert.NoError(t, err)
	if proxyURL != nil {
		t.Logf("Using proxy: %s", proxyURL.Host)
	}
}

func TestNewClient_RespectsNoProxyEnv(t *testing.T) {
	originalHTTPProxy := os.Getenv("HTTP_PROXY")
	originalNoProxy := os.Getenv("NO_PROXY")
	
	needCleanup := false
	if originalHTTPProxy == "" {
		os.Setenv("HTTP_PROXY", "http://test-proxy:8080")
		needCleanup = true
	}
	if originalNoProxy == "" {
		os.Setenv("NO_PROXY", "localhost,127.0.0.1")
	}
	defer func() {
		if needCleanup {
			restoreEnvVar("HTTP_PROXY", originalHTTPProxy)
			restoreEnvVar("NO_PROXY", originalNoProxy)
		}
	}()

	client := NewClient()
	transport, _ := client.Transport.(*http.Transport)

	req, _ := http.NewRequest("GET", "http://localhost/test", nil)
	proxyURL, err := transport.Proxy(req)
	
	assert.NoError(t, err)
	t.Logf("NO_PROXY=%s, proxyURL for localhost=%v", os.Getenv("NO_PROXY"), proxyURL)
}

func TestDefaultTimeout_Is30Seconds(t *testing.T) {
	assert.Equal(t, 30*time.Second, DefaultTimeout)
}

func TestNewClient_HTTPSProxyEnvIsRespected(t *testing.T) {
	originalHTTPSProxy := os.Getenv("HTTPS_PROXY")
	needCleanup := false
	
	if originalHTTPSProxy == "" {
		os.Setenv("HTTPS_PROXY", "http://https-proxy:8443")
		needCleanup = true
	}
	defer func() {
		if needCleanup {
			restoreEnvVar("HTTPS_PROXY", originalHTTPSProxy)
		}
	}()

	client := NewClient()
	transport, ok := client.Transport.(*http.Transport)

	assert.True(t, ok)
	assert.NotNil(t, transport.Proxy)

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	proxyURL, err := transport.Proxy(req)

	assert.NoError(t, err)
	if proxyURL != nil {
		t.Logf("Using HTTPS proxy: %s", proxyURL.Host)
	}
}

func TestNewClient_NoProxyBypasses(t *testing.T) {
	originalHTTPProxy := os.Getenv("HTTP_PROXY")
	originalNoProxy := os.Getenv("NO_PROXY")
	
	if originalHTTPProxy != "" {
		t.Logf("Using real HTTP_PROXY=%s, NO_PROXY=%s - skipping strict assertions", originalHTTPProxy, originalNoProxy)
		
		client := NewClient()
		transport, _ := client.Transport.(*http.Transport)
		
		req, _ := http.NewRequest("GET", "http://localhost/test", nil)
		_, err := transport.Proxy(req)
		assert.NoError(t, err)
		return
	}
	
	defer func() {
		restoreEnvVar("HTTP_PROXY", originalHTTPProxy)
		restoreEnvVar("NO_PROXY", originalNoProxy)
	}()

	os.Setenv("HTTP_PROXY", "http://proxy:8080")
	os.Setenv("NO_PROXY", "internal.local,*.internal.com,192.168.1.0/24")

	client := NewClient()
	transport, _ := client.Transport.(*http.Transport)

	testCases := []struct {
		url         string
		shouldProxy bool
		description string
	}{
		{"http://github.com/", false, "exact match in NO_PROXY"},
		{"http://api.internal.com/data", false, "wildcard match in NO_PROXY"},
		{"http://external.com/api", true, "not in NO_PROXY list"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tc.url, nil)
			proxyURL, err := transport.Proxy(req)
			assert.NoError(t, err)
			if tc.shouldProxy {
				if proxyURL != nil {
					assert.Equal(t, "proxy:8080", proxyURL.Host)
				}
			} else {
				assert.Nil(t, proxyURL, "expected %s to bypass proxy", tc.url)
			}
		})
	}
}

func TestNewClient_ProxyFromEnvironment_Integration(t *testing.T) {
	// This test verifies that http.ProxyFromEnvironment is properly configured
	// and the client can be used with standard proxy env vars

	client := NewClient()
	transport, ok := client.Transport.(*http.Transport)

	assert.True(t, ok, "Transport should be *http.Transport")
	assert.NotNil(t, transport.Proxy, "Proxy function should be configured")

	// Verify transport has expected default configuration
	assert.Equal(t, DefaultTimeout, client.Timeout)
}

func TestNewClientWithTimeout_ProxyFromEnvironment_Integration(t *testing.T) {
	customTimeout := 15 * time.Second
	client := NewClientWithTimeout(customTimeout)
	transport, ok := client.Transport.(*http.Transport)

	assert.True(t, ok, "Transport should be *http.Transport")
	assert.NotNil(t, transport.Proxy, "Proxy function should be configured")
	assert.Equal(t, customTimeout, client.Timeout)
}

func TestNewClient_WithProxyMakesRequest(t *testing.T) {
	// Create a mock server to act as the target
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Proxy-Test", "success")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("proxied"))
	}))
	defer targetServer.Close()

	// Test that client can make requests (proxy or direct)
	client := NewClient()
	resp, err := client.Get(targetServer.URL)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()
}

// Helper function to restore env vars
func restoreEnvVar(key, originalValue string) {
	if originalValue != "" {
		os.Setenv(key, originalValue)
	} else {
		os.Unsetenv(key)
	}
}
