package s3_storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Storage struct {
	StorageID   uuid.UUID `json:"storageId"   gorm:"primaryKey;type:uuid;column:storage_id"`
	S3Bucket    string    `json:"s3Bucket"    gorm:"not null;type:text;column:s3_bucket"`
	S3Region    string    `json:"s3Region"    gorm:"not null;type:text;column:s3_region"`
	S3AccessKey string    `json:"s3AccessKey" gorm:"not null;type:text;column:s3_access_key"`
	S3SecretKey string    `json:"s3SecretKey" gorm:"not null;type:text;column:s3_secret_key"`
	S3Endpoint  string    `json:"s3Endpoint"  gorm:"type:text;column:s3_endpoint"`
}

func (s *S3Storage) TableName() string {
	return "s3_storages"
}

func (s *S3Storage) SaveFile(logger *slog.Logger, fileID uuid.UUID, file io.Reader) error {
	client, err := s.getClient()
	if err != nil {
		return err
	}

	// Upload the file using MinIO client with streaming (size = -1 for unknown size)
	_, err = client.PutObject(
		context.TODO(),
		s.S3Bucket,
		fileID.String(),
		file,
		-1,
		minio.PutObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

func (s *S3Storage) GetFile(fileID uuid.UUID) (io.ReadCloser, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	object, err := client.GetObject(
		context.TODO(),
		s.S3Bucket,
		fileID.String(),
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from S3: %w", err)
	}

	// Check if the file actually exists by reading the first byte
	buf := make([]byte, 1)
	_, readErr := object.Read(buf)
	if readErr != nil && readErr != io.EOF {
		_ = object.Close()
		return nil, fmt.Errorf("file does not exist in S3: %w", readErr)
	}

	// Reset the reader to the beginning
	_, seekErr := object.Seek(0, io.SeekStart)
	if seekErr != nil {
		_ = object.Close()
		return nil, fmt.Errorf("failed to reset file reader: %w", seekErr)
	}

	return object, nil
}

func (s *S3Storage) DeleteFile(fileID uuid.UUID) error {
	client, err := s.getClient()
	if err != nil {
		return err
	}

	// Delete the object using MinIO client
	err = client.RemoveObject(
		context.TODO(),
		s.S3Bucket,
		fileID.String(),
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

func (s *S3Storage) Validate() error {
	if s.S3Bucket == "" {
		return errors.New("S3 bucket is required")
	}
	if s.S3AccessKey == "" {
		return errors.New("S3 access key is required")
	}
	if s.S3SecretKey == "" {
		return errors.New("S3 secret key is required")
	}

	// Try to create a client to validate the configuration
	_, err := s.getClient()
	if err != nil {
		return fmt.Errorf("invalid S3 configuration: %w", err)
	}

	return nil
}

func (s *S3Storage) TestConnection() error {
	client, err := s.getClient()
	if err != nil {
		return err
	}

	// Create a context with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if the bucket exists to verify connection
	exists, err := client.BucketExists(ctx, s.S3Bucket)
	if err != nil {
		// Check if the error is due to context deadline exceeded
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.New("failed to connect to the bucket. Please check params")
		}
		return fmt.Errorf("failed to connect to S3: %w", err)
	}

	if !exists {
		return fmt.Errorf("bucket '%s' does not exist", s.S3Bucket)
	}

	// Test write and delete permissions by uploading and removing a small test file
	testFileID := uuid.New().String() + "-test"
	testData := []byte("test connection")
	testReader := bytes.NewReader(testData)

	// Upload test file
	_, err = client.PutObject(
		ctx,
		s.S3Bucket,
		testFileID,
		testReader,
		int64(len(testData)),
		minio.PutObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to upload test file to S3: %w", err)
	}

	// Delete test file
	err = client.RemoveObject(
		ctx,
		s.S3Bucket,
		testFileID,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to delete test file from S3: %w", err)
	}

	return nil
}

func (s *S3Storage) getClient() (*minio.Client, error) {
	endpoint := s.S3Endpoint
	useSSL := true

	if strings.HasPrefix(endpoint, "http://") {
		useSSL = false
		endpoint = strings.TrimPrefix(endpoint, "http://")
	} else if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.TrimPrefix(endpoint, "https://")
	}

	// If no endpoint is provided, use the AWS S3 endpoint for the region
	if endpoint == "" {
		endpoint = fmt.Sprintf("s3.%s.amazonaws.com", s.S3Region)
	}

	// Initialize the MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.S3AccessKey, s.S3SecretKey, ""),
		Secure: useSSL,
		Region: s.S3Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	return minioClient, nil
}
