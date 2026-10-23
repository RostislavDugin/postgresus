package sftp_storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	storage_connections "databasus-backend/internal/features/storages/connections"
	"databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
)

const (
	sftpConnectTimeout     = 30 * time.Second
	sftpTestConnectTimeout = 10 * time.Second
	sftpDeleteTimeout      = 30 * time.Second
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
	fileName string,
	file io.Reader,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger.InfoContext(ctx, "starting to save file to SFTP storage", "file_name", fileName, "host", s.Host)

	lease, err := s.checkoutSession(ctx, encryptor)
	if err != nil {
		logger.ErrorContext(ctx, "failed to connect to SFTP", "file_name", fileName, "error", err)
		return fmt.Errorf("failed to connect to SFTP: %w", err)
	}

	filePath := s.getFilePath(fileName)
	logger.DebugContext(ctx, "uploading file to SFTP", "file_name", fileName, "file_path", filePath)

	err = writeRemoteFile(lease.GetSession().sftpClient, filePath, &contextReader{ctx: ctx, reader: file})
	lease.Release(err)

	if err != nil {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "SFTP upload cancelled", "file_name", fileName)
			return ctx.Err()
		default:
			logger.ErrorContext(ctx, "failed to upload file to SFTP", "file_name", fileName, "error", err)
			return fmt.Errorf("failed to upload file to SFTP: %w", err)
		}
	}

	logger.InfoContext(
		ctx,
		"successfully saved file to SFTP storage",
		"file_name",
		fileName,
		"file_path",
		filePath,
	)
	return nil
}

// The plain copy ends in the library's ReadFrom, which for a reader of unknown size, such as the
// compression pipe, writes one 32 KB block at a time and waits for each answer. Over a distant
// server that is hundreds of round trips per WAL segment. The concurrent path keeps blocks in flight
// whatever the reader's size. A failed write may leave holes in the remote file, and the caller
// discards the attempt as it does any failed write.
func writeRemoteFile(client *sftp.Client, filePath string, source io.Reader) error {
	remoteFile, err := client.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}

	if _, err := remoteFile.ReadFromWithConcurrency(source, 0); err != nil {
		_ = remoteFile.Close()
		return err
	}

	if err := remoteFile.Close(); err != nil {
		return fmt.Errorf("failed to close remote file: %w", err)
	}

	return nil
}

func (s *SFTPStorage) GetFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) (io.ReadCloser, error) {
	lease, err := s.checkoutSession(ctx, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SFTP: %w", err)
	}

	filePath := s.getFilePath(fileName)

	fileInfo, err := lease.GetSession().sftpClient.Stat(filePath)
	lease.Release(ignoreNotExist(err))

	if err != nil {
		return nil, fmt.Errorf("failed to stat file on SFTP: %w", err)
	}

	totalBytes := fileInfo.Size()

	return io_utils.NewResumingReader(io_utils.ResumingReaderSpec{
		StreamCtx:  ctx,
		Logger:     logger,
		FileName:   fileName,
		TotalBytes: totalBytes,
		OpenAtOffset: func(attemptCtx context.Context, offsetBytes int64) (io.ReadCloser, error) {
			return s.openFileAtOffset(attemptCtx, encryptor, filePath, offsetBytes)
		},
	}), nil
}

func (s *SFTPStorage) DeleteFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) error {
	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation carries its own deadline; the caller's ctx stays for log correlation only.
	deleteCtx, cancel := context.WithTimeout(context.Background(), sftpDeleteTimeout)
	defer cancel()

	lease, err := s.checkoutSession(deleteCtx, encryptor)
	if err != nil {
		return fmt.Errorf("failed to connect to SFTP: %w", err)
	}

	err = ignoreNotExist(lease.GetSession().sftpClient.Remove(s.getFilePath(fileName)))
	lease.Release(err)

	if err != nil {
		return fmt.Errorf("failed to delete file from SFTP: %w", err)
	}

	logger.DebugContext(ctx, "deleted file from sftp storage", "file_name", fileName)

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
		encrypted, err := encryptor.Encrypt(s.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt SFTP password: %w", err)
		}
		s.Password = encrypted
	}

	if s.PrivateKey != "" {
		encrypted, err := encryptor.Encrypt(s.PrivateKey)
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

func (s *SFTPStorage) connectWithContext(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	timeout time.Duration,
) (*sftp.Client, *ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if s.Password != "" {
		password, err := encryptor.Decrypt(s.Password)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decrypt SFTP password: %w", err)
		}
		authMethods = append(authMethods, ssh.Password(password))
	}

	if s.PrivateKey != "" {
		privateKey, err := encryptor.Decrypt(s.PrivateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decrypt SFTP private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Accepted risk: the storage config carries no host key fingerprint to verify against.
	// nosemgrep: go.lang.security.audit.crypto.insecure_ssh.avoid-ssh-insecure-ignore-host-key
	hostKeyCallback := ssh.InsecureIgnoreHostKey()

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

	// The SSH handshake and the SFTP init ignore the context, so a server that accepts the TCP
	// connection and then goes silent would hold the dial forever without this.
	stopCloseOnCancel := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stopCloseOnCancel()

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

	if !stopCloseOnCancel() {
		closeSession(&sftpSession{sshClient: sshClient, sftpClient: sftpClient})
		return nil, nil, fmt.Errorf("failed to connect to SFTP server: %w", ctx.Err())
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

		if _, err := client.Stat(currentPath); err == nil {
			continue
		}

		// Concurrent sessions dial at once, and the one that loses the race to create the directory
		// finds it in place.
		if err := client.Mkdir(currentPath); err != nil {
			if info, statErr := client.Stat(currentPath); statErr == nil && info.IsDir() {
				continue
			}

			return fmt.Errorf("failed to create directory '%s': %w", currentPath, err)
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
	lease   *storage_connections.Lease[*sftpSession]
	readErr error
}

func (r *sftpFileReader) Read(p []byte) (int, error) {
	readBytes, err := r.file.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		r.readErr = err
	}

	return readBytes, err
}

func (r *sftpFileReader) Close() error {
	closeErr := r.file.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("failed to close file: %w", closeErr)
	}

	r.lease.Release(errors.Join(r.readErr, closeErr))

	return closeErr
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

// Each attempt takes its own session. A resumed read follows a dropped one, and the session that
// carried it was closed when its read failed, so the retry gets a live one.
func (s *SFTPStorage) openFileAtOffset(
	attemptCtx context.Context,
	encryptor encryption.FieldEncryptor,
	filePath string,
	offsetBytes int64,
) (io.ReadCloser, error) {
	lease, err := s.checkoutSession(attemptCtx, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SFTP: %w", err)
	}

	remoteFile, err := lease.GetSession().sftpClient.Open(filePath)
	if err != nil {
		lease.Release(ignoreNotExist(err))
		return nil, fmt.Errorf("failed to open file from SFTP: %w", err)
	}

	if offsetBytes > 0 {
		if _, err := remoteFile.Seek(offsetBytes, io.SeekStart); err != nil {
			_ = remoteFile.Close()
			lease.Release(err)

			return nil, fmt.Errorf("failed to resume SFTP read at offset %d: %w", offsetBytes, err)
		}
	}

	return &sftpFileReader{file: remoteFile, lease: lease}, nil
}

func ignoreNotExist(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return err
}
