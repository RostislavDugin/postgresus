package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type HealthcheckConfigRequest struct {
	DatabaseID                        string `json:"databaseId"`
	IsHealthcheckEnabled              bool   `json:"isHealthcheckEnabled"`
	IsSentNotificationWhenUnavailable bool   `json:"isSentNotificationWhenUnavailable"`
	IntervalMinutes                   int    `json:"intervalMinutes"`
	AttemptsBeforeConcideredAsDown    int    `json:"attemptsBeforeConcideredAsDown"`
	StoreAttemptsDays                 int    `json:"storeAttemptsDays"`
}

type HealthcheckConfigResponse struct {
	DatabaseID                        string `json:"databaseId"`
	IsHealthcheckEnabled              bool   `json:"isHealthcheckEnabled"`
	IsSentNotificationWhenUnavailable bool   `json:"isSentNotificationWhenUnavailable"`
	IntervalMinutes                   int    `json:"intervalMinutes"`
	AttemptsBeforeConcideredAsDown    int    `json:"attemptsBeforeConcideredAsDown"`
	StoreAttemptsDays                 int    `json:"storeAttemptsDays"`
}

func (c *DatabasusClient) SaveHealthcheckConfig(ctx context.Context, req *HealthcheckConfigRequest) (*HealthcheckConfigResponse, error) {
	body, statusCode, err := c.do(ctx, http.MethodPost, "/api/v1/healthcheck-config", req)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, parseError(statusCode, body)
	}

	var resp HealthcheckConfigResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal healthcheck config response: %w", err)
	}

	return &resp, nil
}

func (c *DatabasusClient) GetHealthcheckConfig(ctx context.Context, databaseID string) (*HealthcheckConfigResponse, error) {
	body, statusCode, err := c.do(ctx, http.MethodGet, "/api/v1/healthcheck-config/"+databaseID, nil)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusNotFound {
		return nil, nil
	}

	if statusCode != http.StatusOK {
		return nil, parseError(statusCode, body)
	}

	var resp HealthcheckConfigResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal healthcheck config response: %w", err)
	}

	return &resp, nil
}
