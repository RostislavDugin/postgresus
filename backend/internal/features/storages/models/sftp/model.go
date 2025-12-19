package sftp_storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"postgresus-backend/internal/util/encryption"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	sftpConnectTimeout     = 30 * time.Second
	sftpTestConnectTimeout = 10 * time.Second
)

type SFTPStorage struct {
	StorageID         uuid.UUID `json:"storageId"         gorm:"primaryKey;type:uuid;column:storage_id"`
	Host              string    `json:"host"              gorm:"not null;type:text;column:host"`
	Port              int       `json:"port"              gorm:"not null;default:22;column:port"`
	Username          string    `json:"username"          gorm:"not null;type:text;column:username"`
	Password          string    `json:"password"          gorm:"type:text;column:password"`
	PrivateKey        string    `json:"privateKey"        gorm:"type:text;column:private_key"`
	Path              string    `json:"path"              gorm:"type:text;column:path"`
	SkipHostKeyVerify bool      `json:"skipHostKeyVerify" gorm:"not null;default:false;column:skip_host_key_verify"`
}

func (s *SFTPStorage) TableName() string {
	return "sftp_storages"
}

func (s *SFTPStorage) SaveFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileID uuid.UUID,
	file io.Reader,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger.Info("Starting to save file to SFTP storage", "fileId", fileID.String(), "host", s.Host)

	client, sshConn, err := s.connect(encryptor, sftpConnectTimeout)
	if err != nil {
		logger.Error("Failed to connect to SFTP", "fileId", fileID.String(), "error", err)
		return fmt.Errorf("failed to connect to SFTP: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			logger.Error(
				"Failed to close SFTP client",
				"fileId",
				fileID.String(),
				"error",
				closeErr,
			)
		}
		if closeErr := sshConn.Close(); closeErr != nil {
			logger.Error(
				"Failed to close SSH connection",
				"fileId",
				fileID.String(),
				"error",
				closeErr,
			)
		}
	}()

	if s.Path != "" {
		if err := s.ensureDirectory(client, s.Path); err != nil {
			logger.Error(
				"Failed to ensure directory",
				"fileId",
				fileID.String(),
				"path",
				s.Path,
				"error",
				err,
			)
			return fmt.Errorf("failed to ensure directory: %w", err)
		}
	}

	filePath := s.getFilePath(fileID.String())
	logger.Debug("Uploading file to SFTP", "fileId", fileID.String(), "filePath", filePath)

	remoteFile, err := client.Create(filePath)
	if err != nil {
		logger.Error("Failed to create remote file", "fileId", fileID.String(), "error", err)
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer func() {
		_ = remoteFile.Close()
	}()

	ctxReader := &contextReader{ctx: ctx, reader: file}

	_, err = io.Copy(remoteFile, ctxReader)
	if err != nil {
		select {
		case <-ctx.Done():
			logger.Info("SFTP upload cancelled", "fileId", fileID.String())
			return ctx.Err()
		default:
			logger.Error("Failed to upload file to SFTP", "fileId", fileID.String(), "error", err)
			return fmt.Errorf("failed to upload file to SFTP: %w", err)
		}
	}

	logger.Info(
		"Successfully saved file to SFTP storage",
		"fileId",
		fileID.String(),
		"filePath",
		filePath,
	)
	return nil
}

func (s *SFTPStorage) GetFile(
	encryptor encryption.FieldEncryptor,
	fileID uuid.UUID,
) (io.ReadCloser, error) {
	client, sshConn, err := s.connect(encryptor, sftpConnectTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SFTP: %w", err)
	}

	filePath := s.getFilePath(fileID.String())

	remoteFile, err := client.Open(filePath)
	if err != nil {
		_ = client.Close()
		_ = sshConn.Close()
		return nil, fmt.Errorf("failed to open file from SFTP: %w", err)
	}

	return &sftpFileReader{
		file:    remoteFile,
		client:  client,
		sshConn: sshConn,
	}, nil
}

func (s *SFTPStorage) DeleteFile(encryptor encryption.FieldEncryptor, fileID uuid.UUID) error {
	client, sshConn, err := s.connect(encryptor, sftpConnectTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to SFTP: %w", err)
	}
	defer func() {
		_ = client.Close()
		_ = sshConn.Close()
	}()

	filePath := s.getFilePath(fileID.String())

	_, err = client.Stat(filePath)
	if err != nil {
		return nil
	}

	err = client.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file from SFTP: %w", err)
	}

	return nil
}

func (s *SFTPStorage) Validate(encryptor encryption.FieldEncryptor) error {
	if s.Host == "" {
		return errors.New("SFTP host is required")
	}
	if s.Username == "" {
		return errors.New("SFTP username is required")
	}
	if s.Password == "" && s.PrivateKey == "" {
		return errors.New("SFTP password or private key is required")
	}
	if s.Port <= 0 || s.Port > 65535 {
		return errors.New("SFTP port must be between 1 and 65535")
	}

	return nil
}

func (s *SFTPStorage) TestConnection(encryptor encryption.FieldEncryptor) error {
	ctx, cancel := context.WithTimeout(context.Background(), sftpTestConnectTimeout)
	defer cancel()

	client, sshConn, err := s.connectWithContext(ctx, encryptor, sftpTestConnectTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to SFTP: %w", err)
	}
	defer func() {
		_ = client.Close()
		_ = sshConn.Close()
	}()

	if s.Path != "" {
		if err := s.ensureDirectory(client, s.Path); err != nil {
			return fmt.Errorf("failed to access or create path '%s': %w", s.Path, err)
		}
	}

	return nil
}

func (s *SFTPStorage) HideSensitiveData() {
	s.Password = ""
	s.PrivateKey = ""
}

func (s *SFTPStorage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if s.Password != "" {
		encrypted, err := encryptor.Encrypt(s.StorageID, s.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt SFTP password: %w", err)
		}
		s.Password = encrypted
	}

	if s.PrivateKey != "" {
		encrypted, err := encryptor.Encrypt(s.StorageID, s.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt SFTP private key: %w", err)
		}
		s.PrivateKey = encrypted
	}

	return nil
}

func (s *SFTPStorage) Update(incoming *SFTPStorage) {
	s.Host = incoming.Host
	s.Port = incoming.Port
	s.Username = incoming.Username
	s.SkipHostKeyVerify = incoming.SkipHostKeyVerify
	s.Path = incoming.Path

	if incoming.Password != "" {
		s.Password = incoming.Password
	}

	if incoming.PrivateKey != "" {
		s.PrivateKey = incoming.PrivateKey
	}
}

func (s *SFTPStorage) connect(
	encryptor encryption.FieldEncryptor,
	timeout time.Duration,
) (*sftp.Client, *ssh.Client, error) {
	return s.connectWithContext(context.Background(), encryptor, timeout)
}

func (s *SFTPStorage) connectWithContext(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	timeout time.Duration,
) (*sftp.Client, *ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if s.Password != "" {
		password, err := encryptor.Decrypt(s.StorageID, s.Password)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decrypt SFTP password: %w", err)
		}
		authMethods = append(authMethods, ssh.Password(password))
	}

	if s.PrivateKey != "" {
		privateKey, err := encryptor.Decrypt(s.StorageID, s.PrivateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decrypt SFTP private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	var hostKeyCallback ssh.HostKeyCallback
	if s.SkipHostKeyVerify {
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	config := &ssh.ClientConfig{
		User:            s.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}

	address := fmt.Sprintf("%s:%d", s.Host, s.Port)

	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial SFTP server: %w", err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to create SSH connection: %w", err)
	}

	sshClient := ssh.NewClient(sshConn, chans, reqs)

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	return sftpClient, sshClient, nil
}

func (s *SFTPStorage) ensureDirectory(client *sftp.Client, path string) error {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return nil
	}

	parts := strings.Split(path, "/")
	currentPath := ""

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}

		if currentPath == "" {
			currentPath = "/" + part
		} else {
			currentPath = currentPath + "/" + part
		}

		_, err := client.Stat(currentPath)
		if err != nil {
			err = client.Mkdir(currentPath)
			if err != nil {
				return fmt.Errorf("failed to create directory '%s': %w", currentPath, err)
			}
		}
	}

	return nil
}

func (s *SFTPStorage) getFilePath(filename string) string {
	if s.Path == "" {
		return filename
	}

	path := strings.TrimPrefix(s.Path, "/")
	path = strings.TrimSuffix(path, "/")

	return "/" + path + "/" + filename
}

type sftpFileReader struct {
	file    *sftp.File
	client  *sftp.Client
	sshConn *ssh.Client
}

func (r *sftpFileReader) Read(p []byte) (n int, err error) {
	return r.file.Read(p)
}

func (r *sftpFileReader) Close() error {
	var errs []error

	if r.file != nil {
		if err := r.file.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close file: %w", err))
		}
	}

	if r.client != nil {
		if err := r.client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close SFTP client: %w", err))
		}
	}

	if r.sshConn != nil {
		if err := r.sshConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close SSH connection: %w", err))
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.reader.Read(p)
	}
}
