package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	chainValidPath      = "/api/v1/backups/postgres/wal/is-wal-chain-valid-since-last-full-backup"
	nextBackupTimePath  = "/api/v1/backups/postgres/wal/next-full-backup-time"
	walUploadPath       = "/api/v1/backups/postgres/wal/upload/wal"
	fullStartPath       = "/api/v1/backups/postgres/wal/upload/full-start"
	fullCompletePath    = "/api/v1/backups/postgres/wal/upload/full-complete"
	reportErrorPath     = "/api/v1/backups/postgres/wal/error"
	restorePlanPath     = "/api/v1/backups/postgres/wal/restore/plan"
	restoreDownloadPath = "/api/v1/backups/postgres/wal/restore/download"
	versionPath         = "/api/v1/system/version"
	agentBinaryPath     = "/api/v1/system/agent"

	apiCallTimeout   = 30 * time.Second
	maxRetryAttempts = 3
	retryBaseDelay   = 1 * time.Second
)

// For stream uploads (basebackup and WAL segments) the standard resty client is not used,
// because it buffers the entire body in memory before sending.
type Client struct {
	json       *resty.Client
	streamHTTP *http.Client
	host       string
	token      string
	log        *slog.Logger
}

func NewClient(host, token string, isInsecureSkipVerify bool, log *slog.Logger) *Client {
	setAuth := func(_ *resty.Client, req *resty.Request) error {
		if token != "" {
			req.SetHeader("Authorization", token)
		}

		return nil
	}

	jsonClient := resty.New().
		SetTimeout(apiCallTimeout).
		SetRetryCount(maxRetryAttempts - 1).
		SetRetryWaitTime(retryBaseDelay).
		SetRetryMaxWaitTime(4 * retryBaseDelay).
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			return err != nil || resp.StatusCode() >= 500
		}).
		OnBeforeRequest(setAuth)

	if isInsecureSkipVerify {
		tlsConfig := &tls.Config{InsecureSkipVerify: true} //nolint:gosec // user opt-in for self-signed servers
		jsonClient.SetTLSClientConfig(tlsConfig)

		if log != nil {
			log.Warn("TLS certificate verification is disabled for Databasus API; use only with self-signed or non-production servers")
		}
	}

	streamHTTP := &http.Client{}
	if isInsecureSkipVerify {
		transport, isOk := http.DefaultTransport.(*http.Transport)
		if isOk {
			cloned := transport.Clone()
			cloned.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
			streamHTTP.Transport = cloned
		} else {
			streamHTTP.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			}
		}
	}

	return &Client{
		json:       jsonClient,
		streamHTTP: streamHTTP,
		host:       host,
		token:      token,
		log:        log,
	}
}

func (c *Client) CheckWalChainValidity(ctx context.Context) (*WalChainValidityResponse, error) {
	var resp WalChainValidityResponse

	httpResp, err := c.json.R().
		SetContext(ctx).
		SetResult(&resp).
		Get(c.buildURL(chainValidPath))
	if err != nil {
		return nil, err
	}

	if err := c.checkResponse(httpResp, "check WAL chain validity"); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) GetNextFullBackupTime(ctx context.Context) (*NextFullBackupTimeResponse, error) {
	var resp NextFullBackupTimeResponse

	httpResp, err := c.json.R().
		SetContext(ctx).
		SetResult(&resp).
		Get(c.buildURL(nextBackupTimePath))
	if err != nil {
		return nil, err
	}

	if err := c.checkResponse(httpResp, "get next full backup time"); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) ReportBackupError(ctx context.Context, errMsg string) error {
	httpResp, err := c.json.R().
		SetContext(ctx).
		SetBody(reportErrorRequest{Error: errMsg}).
		Post(c.buildURL(reportErrorPath))
	if err != nil {
		return err
	}

	return c.checkResponse(httpResp, "report backup error")
}

func (c *Client) UploadBasebackup(
	ctx context.Context,
	body io.Reader,
) (*UploadBasebackupResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildURL(fullStartPath), body)
	if err != nil {
		return nil, fmt.Errorf("create upload request: %w", err)
	}

	c.setStreamHeaders(req)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.streamHTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result UploadBasebackupResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode upload response: %w", err)
	}

	return &result, nil
}

func (c *Client) FinalizeBasebackup(
	ctx context.Context,
	backupID string,
	startSegment string,
	stopSegment string,
) error {
	resp, err := c.json.R().
		SetContext(ctx).
		SetBody(finalizeBasebackupRequest{
			BackupID:     backupID,
			StartSegment: startSegment,
			StopSegment:  stopSegment,
		}).
		Post(c.buildURL(fullCompletePath))
	if err != nil {
		return fmt.Errorf("finalize request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("finalize failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

func (c *Client) FinalizeBasebackupWithError(
	ctx context.Context,
	backupID string,
	errMsg string,
) error {
	resp, err := c.json.R().
		SetContext(ctx).
		SetBody(finalizeBasebackupRequest{
			BackupID: backupID,
			Error:    &errMsg,
		}).
		Post(c.buildURL(fullCompletePath))
	if err != nil {
		return fmt.Errorf("finalize-with-error request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("finalize-with-error failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

func (c *Client) UploadWalSegment(
	ctx context.Context,
	segmentName string,
	body io.Reader,
) (*UploadWalSegmentResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildURL(walUploadPath), body)
	if err != nil {
		return nil, fmt.Errorf("create WAL upload request: %w", err)
	}

	c.setStreamHeaders(req)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Wal-Segment-Name", segmentName)

	resp, err := c.streamHTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return &UploadWalSegmentResult{IsGapDetected: false}, nil

	case http.StatusConflict:
		var errResp uploadErrorResponse

		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return &UploadWalSegmentResult{IsGapDetected: true}, nil
		}

		return &UploadWalSegmentResult{
			IsGapDetected:       true,
			ExpectedSegmentName: errResp.ExpectedSegmentName,
			ReceivedSegmentName: errResp.ReceivedSegmentName,
		}, nil

	default:
		respBody, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}
}

func (c *Client) GetRestorePlan(
	ctx context.Context,
	backupID string,
) (*GetRestorePlanResponse, *GetRestorePlanErrorResponse, error) {
	request := c.json.R().SetContext(ctx)

	if backupID != "" {
		request.SetQueryParam("backupId", backupID)
	}

	httpResp, err := request.Get(c.buildURL(restorePlanPath))
	if err != nil {
		return nil, nil, fmt.Errorf("get restore plan: %w", err)
	}

	switch httpResp.StatusCode() {
	case http.StatusOK:
		var response GetRestorePlanResponse
		if err := json.Unmarshal(httpResp.Body(), &response); err != nil {
			return nil, nil, fmt.Errorf("decode restore plan response: %w", err)
		}

		return &response, nil, nil

	case http.StatusBadRequest:
		var errorResponse GetRestorePlanErrorResponse
		if err := json.Unmarshal(httpResp.Body(), &errorResponse); err != nil {
			return nil, nil, fmt.Errorf("decode restore plan error: %w", err)
		}

		return nil, &errorResponse, nil

	default:
		return nil, nil, fmt.Errorf("get restore plan: server returned status %d: %s",
			httpResp.StatusCode(), httpResp.String())
	}
}

func (c *Client) DownloadBackupFile(
	ctx context.Context,
	backupID string,
) (io.ReadCloser, error) {
	requestURL := c.buildURL(restoreDownloadPath) + "?" + url.Values{"backupId": {backupID}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	c.setStreamHeaders(req)

	resp, err := c.streamHTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download backup file: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		return nil, fmt.Errorf("download backup file: server returned status %d: %s",
			resp.StatusCode, string(respBody))
	}

	return resp.Body, nil
}

func (c *Client) FetchServerVersion(ctx context.Context) (string, error) {
	var ver versionResponse

	httpResp, err := c.json.R().
		SetContext(ctx).
		SetResult(&ver).
		Get(c.buildURL(versionPath))
	if err != nil {
		return "", err
	}

	if err := c.checkResponse(httpResp, "fetch server version"); err != nil {
		return "", err
	}

	return ver.Version, nil
}

func (c *Client) DownloadAgentBinary(ctx context.Context, arch, destPath string) error {
	requestURL := c.buildURL(agentBinaryPath) + "?" + url.Values{"arch": {arch}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("create agent download request: %w", err)
	}

	c.setStreamHeaders(req)

	resp, err := c.streamHTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d for agent download", resp.StatusCode)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, resp.Body)

	return err
}

func (c *Client) buildURL(path string) string {
	return c.host + path
}

func (c *Client) checkResponse(resp *resty.Response, method string) error {
	if resp.StatusCode() >= 400 {
		return fmt.Errorf("%s: server returned status %d: %s", method, resp.StatusCode(), resp.String())
	}

	return nil
}

func (c *Client) setStreamHeaders(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", c.token)
	}
}
