package s3_storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"postgresus-backend/internal/util/encryption"
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

	S3Prefix                string `json:"s3Prefix"                gorm:"type:text;column:s3_prefix"`
	S3UseVirtualHostedStyle bool   `json:"s3UseVirtualHostedStyle" gorm:"default:false;column:s3_use_virtual_hosted_style"`
}

func (s *S3Storage) TableName() string {
	return "s3_storages"
}

func (s *S3Storage) SaveFile(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileID uuid.UUID,
	file io.Reader,
) error {
	client, err := s.getClient(encryptor)
	if err != nil {
		return err
	}

	objectKey := s.buildObjectKey(fileID.String())

	// Upload the file using MinIO client with streaming (size = -1 for unknown size)
	_, err = client.PutObject(
		context.TODO(),
		s.S3Bucket,
		objectKey,
		file,
		-1,
		minio.PutObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

func (s *S3Storage) GetFile(
	encryptor encryption.FieldEncryptor,
	fileID uuid.UUID,
) (io.ReadCloser, error) {
	client, err := s.getClient(encryptor)
	if err != nil {
		return nil, err
	}

	objectKey := s.buildObjectKey(fileID.String())

	object, err := client.GetObject(
		context.TODO(),
		s.S3Bucket,
		objectKey,
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

func (s *S3Storage) DeleteFile(encryptor encryption.FieldEncryptor, fileID uuid.UUID) error {
	client, err := s.getClient(encryptor)
	if err != nil {
		return err
	}

	objectKey := s.buildObjectKey(fileID.String())

	// Delete the object using MinIO client
	err = client.RemoveObject(
		context.TODO(),
		s.S3Bucket,
		objectKey,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

func (s *S3Storage) Validate(encryptor encryption.FieldEncryptor) error {
	if s.S3Bucket == "" {
		return errors.New("S3 bucket is required")
	}
	if s.S3AccessKey == "" {
		return errors.New("S3 access key is required")
	}
	if s.S3SecretKey == "" {
		return errors.New("S3 secret key is required")
	}

	return nil
}

func (s *S3Storage) TestConnection(encryptor encryption.FieldEncryptor) error {
	client, err := s.getClient(encryptor)
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
	testObjectKey := s.buildObjectKey(testFileID)
	testData := []byte("test connection")
	testReader := bytes.NewReader(testData)

	// Upload test file
	_, err = client.PutObject(
		ctx,
		s.S3Bucket,
		testObjectKey,
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
		testObjectKey,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to delete test file from S3: %w", err)
	}

	return nil
}

func (s *S3Storage) HideSensitiveData() {
	s.S3AccessKey = ""
	s.S3SecretKey = ""
}

func (s *S3Storage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	var err error

	if s.S3AccessKey != "" {
		s.S3AccessKey, err = encryptor.Encrypt(s.StorageID, s.S3AccessKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt S3 access key: %w", err)
		}
	}

	if s.S3SecretKey != "" {
		s.S3SecretKey, err = encryptor.Encrypt(s.StorageID, s.S3SecretKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt S3 secret key: %w", err)
		}
	}

	return nil
}

func (s *S3Storage) Update(incoming *S3Storage) {
	s.S3Bucket = incoming.S3Bucket
	s.S3Region = incoming.S3Region
	s.S3Endpoint = incoming.S3Endpoint
	s.S3UseVirtualHostedStyle = incoming.S3UseVirtualHostedStyle

	if incoming.S3AccessKey != "" {
		s.S3AccessKey = incoming.S3AccessKey
	}

	if incoming.S3SecretKey != "" {
		s.S3SecretKey = incoming.S3SecretKey
	}

	// we do not allow to change the prefix after creation,
	// otherwise we will have to migrate all the data to the new prefix
}

func (s *S3Storage) buildObjectKey(fileName string) string {
	if s.S3Prefix == "" {
		return fileName
	}

	prefix := s.S3Prefix
	prefix = strings.TrimPrefix(prefix, "/")

	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	return prefix + fileName
}

func (s *S3Storage) getClient(encryptor encryption.FieldEncryptor) (*minio.Client, error) {
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

	// Decrypt credentials before use
	accessKey, err := encryptor.Decrypt(s.StorageID, s.S3AccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt S3 access key: %w", err)
	}

	secretKey, err := encryptor.Decrypt(s.StorageID, s.S3SecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt S3 secret key: %w", err)
	}

	// Configure bucket lookup strategy
	bucketLookup := minio.BucketLookupAuto
	if s.S3UseVirtualHostedStyle {
		bucketLookup = minio.BucketLookupDNS
	}

	// Initialize the MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       useSSL,
		Region:       s.S3Region,
		BucketLookup: bucketLookup,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	return minioClient, nil
}
