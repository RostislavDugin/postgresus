package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CheckServerAlive_WhenServerReturns200_ReturnsNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/system/version", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := checkServerAlive(server.URL)

	assert.NoError(t, err)
}

func Test_CheckServerAlive_WhenServerReturnsNon200_ReturnsErrorWithStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := checkServerAlive(server.URL)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func Test_CheckServerAlive_WhenServerUnreachable_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	unreachableURL := server.URL
	server.Close()

	err := checkServerAlive(unreachableURL)

	assert.Error(t, err)
}
