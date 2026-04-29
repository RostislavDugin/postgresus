package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DatabaseRequest struct {
	ID          string `json:"id,omitempty"`
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	Type        string `json:"type"`

	Postgresql *PostgresqlRequest `json:"postgresql,omitempty"`
	Mysql      *MysqlRequest      `json:"mysql,omitempty"`
	Mariadb    *MariadbRequest    `json:"mariadb,omitempty"`
	Mongodb    *MongodbRequest    `json:"mongodb,omitempty"`

	Notifiers []NotifierRef `json:"notifiers,omitempty"`
}

type NotifierRef struct {
	ID string `json:"id"`
}

type PostgresqlRequest struct {
	Version        string   `json:"version"`
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	Username       string   `json:"username"`
	Password       string   `json:"password"`
	Database       *string  `json:"database,omitempty"`
	IsHttps        bool     `json:"isHttps"`
	BackupType     string   `json:"backupType"`
	IncludeSchemas []string `json:"includeSchemas,omitempty"`
	CpuCount       int      `json:"cpuCount"`
}

type MysqlRequest struct {
	Version  string `json:"version"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database,omitempty"`
	IsHttps  bool   `json:"isHttps"`
}

type MariadbRequest struct {
	Version  string `json:"version"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database,omitempty"`
	IsHttps  bool   `json:"isHttps"`
}

type MongodbRequest struct {
	Version            string `json:"version"`
	Host               string `json:"host"`
	Port               *int   `json:"port,omitempty"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	Database           string `json:"database"`
	AuthDatabase       string `json:"authDatabase,omitempty"`
	IsHttps            bool   `json:"isHttps"`
	IsSrv              bool   `json:"isSrv"`
	IsDirectConnection bool   `json:"isDirectConnection"`
	CpuCount           int    `json:"cpuCount"`
}

type DatabaseResponse struct {
	ID                     string     `json:"id"`
	WorkspaceID            *string    `json:"workspaceId"`
	Name                   string     `json:"name"`
	Type                   string     `json:"type"`
	HealthStatus           *string    `json:"healthStatus"`
	LastBackupTime         *time.Time `json:"lastBackupTime"`
	LastBackupErrorMessage *string    `json:"lastBackupErrorMessage"`
}

func (c *DatabasusClient) CreateDatabase(ctx context.Context, req *DatabaseRequest) (*DatabaseResponse, error) {
	if req.WorkspaceID == "" {
		req.WorkspaceID = c.workspaceID
	}

	body, statusCode, err := c.do(ctx, http.MethodPost, "/api/v1/databases/create", req)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusCreated && statusCode != http.StatusOK {
		return nil, parseError(statusCode, body)
	}

	var resp DatabaseResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal database response: %w", err)
	}

	return &resp, nil
}

func (c *DatabasusClient) UpdateDatabase(ctx context.Context, req *DatabaseRequest) (*DatabaseResponse, error) {
	body, statusCode, err := c.do(ctx, http.MethodPost, "/api/v1/databases/update", req)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, parseError(statusCode, body)
	}

	var resp DatabaseResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal database response: %w", err)
	}

	return &resp, nil
}

func (c *DatabasusClient) GetDatabase(ctx context.Context, databaseID string) (*DatabaseResponse, error) {
	body, statusCode, err := c.do(ctx, http.MethodGet, "/api/v1/databases/"+databaseID, nil)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusNotFound {
		return nil, nil
	}

	if statusCode != http.StatusOK {
		return nil, parseError(statusCode, body)
	}

	var resp DatabaseResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal database response: %w", err)
	}

	return &resp, nil
}

func (c *DatabasusClient) DeleteDatabase(ctx context.Context, databaseID string) error {
	body, statusCode, err := c.do(ctx, http.MethodDelete, "/api/v1/databases/"+databaseID, nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
		return parseError(statusCode, body)
	}

	return nil
}
