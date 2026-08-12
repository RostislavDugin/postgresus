package ftp_storage

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/textproto"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jlaffaye/ftp"

	"databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
)

const (
	ftpConnectTimeout     = 30 * time.Second
	ftpTestConnectTimeout = 10 * time.Second
	ftpDeleteTimeout      = 30 * time.Second
	ftpChunkSize          = 16 * 1024 * 1024
)

type FTPStorage struct {
	StorageID     uuid.UUID `json:"storageId"     gorm:"primaryKey;type:uuid;column:storage_id"`
	Host          string    `json:"host"          gorm:"not null;type:text;column:host"`
	Port          int       `json:"port"          gorm:"not null;default:21;column:port"`
	Username      string    `json:"username"      gorm:"not null;type:text;column:username"`
	Password      string    `json:"password"      gorm:"not null;type:text;column:password"`
	Path          string    `json:"path"          gorm:"type:text;column:path"`
	UseSSL        bool      `json:"useSsl"        gorm:"not null;default:false;column:use_ssl"`
	SkipTLSVerify bool      `json:"skipTlsVerify" gorm:"not null;default:false;column:skip_tls_verify"`
}

func (f *FTPStorage) TableName() string {
	return "ftp_storages"
}

func (f *FTPStorage) SaveFile(
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

	logger.Info("Starting to save file to FTP storage", "fileName", fileName, "host", f.Host)

	conn, err := f.connect(encryptor, ftpConnectTimeout)
	if err != nil {
		logger.Error("Failed to connect to FTP", "fileName", fileName, "error", err)
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}
	defer func() {
		if quitErr := conn.Quit(); quitErr != nil {
			logger.Error(
				"Failed to close FTP connection",
				"fileName",
				fileName,
				"error",
				quitErr,
			)
		}
	}()

	if f.Path != "" {
		if err := f.ensureDirectory(conn, f.Path); err != nil {
			logger.Error(
				"Failed to ensure directory",
				"fileName",
				fileName,
				"path",
				f.Path,
				"error",
				err,
			)
			return fmt.Errorf("failed to ensure directory: %w", err)
		}
	}

	filePath := f.getFilePath(fileName)
	logger.Debug("Uploading file to FTP", "fileName", fileName, "filePath", filePath)

	ctxReader := &contextReader{ctx: ctx, reader: file}

	err = conn.Stor(filePath, ctxReader)
	if err != nil {
		select {
		case <-ctx.Done():
			logger.Info("FTP upload cancelled", "fileName", fileName)
			return ctx.Err()
		default:
			logger.Error("Failed to upload file to FTP", "fileName", fileName, "error", err)
			return fmt.Errorf("failed to upload file to FTP: %w", err)
		}
	}

	logger.Info(
		"Successfully saved file to FTP storage",
		"fileName",
		fileName,
		"filePath",
		filePath,
	)
	return nil
}

func (f *FTPStorage) GetFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) (io.ReadCloser, error) {
	conn, err := f.connect(encryptor, ftpConnectTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to FTP: %w", err)
	}

	filePath := f.getFilePath(fileName)

	totalBytes, err := conn.FileSize(filePath)
	if err != nil {
		// A server that does not implement SIZE still serves the file, so an unsupported command
		// costs the completeness check rather than the whole restore. Any other reply, 550 above
		// all, means the file itself is unreachable.
		if !isUnsupportedFtpCommand(err) {
			_ = conn.Quit()

			return nil, fmt.Errorf("failed to stat file on FTP: %w", err)
		}

		logger.Warn("storage does not support the FTP SIZE command", "file_name", fileName, "error", err)

		totalBytes = io_utils.UnknownTotalBytes
	}

	if err := conn.Quit(); err != nil {
		logger.Warn("failed to close the FTP connection used for stat", "error", err)
	}

	return io_utils.NewResumingReader(io_utils.ResumingReaderSpec{
		StreamCtx:  ctx,
		Logger:     logger,
		FileName:   fileName,
		TotalBytes: totalBytes,
		OpenAtOffset: func(attemptCtx context.Context, offsetBytes int64) (io.ReadCloser, error) {
			return f.retrieveFileFromOffset(attemptCtx, encryptor, filePath, offsetBytes)
		},
	}), nil
}

func (f *FTPStorage) DeleteFile(encryptor encryption.FieldEncryptor, fileName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), ftpDeleteTimeout)
	defer cancel()

	conn, err := f.connectWithContext(ctx, encryptor, ftpDeleteTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}
	defer func() {
		_ = conn.Quit()
	}()

	filePath := f.getFilePath(fileName)

	_, err = conn.FileSize(filePath)
	if err != nil {
		return nil
	}

	err = conn.Delete(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file from FTP: %w", err)
	}

	return nil
}

func (f *FTPStorage) Validate(encryptor encryption.FieldEncryptor) error {
	if f.Host == "" {
		return errors.New("FTP host is required")
	}
	if f.Username == "" {
		return errors.New("FTP username is required")
	}
	if f.Password == "" {
		return errors.New("FTP password is required")
	}
	if f.Port <= 0 || f.Port > 65535 {
		return errors.New("FTP port must be between 1 and 65535")
	}

	return nil
}

func (f *FTPStorage) TestConnection(encryptor encryption.FieldEncryptor) error {
	ctx, cancel := context.WithTimeout(context.Background(), ftpTestConnectTimeout)
	defer cancel()

	conn, err := f.connectWithContext(ctx, encryptor, ftpTestConnectTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}
	defer func() {
		_ = conn.Quit()
	}()

	if f.Path != "" {
		if err := f.ensureDirectory(conn, f.Path); err != nil {
			return fmt.Errorf("failed to access or create path '%s': %w", f.Path, err)
		}
	}

	return nil
}

func (f *FTPStorage) HideSensitiveData() {
	f.Password = ""
}

func (f *FTPStorage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if f.Password != "" {
		encrypted, err := encryptor.Encrypt(f.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt FTP password: %w", err)
		}
		f.Password = encrypted
	}

	return nil
}

func (f *FTPStorage) Update(incoming *FTPStorage) {
	f.Host = incoming.Host
	f.Port = incoming.Port
	f.Username = incoming.Username
	f.UseSSL = incoming.UseSSL
	f.SkipTLSVerify = incoming.SkipTLSVerify
	f.Path = incoming.Path

	if incoming.Password != "" {
		f.Password = incoming.Password
	}
}

func (f *FTPStorage) connect(
	encryptor encryption.FieldEncryptor,
	timeout time.Duration,
) (*ftp.ServerConn, error) {
	return f.connectWithContext(context.Background(), encryptor, timeout)
}

func (f *FTPStorage) connectWithContext(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	timeout time.Duration,
) (*ftp.ServerConn, error) {
	password, err := encryptor.Decrypt(f.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt FTP password: %w", err)
	}

	address := fmt.Sprintf("%s:%d", f.Host, f.Port)

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var conn *ftp.ServerConn
	if f.UseSSL {
		tlsConfig := &tls.Config{
			ServerName:         f.Host,
			InsecureSkipVerify: f.SkipTLSVerify,
		}
		conn, err = ftp.Dial(address,
			ftp.DialWithContext(dialCtx),
			ftp.DialWithExplicitTLS(tlsConfig),
		)
	} else {
		conn, err = ftp.Dial(address, ftp.DialWithContext(dialCtx))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to dial FTP server: %w", err)
	}

	err = conn.Login(f.Username, password)
	if err != nil {
		_ = conn.Quit()
		return nil, fmt.Errorf("failed to login to FTP server: %w", err)
	}

	return conn, nil
}

func (f *FTPStorage) ensureDirectory(conn *ftp.ServerConn, path string) error {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return nil
	}

	parts := strings.Split(path, "/")

	currentDir, err := conn.CurrentDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer func() {
		_ = conn.ChangeDir(currentDir)
	}()

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}

		err := conn.ChangeDir(part)
		if err != nil {
			err = conn.MakeDir(part)
			if err != nil {
				return fmt.Errorf("failed to create directory '%s': %w", part, err)
			}
			err = conn.ChangeDir(part)
			if err != nil {
				return fmt.Errorf("failed to change into directory '%s': %w", part, err)
			}
		}
	}

	return nil
}

func (f *FTPStorage) getFilePath(filename string) string {
	if f.Path == "" {
		return filename
	}

	path := strings.TrimPrefix(f.Path, "/")
	path = strings.TrimSuffix(path, "/")

	return path + "/" + filename
}

type ftpFileReader struct {
	response             *ftp.Response
	conn                 *ftp.ServerConn
	stopDeadlineOnCancel func() bool
}

func (r *ftpFileReader) Read(p []byte) (n int, err error) {
	return r.response.Read(p)
}

func (r *ftpFileReader) Close() error {
	var errs []error

	if r.stopDeadlineOnCancel != nil {
		r.stopDeadlineOnCancel()
	}

	if r.response != nil {
		if err := r.response.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close response: %w", err))
		}
	}

	if r.conn != nil {
		if err := r.conn.Quit(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection: %w", err))
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

// Each attempt dials a fresh control connection: an FTP data transfer dies together with the
// control channel that opened it, so a resumed REST cannot reuse the broken one.
func (f *FTPStorage) retrieveFileFromOffset(
	attemptCtx context.Context,
	encryptor encryption.FieldEncryptor,
	filePath string,
	offsetBytes int64,
) (io.ReadCloser, error) {
	conn, err := f.connect(encryptor, ftpConnectTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to FTP: %w", err)
	}

	response, err := conn.RetrFrom(filePath, uint64(offsetBytes))
	if err != nil {
		_ = conn.Quit()

		return nil, fmt.Errorf("failed to retrieve file from FTP: %w", err)
	}

	// The FTP client takes no context, so a read blocked on a silent server would ignore the
	// caller's cancellation; an expired deadline is what unblocks it.
	stopDeadlineOnCancel := context.AfterFunc(attemptCtx, func() {
		_ = response.SetDeadline(time.Now())
	})

	return &ftpFileReader{
		response:             response,
		conn:                 conn,
		stopDeadlineOnCancel: stopDeadlineOnCancel,
	}, nil
}

func isUnsupportedFtpCommand(err error) bool {
	var protocolError *textproto.Error
	if !errors.As(err, &protocolError) {
		return false
	}

	return protocolError.Code == ftp.StatusNotImplemented ||
		protocolError.Code == ftp.StatusBadCommand ||
		protocolError.Code == ftp.StatusCommandNotImplemented
}
