package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type StorageRequest struct {
	ID          string      `json:"id,omitempty"`
	WorkspaceID string      `json:"workspaceId"`
	Type        string      `json:"type"`
	Name        string      `json:"name"`
	IsSystem    bool        `json:"isSystem"`
	S3Storage   *S3Request  `json:"s3Storage,omitempty"`
	SFTPStorage *SFTPRequest `json:"sftpStorage,omitempty"`
	// Additional storage types can be added as needed.
}

type S3Request struct {
	S3Bucket                string `json:"s3Bucket"`
	S3Region                string `json:"s3Region"`
	S3AccessKey             string `json:"s3AccessKey"`
	S3SecretKey             string `json:"s3SecretKey"`
	S3Endpoint              string `json:"s3Endpoint,omitempty"`
	S3Prefix                string `json:"s3Prefix,omitempty"`
	S3UseVirtualHostedStyle bool   `json:"s3UseVirtualHostedStyle"`
	SkipTLSVerify           bool   `json:"skipTLSVerify"`
	S3StorageClass          string `json:"s3StorageClass,omitempty"`
}

type SFTPRequest struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	Username          string `json:"username"`
	Password          string `json:"password,omitempty"`
	PrivateKey        string `json:"privateKey,omitempty"`
	Path              string `json:"path,omitempty"`
	IsSkipHostKeyVerify bool `json:"isSkipHostKeyVerify"`
}

type StorageResponse struct {
	ID            string  `json:"id"`
	WorkspaceID   string  `json:"workspaceId"`
	Type          string  `json:"type"`
	Name          string  `json:"name"`
	LastSaveError *string `json:"lastSaveError"`
}

func (c *DatabasusClient) SaveStorage(ctx context.Context, req *StorageRequest) (*StorageResponse, error) {
	if req.WorkspaceID == "" {
		req.WorkspaceID = c.workspaceID
	}

	body, statusCode, err := c.do(ctx, http.MethodPost, "/api/v1/storages", req)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, parseError(statusCode, body)
	}

	var resp StorageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage response: %w", err)
	}

	return &resp, nil
}

func (c *DatabasusClient) GetStorage(ctx context.Context, storageID string) (*StorageResponse, error) {
	body, statusCode, err := c.do(ctx, http.MethodGet, "/api/v1/storages/"+storageID, nil)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusNotFound {
		return nil, nil
	}

	if statusCode != http.StatusOK {
		return nil, parseError(statusCode, body)
	}

	var resp StorageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage response: %w", err)
	}

	return &resp, nil
}

func (c *DatabasusClient) DeleteStorage(ctx context.Context, storageID string) error {
	body, statusCode, err := c.do(ctx, http.MethodDelete, "/api/v1/storages/"+storageID, nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
		return parseError(statusCode, body)
	}

	return nil
}
