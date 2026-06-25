package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-verification-agent/internal/testutil"
)

func instantAfter(t *testing.T) {
	t.Helper()

	original := timeAfterFn
	timeAfterFn = func(time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()

		return ch
	}

	t.Cleanup(func() { timeAfterFn = original })
}

func Test_ClaimVerification_WhenServer200_SendsNestedMaxRamMbAndDeserializes(t *testing.T) {
	var capturedPath, capturedAuth string
	var capturedBody map[string]any

	verificationID := uuid.New()
	backupID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"verificationId":     verificationID.String(),
			"backupId":           backupID.String(),
			"backupSizeMb":       120.5,
			"maxContainerDiskMb": 800.25,
			"database": map[string]any{
				"type":              "POSTGRES_LOGICAL",
				"postgresqlLogical": map[string]any{"version": "16"},
			},
		})
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "tok123", "agent-9", testutil.DiscardLogger())

	assignment, err := client.ClaimVerification(t.Context(), AgentCapacity{
		MaxCPU: 8, MaxRAMMb: 4096, MaxDiskGb: 100, MaxConcurrentJobs: 4,
	})

	require.NoError(t, err)
	require.NotNil(t, assignment)

	assert.Equal(t, "/api/v1/agent/verifications/agent-9/claim", capturedPath)
	assert.Equal(t, "Bearer tok123", capturedAuth)

	capacity, ok := capturedBody["capacity"].(map[string]any)
	require.True(t, ok, "claim body must nest capacity under \"capacity\"")
	assert.Equal(t, float64(4096), capacity["maxRamMb"])
	_, hasRamGb := capacity["maxRamGb"]
	assert.False(t, hasRamGb, "claim envelope must use maxRamMb, never maxRamGb")

	assert.Equal(t, verificationID, assignment.VerificationID)
	assert.Equal(t, backupID, assignment.BackupID)
	assert.InDelta(t, 120.5, assignment.BackupSizeMb, 0.001)
	assert.InDelta(t, 800.25, assignment.MaxContainerDiskMb, 0.001)
	assert.Equal(t, "POSTGRES_LOGICAL", assignment.Database.Type)
	require.NotNil(t, assignment.Database.PostgresqlLogical)
	assert.Equal(t, "16", assignment.Database.PostgresqlLogical.Version)
}

func Test_ClaimVerification_WhenServer204_ReturnsNilNoError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "t", "a", testutil.DiscardLogger())

	assignment, err := client.ClaimVerification(t.Context(), AgentCapacity{})

	require.NoError(t, err)
	assert.Nil(t, assignment)
}

func Test_ClaimVerification_WhenServer500_ReturnsRetryableResponseError(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "t", "a", testutil.DiscardLogger())

	_, err := client.ClaimVerification(t.Context(), AgentCapacity{})

	require.Error(t, err)
	assert.Equal(t, int32(1), calls.Load(), "claim must not retry internally — the loop owns retry")

	var respErr *ResponseError
	require.True(t, errors.As(err, &respErr))
	assert.True(t, respErr.Retryable())
}

func Test_DownloadBackup_WhenServer200_ReturnsBody(t *testing.T) {
	payload := []byte("PGDMP fake custom-format archive")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t,
			"/api/v1/agent/verifications/agent-x/"+
				"22222222-2222-2222-2222-222222222222/backup-stream",
			r.URL.Path)
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "", "agent-x", testutil.DiscardLogger())

	body, err := client.DownloadBackup(
		t.Context(), uuid.MustParse("22222222-2222-2222-2222-222222222222"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = body.Close() })

	read, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, payload, read)
}

func Test_DownloadBackup_WhenServer410_ReturnsGoneResponseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte(`{"reason":"gone"}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "", "a", testutil.DiscardLogger())

	_, err := client.DownloadBackup(t.Context(), uuid.New())

	require.Error(t, err)
	var respErr *ResponseError
	require.True(t, errors.As(err, &respErr))
	assert.True(t, respErr.IsGone())
	assert.False(t, respErr.Retryable())
}

func Test_Report_WhenServer204_Succeeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/agent/verifications/a/33333333-3333-3333-3333-333333333333/report",
			r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "", "a", testutil.DiscardLogger())

	err := client.Report(t.Context(),
		uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		ReportRequest{Status: VerificationStatusCompleted})

	require.NoError(t, err)
}

func Test_Report_WhenServer410_ReturnsErrReportGoneWithoutRetry(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusGone)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "", "a", testutil.DiscardLogger())

	err := client.Report(t.Context(), uuid.New(), ReportRequest{Status: VerificationStatusFailed})

	require.ErrorIs(t, err, ErrReportGone)
	assert.Equal(t, int32(1), calls.Load(), "410 must not be retried")
}

func Test_Report_When5xxThenRecovers_RetriesThenSucceeds(t *testing.T) {
	instantAfter(t)

	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "", "a", testutil.DiscardLogger())

	err := client.Report(t.Context(), uuid.New(), ReportRequest{Status: VerificationStatusCompleted})

	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func Test_Report_When5xxAndBudgetExhausted_ReturnsBudgetError(t *testing.T) {
	originalBudget := reportRetryBudget
	reportRetryBudget = 25 * time.Millisecond
	t.Cleanup(func() { reportRetryBudget = originalBudget })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "", "a", testutil.DiscardLogger())

	err := client.Report(t.Context(), uuid.New(), ReportRequest{Status: VerificationStatusFailed})

	require.ErrorIs(t, err, ErrReportBudgetExhausted)
}

func Test_Constants_SatisfyInvariants(t *testing.T) {
	// reportRetryBudget must stay a short, bounded window: once it is exhausted
	// the backend reclaims the run on the agent's next heartbeat, so retrying
	// for minutes would be pointless. 5m is a generous sanity ceiling.
	assert.Less(t, reportRetryBudget, 5*time.Minute)

	assert.Equal(t, maxBackoff, reportRetryDelay(99), "report backoff must cap at maxBackoff")
	assert.Equal(t, 1*time.Second, reportRetryDelay(1))
	assert.Equal(t, 2*time.Second, reportRetryDelay(2))
}

func Test_Heartbeat_WhenCalled_SendsFlatEnvelopeWithBearerAndAgentPath(t *testing.T) {
	var capturedPath, capturedMethod, capturedAuth string
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		capturedAuth = r.Header.Get("Authorization")

		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"lastSeenAt":           time.Now().UTC().Format(time.RFC3339),
			"abortVerificationIds": []string{},
		})
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "tok123", "11111111-1111-1111-1111-111111111111", testutil.DiscardLogger())

	response, err := client.Heartbeat(t.Context(), HeartbeatRequest{
		MaxCPU:                 8,
		MaxRAMGb:               4,
		MaxDiskGb:              100,
		MaxConcurrentJobs:      4,
		CurrentVerificationIDs: []uuid.UUID{},
	})

	require.NoError(t, err)
	require.NotNil(t, response)

	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t,
		"/api/v1/agent/verification/11111111-1111-1111-1111-111111111111/heartbeat",
		capturedPath)
	assert.Equal(t, "Bearer tok123", capturedAuth)

	assert.Equal(t, float64(8), capturedBody["maxCpu"])
	assert.Equal(t, float64(4), capturedBody["maxRamGb"])
	assert.Equal(t, float64(100), capturedBody["maxDiskGb"])
	assert.Equal(t, float64(4), capturedBody["maxConcurrentJobs"])

	_, hasRamMb := capturedBody["maxRamMb"]
	assert.False(t, hasRamMb, "heartbeat envelope must use maxRamGb, never maxRamMb")

	require.Contains(t, capturedBody, "currentVerificationIds")
	assert.Equal(t, []any{}, capturedBody["currentVerificationIds"],
		"currentVerificationIds must serialize as [] not null")
}

func Test_Heartbeat_WhenServerReturns4xx_ReturnsErrorWithoutRetry(t *testing.T) {
	var calls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid agent credentials"}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "bad", "agent-1", testutil.DiscardLogger())

	_, err := client.Heartbeat(t.Context(), HeartbeatRequest{CurrentVerificationIDs: []uuid.UUID{}})

	require.Error(t, err)
	assert.Equal(t, 1, calls, "4xx must not be retried")
}

func Test_FetchServerVersion_WhenServerResponds_ReturnsVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/system/version", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v3.27.0"})
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "", "", testutil.DiscardLogger())

	version, err := client.FetchServerVersion(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "v3.27.0", version)
}

func Test_DownloadVerificationAgentBinary_WhenServerServesFile_WritesItToDisk(t *testing.T) {
	payload := []byte("\x7fELF fake verification agent")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/system/verification-agent", r.URL.Path)
		assert.Equal(t, "arm64", r.URL.Query().Get("arch"))
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "", "", testutil.DiscardLogger())
	destPath := filepath.Join(t.TempDir(), "agent")

	err := client.DownloadVerificationAgentBinary(t.Context(), "arm64", destPath)
	require.NoError(t, err)

	written, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, payload, written)
}

func Test_DownloadVerificationAgentBinary_WhenServer404_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "", "", testutil.DiscardLogger())

	err := client.DownloadVerificationAgentBinary(t.Context(), "amd64", filepath.Join(t.TempDir(), "a"))

	require.Error(t, err)
}
