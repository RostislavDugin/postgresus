package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TelemetrySender interface {
	Send(ctx context.Context, req *CollectRequest) error
}

type HTTPTelemetrySender struct {
	endpoint   string
	appVersion string
	httpClient *http.Client
}

func NewHTTPTelemetrySender(endpoint, appVersion string) *HTTPTelemetrySender {
	return &HTTPTelemetrySender{
		endpoint:   endpoint,
		appVersion: appVersion,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *HTTPTelemetrySender) Send(ctx context.Context, req *CollectRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode telemetry request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, s.endpoint, bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to build telemetry request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Databasus-Telemetry/"+s.appVersion)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("telemetry request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Drain a small slice so the connection can be reused.
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("telemetry endpoint returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
