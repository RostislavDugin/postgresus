package google_drive_storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"postgresus-backend/internal/util/encryption"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	gdConnectTimeout      = 30 * time.Second
	gdResponseTimeout     = 30 * time.Second
	gdIdleConnTimeout     = 90 * time.Second
	gdTLSHandshakeTimeout = 30 * time.Second

	// Chunk size for Google Drive resumable uploads - 16MB provides good balance
	// between memory usage and upload efficiency. Google Drive requires chunks
	// to be multiples of 256KB for resumable uploads.
	gdChunkSize = 16 * 1024 * 1024
)

type GoogleDriveStorage struct {
	StorageID    uuid.UUID `json:"storageId"    gorm:"primaryKey;type:uuid;column:storage_id"`
	ClientID     string    `json:"clientId"     gorm:"not null;type:text;column:client_id"`
	ClientSecret string    `json:"clientSecret" gorm:"not null;type:text;column:client_secret"`
	TokenJSON    string    `json:"tokenJson"    gorm:"not null;type:text;column:token_json"`
}

func (s *GoogleDriveStorage) TableName() string {
	return "google_drive_storages"
}

func (s *GoogleDriveStorage) SaveFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileID uuid.UUID,
	file io.Reader,
) error {
	return s.withRetryOnAuth(ctx, encryptor, func(driveService *drive.Service) error {
		filename := fileID.String()

		folderID, err := s.ensureBackupsFolderExists(ctx, driveService)
		if err != nil {
			return fmt.Errorf("failed to create/find backups folder: %w", err)
		}

		_ = s.deleteByName(ctx, driveService, filename, folderID)

		fileMeta := &drive.File{
			Name:    filename,
			Parents: []string{folderID},
		}

		backpressureReader := &backpressureReader{
			reader:    file,
			ctx:       ctx,
			chunkSize: gdChunkSize,
			buf:       make([]byte, gdChunkSize),
		}

		_, err = driveService.Files.Create(fileMeta).
			Media(backpressureReader, googleapi.ChunkSize(gdChunkSize)).
			Context(ctx).
			Do()
		if err != nil {
			select {
			case <-ctx.Done():
				return fmt.Errorf("upload cancelled: %w", ctx.Err())
			default:
			}
			return fmt.Errorf("failed to upload file to Google Drive: %w", err)
		}

		logger.Info(
			"file uploaded to Google Drive",
			"name",
			filename,
			"folder",
			"postgresus_backups",
		)
		return nil
	})
}

type backpressureReader struct {
	reader     io.Reader
	ctx        context.Context
	chunkSize  int
	buf        []byte
	bufStart   int
	bufEnd     int
	totalBytes int64
	chunkCount int
}

func (r *backpressureReader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
	}

	if r.bufStart >= r.bufEnd {
		r.chunkCount++

		bytesRead, readErr := io.ReadFull(r.reader, r.buf)
		if bytesRead > 0 {
			r.bufStart = 0
			r.bufEnd = bytesRead
		}

		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return 0, readErr
		}

		if bytesRead == 0 && readErr == io.EOF {
			return 0, io.EOF
		}
	}

	n = copy(p, r.buf[r.bufStart:r.bufEnd])
	r.bufStart += n
	r.totalBytes += int64(n)

	if r.bufStart >= r.bufEnd {
		select {
		case <-r.ctx.Done():
			return n, r.ctx.Err()
		default:
		}
	}

	return n, nil
}

func (s *GoogleDriveStorage) GetFile(
	encryptor encryption.FieldEncryptor,
	fileID uuid.UUID,
) (io.ReadCloser, error) {
	var result io.ReadCloser
	err := s.withRetryOnAuth(
		context.Background(),
		encryptor,
		func(driveService *drive.Service) error {
			folderID, err := s.findBackupsFolder(driveService)
			if err != nil {
				return fmt.Errorf("failed to find backups folder: %w", err)
			}

			fileIDGoogle, err := s.lookupFileID(driveService, fileID.String(), folderID)
			if err != nil {
				return err
			}

			resp, err := driveService.Files.Get(fileIDGoogle).Download()
			if err != nil {
				return fmt.Errorf("failed to download file from Google Drive: %w", err)
			}

			result = resp.Body
			return nil
		},
	)

	return result, err
}

func (s *GoogleDriveStorage) DeleteFile(
	encryptor encryption.FieldEncryptor,
	fileID uuid.UUID,
) error {
	ctx := context.Background()
	return s.withRetryOnAuth(ctx, encryptor, func(driveService *drive.Service) error {
		folderID, err := s.findBackupsFolder(driveService)
		if err != nil {
			return fmt.Errorf("failed to find backups folder: %w", err)
		}

		return s.deleteByName(ctx, driveService, fileID.String(), folderID)
	})
}

func (s *GoogleDriveStorage) Validate(encryptor encryption.FieldEncryptor) error {
	switch {
	case s.ClientID == "":
		return errors.New("client ID is required")
	case s.ClientSecret == "":
		return errors.New("client secret is required")
	case s.TokenJSON == "":
		return errors.New("token JSON is required")
	}

	// Skip JSON validation if token is already encrypted
	if strings.HasPrefix(s.TokenJSON, "enc:") {
		return nil
	}

	// Validate that the token JSON contains a refresh token
	var token oauth2.Token
	if err := json.Unmarshal([]byte(s.TokenJSON), &token); err != nil {
		return fmt.Errorf("invalid token JSON format: %w", err)
	}

	if token.RefreshToken == "" {
		return errors.New("token JSON must contain a refresh token for automatic token refresh")
	}

	return nil
}

func (s *GoogleDriveStorage) TestConnection(encryptor encryption.FieldEncryptor) error {
	ctx := context.Background()
	return s.withRetryOnAuth(ctx, encryptor, func(driveService *drive.Service) error {
		testFilename := "test-connection-" + uuid.New().String()
		testData := []byte("test")

		// Ensure the postgresus_backups folder exists
		folderID, err := s.ensureBackupsFolderExists(ctx, driveService)
		if err != nil {
			return fmt.Errorf("failed to create/find backups folder: %w", err)
		}

		// Test write operation
		fileMeta := &drive.File{
			Name:    testFilename,
			Parents: []string{folderID},
		}
		file, err := driveService.Files.Create(fileMeta).
			Media(strings.NewReader(string(testData))).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("failed to write test file to Google Drive: %w", err)
		}

		// Test read operation
		resp, err := driveService.Files.Get(file.Id).Download()
		if err != nil {
			// Clean up test file before returning error
			_ = driveService.Files.Delete(file.Id).Context(ctx).Do()
			return fmt.Errorf("failed to read test file from Google Drive: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("failed to close response body: %v\n", err)
			}
		}()

		readData, err := io.ReadAll(resp.Body)
		if err != nil {
			// Clean up test file before returning error
			_ = driveService.Files.Delete(file.Id).Context(ctx).Do()
			return fmt.Errorf("failed to read test file data: %w", err)
		}

		// Clean up test file
		if err := driveService.Files.Delete(file.Id).Context(ctx).Do(); err != nil {
			return fmt.Errorf("failed to clean up test file: %w", err)
		}

		// Verify data matches
		if string(readData) != string(testData) {
			return fmt.Errorf(
				"test file data mismatch: expected %q, got %q",
				string(testData),
				string(readData),
			)
		}

		return nil
	})
}

func (s *GoogleDriveStorage) HideSensitiveData() {
	s.ClientSecret = ""
	s.TokenJSON = ""
}

func (s *GoogleDriveStorage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	var err error

	if s.ClientSecret != "" {
		s.ClientSecret, err = encryptor.Encrypt(s.StorageID, s.ClientSecret)
		if err != nil {
			return fmt.Errorf("failed to encrypt Google Drive client secret: %w", err)
		}
	}

	if s.TokenJSON != "" {
		s.TokenJSON, err = encryptor.Encrypt(s.StorageID, s.TokenJSON)
		if err != nil {
			return fmt.Errorf("failed to encrypt Google Drive token JSON: %w", err)
		}
	}

	return nil
}

func (s *GoogleDriveStorage) Update(incoming *GoogleDriveStorage) {
	s.ClientID = incoming.ClientID

	if incoming.ClientSecret != "" {
		s.ClientSecret = incoming.ClientSecret
	}

	if incoming.TokenJSON != "" {
		s.TokenJSON = incoming.TokenJSON
	}
}

// withRetryOnAuth executes the provided function with retry logic for authentication errors
func (s *GoogleDriveStorage) withRetryOnAuth(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	fn func(*drive.Service) error,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	driveService, err := s.getDriveService(encryptor)
	if err != nil {
		return err
	}

	err = fn(driveService)
	if err != nil && s.isAuthError(err) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to refresh token and retry once
		fmt.Printf("Google Drive auth error detected, attempting token refresh: %v\n", err)

		if refreshErr := s.refreshToken(encryptor); refreshErr != nil {
			// If refresh fails, return a more helpful error message
			if strings.Contains(refreshErr.Error(), "invalid_grant") ||
				strings.Contains(refreshErr.Error(), "refresh token") {
				return fmt.Errorf(
					"google drive refresh token has expired. Please re-authenticate and update your token configuration. Original error: %w. Refresh error: %v",
					err,
					refreshErr,
				)
			}

			return fmt.Errorf("failed to refresh token after auth error: %w", refreshErr)
		}

		fmt.Printf("Token refresh successful, retrying operation\n")

		// Get new service with refreshed token
		driveService, err = s.getDriveService(encryptor)
		if err != nil {
			return fmt.Errorf("failed to create service after token refresh: %w", err)
		}

		// Retry the operation
		err = fn(driveService)
		if err != nil {
			fmt.Printf("Retry after token refresh also failed: %v\n", err)
		} else {
			fmt.Printf("Operation succeeded after token refresh\n")
		}
	}

	return err
}

// isAuthError checks if the error is a 401 authentication error
func (s *GoogleDriveStorage) isAuthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "Invalid Credentials") ||
		strings.Contains(errStr, "authError") ||
		strings.Contains(errStr, "invalid authentication credentials")
}

// refreshToken refreshes the OAuth2 token and updates the TokenJSON field
func (s *GoogleDriveStorage) refreshToken(encryptor encryption.FieldEncryptor) error {
	if err := s.Validate(encryptor); err != nil {
		return err
	}

	// Decrypt credentials before use
	clientSecret, err := encryptor.Decrypt(s.StorageID, s.ClientSecret)
	if err != nil {
		return fmt.Errorf("failed to decrypt Google Drive client secret: %w", err)
	}

	tokenJSON, err := encryptor.Decrypt(s.StorageID, s.TokenJSON)
	if err != nil {
		return fmt.Errorf("failed to decrypt Google Drive token JSON: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		return fmt.Errorf("invalid token JSON: %w", err)
	}

	// Check if we have a refresh token
	if token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available in stored token")
	}

	fmt.Printf("Original token - Access Token: %s..., Refresh Token: %s..., Expiry: %v\n",
		truncateString(token.AccessToken, 20),
		truncateString(token.RefreshToken, 20),
		token.Expiry)

	// Debug: Print the full token JSON structure (sensitive data masked)
	fmt.Printf("Original token JSON structure: %s\n", maskSensitiveData(tokenJSON))

	ctx := context.Background()
	cfg := &oauth2.Config{
		ClientID:     s.ClientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"https://www.googleapis.com/auth/drive.file"},
	}

	// Force the token to be expired so refresh is guaranteed
	token.Expiry = time.Now().Add(-time.Hour)
	fmt.Printf("Forcing token expiry to trigger refresh: %v\n", token.Expiry)

	tokenSource := cfg.TokenSource(ctx, &token)

	// Force token refresh
	fmt.Printf("Attempting to refresh Google Drive token...\n")
	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	fmt.Printf("New token - Access Token: %s..., Refresh Token: %s..., Expiry: %v\n",
		truncateString(newToken.AccessToken, 20),
		truncateString(newToken.RefreshToken, 20),
		newToken.Expiry)

	// Check if we actually got a new token
	if newToken.AccessToken == token.AccessToken {
		return fmt.Errorf(
			"token refresh did not return a new access token - this indicates the refresh token may be invalid",
		)
	}

	// Ensure the new token has a refresh token (preserve the original if not returned)
	if newToken.RefreshToken == "" {
		fmt.Printf("New token doesn't have refresh token, preserving original\n")
		newToken.RefreshToken = token.RefreshToken
	}

	// Update the stored token JSON (keep as plaintext in memory, encryption happens on save)
	newTokenJSON, err := json.Marshal(newToken)
	if err != nil {
		return fmt.Errorf("failed to marshal refreshed token: %w", err)
	}

	s.TokenJSON = string(newTokenJSON)
	fmt.Printf("Token refresh completed successfully with new access token\n")
	return nil
}

// maskSensitiveData masks sensitive information in token JSON for logging
func maskSensitiveData(tokenJSON string) string {
	// Replace sensitive values with masked versions
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(tokenJSON), &data); err != nil {
		return "invalid JSON"
	}

	if accessToken, ok := data["access_token"].(string); ok && len(accessToken) > 10 {
		data["access_token"] = accessToken[:10] + "..."
	}
	if refreshToken, ok := data["refresh_token"].(string); ok && len(refreshToken) > 10 {
		data["refresh_token"] = refreshToken[:10] + "..."
	}

	masked, _ := json.Marshal(data)
	return string(masked)
}

// truncateString safely truncates a string for logging purposes
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func (s *GoogleDriveStorage) getDriveService(
	encryptor encryption.FieldEncryptor,
) (*drive.Service, error) {
	if err := s.Validate(encryptor); err != nil {
		return nil, err
	}

	clientSecret, err := encryptor.Decrypt(s.StorageID, s.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Google Drive client secret: %w", err)
	}

	tokenJSON, err := encryptor.Decrypt(s.StorageID, s.TokenJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Google Drive token JSON: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		return nil, fmt.Errorf("invalid token JSON: %w", err)
	}

	ctx := context.Background()

	cfg := &oauth2.Config{
		ClientID:     s.ClientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"https://www.googleapis.com/auth/drive.file"},
	}

	tokenSource := cfg.TokenSource(ctx, &token)

	currentToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get current token: %w", err)
	}

	validatedTokenSource := oauth2.StaticTokenSource(currentToken)

	httpClient := s.buildHTTPClient(validatedTokenSource)

	driveService, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("unable to create Drive client: %w", err)
	}

	return driveService, nil
}

func (s *GoogleDriveStorage) buildHTTPClient(tokenSource oauth2.TokenSource) *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: gdConnectTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   gdTLSHandshakeTimeout,
		ResponseHeaderTimeout: gdResponseTimeout,
		IdleConnTimeout:       gdIdleConnTimeout,
	}

	return &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
			Base:   transport,
		},
	}
}

func (s *GoogleDriveStorage) lookupFileID(
	driveService *drive.Service,
	name string,
	folderID string,
) (string, error) {
	query := fmt.Sprintf(
		"name = '%s' and trashed = false and '%s' in parents",
		escapeForQuery(name),
		folderID,
	)

	results, err := driveService.Files.List().
		Q(query).
		Fields("files(id)").
		PageSize(1).
		Do()
	if err != nil {
		return "", fmt.Errorf("file lookup failed: %w", err)
	}

	if len(results.Files) == 0 {
		return "", fmt.Errorf("file %q not found in Google Drive backups folder", name)
	}

	return results.Files[0].Id, nil
}

func (s *GoogleDriveStorage) deleteByName(
	ctx context.Context,
	driveService *drive.Service,
	name string,
	folderID string,
) error {
	query := fmt.Sprintf(
		"name = '%s' and trashed = false and '%s' in parents",
		escapeForQuery(name),
		folderID,
	)

	err := driveService.
		Files.
		List().
		Q(query).
		Fields("files(id)").
		Pages(ctx, func(p *drive.FileList) error {
			for _, file := range p.Files {
				if err := driveService.Files.Delete(file.Id).Context(ctx).Do(); err != nil {
					return err
				}
			}

			return nil
		})

	if err != nil {
		return fmt.Errorf("failed to delete %q: %w", name, err)
	}

	return nil
}

func escapeForQuery(s string) string {
	return strings.ReplaceAll(s, `'`, `\'`)
}

// ensureBackupsFolderExists creates the postgresus_backups folder if it doesn't exist
func (s *GoogleDriveStorage) ensureBackupsFolderExists(
	ctx context.Context,
	driveService *drive.Service,
) (string, error) {
	folderID, err := s.findBackupsFolder(driveService)
	if err == nil {
		return folderID, nil
	}

	// Folder doesn't exist, create it
	folderMeta := &drive.File{
		Name:     "postgresus_backups",
		MimeType: "application/vnd.google-apps.folder",
	}

	folder, err := driveService.Files.Create(folderMeta).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create postgresus_backups folder: %w", err)
	}

	return folder.Id, nil
}

// findBackupsFolder finds the postgresus_backups folder ID
func (s *GoogleDriveStorage) findBackupsFolder(driveService *drive.Service) (string, error) {
	query := "name = 'postgresus_backups' and mimeType = 'application/vnd.google-apps.folder' and trashed = false"

	results, err := driveService.Files.List().
		Q(query).
		Fields("files(id)").
		PageSize(1).
		Do()
	if err != nil {
		return "", fmt.Errorf("failed to search for backups folder: %w", err)
	}

	if len(results.Files) == 0 {
		return "", fmt.Errorf("postgresus_backups folder not found")
	}

	return results.Files[0].Id, nil
}
