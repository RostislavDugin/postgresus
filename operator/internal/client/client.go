package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type DatabasusClient struct {
	httpClient  *http.Client
	baseURL     string
	token       string
	workspaceID string
}

type Config struct {
	BaseURL     string
	Token       string
	WorkspaceID string
}

func New(cfg Config) *DatabasusClient {
	return &DatabasusClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:     cfg.BaseURL,
		token:       cfg.Token,
		workspaceID: cfg.WorkspaceID,
	}
}

func (c *DatabasusClient) WorkspaceID() string {
	return c.workspaceID
}

func (c *DatabasusClient) do(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}

		bodyReader = bytes.NewReader(jsonBody)
	}

	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// APIError represents an error response from the databasus API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("databasus API error (status %d): %s", e.StatusCode, e.Message)
}

func parseError(statusCode int, body []byte) error {
	var errResp struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err != nil {
		return &APIError{StatusCode: statusCode, Message: string(body)}
	}

	return &APIError{StatusCode: statusCode, Message: errResp.Error}
}

// HealthCheck verifies connectivity to the databasus API.
func (c *DatabasusClient) HealthCheck(ctx context.Context) error {
	_, statusCode, err := c.do(ctx, http.MethodGet, "/api/v1/system/health", nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", statusCode)
	}

	return nil
}
