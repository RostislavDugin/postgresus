package telemetry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Send_PostsCorrectBodyAndHeaders(t *testing.T) {
	var captured CollectRequest
	var captureMethod string
	var captureContentType string
	var captureUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captureMethod = r.Method
		captureContentType = r.Header.Get("Content-Type")
		captureUserAgent = r.Header.Get("User-Agent")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &captured))

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"accepted"}`))
	}))
	defer server.Close()

	sender := NewHTTPTelemetrySender(server.URL, "9.9.9")
	req := &CollectRequest{
		InstanceID:  "550e8400-e29b-41d4-a716-446655440000",
		AppVersion:  "9.9.9",
		OS:          "linux",
		Arch:        "amd64",
		InstalledAt: "2026-04-29",
		Databases:   []DatabaseEntry{{Type: "POSTGRES_LOGICAL", Version: "16"}},
		Storages:    []string{"S3"},
		Notifiers:   []string{"EMAIL"},
	}

	require.NoError(t, sender.Send(context.Background(), req))

	assert.Equal(t, http.MethodPost, captureMethod)
	assert.Equal(t, "application/json", captureContentType)
	assert.True(t, strings.HasPrefix(captureUserAgent, "Databasus-Telemetry/"))
	assert.Contains(t, captureUserAgent, "9.9.9")
	assert.Equal(t, *req, captured)
}

func Test_Send_WhenServerReturns202_ReturnsNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sender := NewHTTPTelemetrySender(server.URL, "1.0.0")
	assert.NoError(t, sender.Send(context.Background(), &CollectRequest{}))
}

func Test_Send_WhenServerReturns500_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer server.Close()

	sender := NewHTTPTelemetrySender(server.URL, "1.0.0")
	err := sender.Send(context.Background(), &CollectRequest{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func Test_Send_WhenEndpointUnreachable_ReturnsError(t *testing.T) {
	sender := NewHTTPTelemetrySender("http://127.0.0.1:1/never", "1.0.0")
	err := sender.Send(context.Background(), &CollectRequest{})
	require.Error(t, err)
}
