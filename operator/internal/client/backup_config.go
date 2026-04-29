package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type BackupConfigRequest struct {
	DatabaseID          string `json:"databaseId"`
	IsBackupsEnabled    bool   `json:"isBackupsEnabled"`
	RetentionPolicyType string `json:"retentionPolicyType"`
	RetentionTimePeriod string `json:"retentionTimePeriod,omitempty"`
	RetentionCount      int    `json:"retentionCount,omitempty"`
	RetentionGfsHours   int    `json:"retentionGfsHours,omitempty"`
	RetentionGfsDays    int    `json:"retentionGfsDays,omitempty"`
	RetentionGfsWeeks   int    `json:"retentionGfsWeeks,omitempty"`
	RetentionGfsMonths  int    `json:"retentionGfsMonths,omitempty"`
	RetentionGfsYears   int    `json:"retentionGfsYears,omitempty"`

	StorageID string `json:"storageId"`

	BackupInterval *IntervalRequest `json:"backupInterval"`

	SendNotificationsOn []string `json:"sendNotificationsOn"`
	IsRetryIfFailed     bool     `json:"isRetryIfFailed"`
	MaxFailedTriesCount int      `json:"maxFailedTriesCount"`
	Encryption          string   `json:"encryption"`
}

type IntervalRequest struct {
	ID             string  `json:"id,omitempty"`
	Interval       string  `json:"interval"`
	TimeOfDay      *string `json:"timeOfDay,omitempty"`
	Weekday        *int    `json:"weekday,omitempty"`
	DayOfMonth     *int    `json:"dayOfMonth,omitempty"`
	CronExpression *string `json:"cronExpression,omitempty"`
}

type BackupConfigResponse struct {
	DatabaseID       string `json:"databaseId"`
	IsBackupsEnabled bool   `json:"isBackupsEnabled"`
	StorageID        *string `json:"storageId"`
}

func (c *DatabasusClient) SaveBackupConfig(ctx context.Context, req *BackupConfigRequest) (*BackupConfigResponse, error) {
	body, statusCode, err := c.do(ctx, http.MethodPost, "/api/v1/backup-configs/save", req)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, parseError(statusCode, body)
	}

	var resp BackupConfigResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup config response: %w", err)
	}

	return &resp, nil
}

func (c *DatabasusClient) GetBackupConfig(ctx context.Context, databaseID string) (*BackupConfigResponse, error) {
	body, statusCode, err := c.do(ctx, http.MethodGet, "/api/v1/backup-configs/database/"+databaseID, nil)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusNotFound {
		return nil, nil
	}

	if statusCode != http.StatusOK {
		return nil, parseError(statusCode, body)
	}

	var resp BackupConfigResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup config response: %w", err)
	}

	return &resp, nil
}
