package azure_blob_storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"postgresus-backend/internal/util/encryption"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/google/uuid"
)

const (
	azureConnectTimeout      = 30 * time.Second
	azureResponseTimeout     = 30 * time.Second
	azureIdleConnTimeout     = 90 * time.Second
	azureTLSHandshakeTimeout = 30 * time.Second

	// Chunk size for block blob uploads - 16MB provides good balance between
	// memory usage and upload efficiency. This creates backpressure to pg_dump
	// by only reading one chunk at a time and waiting for Azure to confirm receipt.
	azureChunkSize = 16 * 1024 * 1024
)

type readSeekCloser struct {
	*bytes.Reader
}

func (r *readSeekCloser) Close() error {
	return nil
}

type AuthMethod string

const (
	AuthMethodConnectionString AuthMethod = "CONNECTION_STRING"
	AuthMethodAccountKey       AuthMethod = "ACCOUNT_KEY"
)

type AzureBlobStorage struct {
	StorageID        uuid.UUID  `json:"storageId"        gorm:"primaryKey;type:uuid;column:storage_id"`
	AuthMethod       AuthMethod `json:"authMethod"       gorm:"not null;type:text;column:auth_method"`
	ConnectionString string     `json:"connectionString" gorm:"type:text;column:connection_string"`
	AccountName      string     `json:"accountName"      gorm:"type:text;column:account_name"`
	AccountKey       string     `json:"accountKey"       gorm:"type:text;column:account_key"`
	ContainerName    string     `json:"containerName"    gorm:"not null;type:text;column:container_name"`
	Endpoint         string     `json:"endpoint"         gorm:"type:text;column:endpoint"`
	Prefix           string     `json:"prefix"           gorm:"type:text;column:prefix"`
}

func (s *AzureBlobStorage) TableName() string {
	return "azure_blob_storages"
}

func (s *AzureBlobStorage) SaveFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileID uuid.UUID,
	file io.Reader,
) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("upload cancelled before start: %w", ctx.Err())
	default:
	}

	client, err := s.getClient(encryptor)
	if err != nil {
		return err
	}

	blobName := s.buildBlobName(fileID.String())
	blockBlobClient := client.ServiceClient().
		NewContainerClient(s.ContainerName).
		NewBlockBlobClient(blobName)

	var blockIDs []string
	blockNumber := 0
	buf := make([]byte, azureChunkSize)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("upload cancelled: %w", ctx.Err())
		default:
		}

		n, readErr := io.ReadFull(file, buf)

		if n == 0 && readErr == io.EOF {
			break
		}

		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return fmt.Errorf("read error: %w", readErr)
		}

		blockID := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%06d", blockNumber)))

		_, err := blockBlobClient.StageBlock(
			ctx,
			blockID,
			&readSeekCloser{bytes.NewReader(buf[:n])},
			nil,
		)
		if err != nil {
			select {
			case <-ctx.Done():
				return fmt.Errorf("upload cancelled: %w", ctx.Err())
			default:
				return fmt.Errorf("failed to stage block %d: %w", blockNumber, err)
			}
		}

		blockIDs = append(blockIDs, blockID)
		blockNumber++

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	if len(blockIDs) == 0 {
		_, err = client.UploadStream(
			ctx,
			s.ContainerName,
			blobName,
			bytes.NewReader([]byte{}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to upload empty blob: %w", err)
		}
		return nil
	}

	_, err = blockBlobClient.CommitBlockList(ctx, blockIDs, &blockblob.CommitBlockListOptions{})
	if err != nil {
		return fmt.Errorf("failed to commit block list: %w", err)
	}

	return nil
}

func (s *AzureBlobStorage) GetFile(
	encryptor encryption.FieldEncryptor,
	fileID uuid.UUID,
) (io.ReadCloser, error) {
	client, err := s.getClient(encryptor)
	if err != nil {
		return nil, err
	}

	blobName := s.buildBlobName(fileID.String())

	response, err := client.DownloadStream(
		context.TODO(),
		s.ContainerName,
		blobName,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to download blob from Azure: %w", err)
	}

	return response.Body, nil
}

func (s *AzureBlobStorage) DeleteFile(encryptor encryption.FieldEncryptor, fileID uuid.UUID) error {
	client, err := s.getClient(encryptor)
	if err != nil {
		return err
	}

	blobName := s.buildBlobName(fileID.String())

	_, err = client.DeleteBlob(
		context.TODO(),
		s.ContainerName,
		blobName,
		nil,
	)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 404 {
			return nil
		}
		return fmt.Errorf("failed to delete blob from Azure: %w", err)
	}

	return nil
}

func (s *AzureBlobStorage) Validate(encryptor encryption.FieldEncryptor) error {
	if s.ContainerName == "" {
		return errors.New("container name is required")
	}

	switch s.AuthMethod {
	case AuthMethodConnectionString:
		if s.ConnectionString == "" {
			return errors.New(
				"connection string is required when using CONNECTION_STRING auth method",
			)
		}
	case AuthMethodAccountKey:
		if s.AccountName == "" {
			return errors.New("account name is required when using ACCOUNT_KEY auth method")
		}
		if s.AccountKey == "" {
			return errors.New("account key is required when using ACCOUNT_KEY auth method")
		}
	default:
		return fmt.Errorf("invalid auth method: %s", s.AuthMethod)
	}

	return nil
}

func (s *AzureBlobStorage) TestConnection(encryptor encryption.FieldEncryptor) error {
	client, err := s.getClient(encryptor)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containerClient := client.ServiceClient().NewContainerClient(s.ContainerName)
	_, err = containerClient.GetProperties(ctx, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) {
			if respErr.StatusCode == 404 {
				return fmt.Errorf("container '%s' does not exist", s.ContainerName)
			}
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.New("failed to connect to Azure Blob Storage. Please check params")
		}
		return fmt.Errorf("failed to connect to Azure Blob Storage: %w", err)
	}

	testBlobName := s.buildBlobName(uuid.New().String() + "-test")
	testData := []byte("test connection")

	_, err = client.UploadStream(
		ctx,
		s.ContainerName,
		testBlobName,
		bytes.NewReader(testData),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to upload test blob to Azure: %w", err)
	}

	_, err = client.DeleteBlob(
		ctx,
		s.ContainerName,
		testBlobName,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to delete test blob from Azure: %w", err)
	}

	return nil
}

func (s *AzureBlobStorage) HideSensitiveData() {
	s.ConnectionString = ""
	s.AccountKey = ""
}

func (s *AzureBlobStorage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	var err error

	if s.ConnectionString != "" {
		s.ConnectionString, err = encryptor.Encrypt(s.StorageID, s.ConnectionString)
		if err != nil {
			return fmt.Errorf("failed to encrypt Azure connection string: %w", err)
		}
	}

	if s.AccountKey != "" {
		s.AccountKey, err = encryptor.Encrypt(s.StorageID, s.AccountKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt Azure account key: %w", err)
		}
	}

	return nil
}

func (s *AzureBlobStorage) Update(incoming *AzureBlobStorage) {
	s.AuthMethod = incoming.AuthMethod
	s.ContainerName = incoming.ContainerName
	s.Endpoint = incoming.Endpoint

	if incoming.ConnectionString != "" {
		s.ConnectionString = incoming.ConnectionString
	}

	if incoming.AccountName != "" {
		s.AccountName = incoming.AccountName
	}

	if incoming.AccountKey != "" {
		s.AccountKey = incoming.AccountKey
	}
}

func (s *AzureBlobStorage) buildBlobName(fileName string) string {
	if s.Prefix == "" {
		return fileName
	}

	prefix := s.Prefix
	prefix = strings.TrimPrefix(prefix, "/")

	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	return prefix + fileName
}

func (s *AzureBlobStorage) getClient(encryptor encryption.FieldEncryptor) (*azblob.Client, error) {
	var client *azblob.Client
	var err error

	clientOptions := s.buildClientOptions()

	switch s.AuthMethod {
	case AuthMethodConnectionString:
		connectionString, decryptErr := encryptor.Decrypt(s.StorageID, s.ConnectionString)
		if decryptErr != nil {
			return nil, fmt.Errorf("failed to decrypt Azure connection string: %w", decryptErr)
		}

		client, err = azblob.NewClientFromConnectionString(connectionString, clientOptions)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create Azure Blob client from connection string: %w",
				err,
			)
		}
	case AuthMethodAccountKey:
		accountKey, decryptErr := encryptor.Decrypt(s.StorageID, s.AccountKey)
		if decryptErr != nil {
			return nil, fmt.Errorf("failed to decrypt Azure account key: %w", decryptErr)
		}

		accountURL := s.buildAccountURL()
		credential, credErr := azblob.NewSharedKeyCredential(s.AccountName, accountKey)
		if credErr != nil {
			return nil, fmt.Errorf("failed to create Azure shared key credential: %w", credErr)
		}

		client, err = azblob.NewClientWithSharedKeyCredential(accountURL, credential, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure Blob client with shared key: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", s.AuthMethod)
	}

	return client, nil
}

func (s *AzureBlobStorage) buildClientOptions() *azblob.ClientOptions {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: azureConnectTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   azureTLSHandshakeTimeout,
		ResponseHeaderTimeout: azureResponseTimeout,
		IdleConnTimeout:       azureIdleConnTimeout,
	}

	return &azblob.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: &http.Client{Transport: transport},
			Retry: policy.RetryOptions{
				MaxRetries: 0,
			},
		},
	}
}

func (s *AzureBlobStorage) buildAccountURL() string {
	if s.Endpoint != "" {
		endpoint := s.Endpoint
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = "https://" + endpoint
		}
		return endpoint
	}

	return fmt.Sprintf("https://%s.blob.core.windows.net/", s.AccountName)
}
