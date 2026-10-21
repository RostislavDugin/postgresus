package ftp_storage

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/textproto"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jlaffaye/ftp"

	storage_connections "databasus-backend/internal/features/storages/connections"
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

	logger.InfoContext(ctx, "starting to save file to FTP storage", "file_name", fileName, "host", f.Host)

	lease, err := f.checkoutSession(ctx, encryptor)
	if err != nil {
		logger.ErrorContext(ctx, "failed to connect to FTP", "file_name", fileName, "error", err)
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}

	filePath := f.getFilePath(fileName)
	logger.DebugContext(ctx, "uploading file to FTP", "file_name", fileName, "file_path", filePath)

	err = lease.GetSession().serverConn.Stor(filePath, &contextReader{ctx: ctx, reader: file})
	lease.Release(ignoreFtpReply(err))

	if err != nil {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "FTP upload cancelled", "file_name", fileName)
			return ctx.Err()
		default:
			logger.ErrorContext(ctx, "failed to upload file to FTP", "file_name", fileName, "error", err)
			return fmt.Errorf("failed to upload file to FTP: %w", err)
		}
	}

	logger.InfoContext(
		ctx,
		"successfully saved file to FTP storage",
		"file_name",
		fileName,
		"file_path",
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
	lease, err := f.checkoutSession(ctx, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to FTP: %w", err)
	}

	filePath := f.getFilePath(fileName)

	totalBytes, err := lease.GetSession().serverConn.FileSize(filePath)
	if err != nil {
		// A server that does not implement SIZE still serves the file, so an unsupported command
		// costs the completeness check rather than the whole restore. Any other reply, 550 above
		// all, means the file itself is unreachable.
		if !isUnsupportedFtpCommand(err) {
			lease.Release(ignoreFtpReply(err))

			return nil, fmt.Errorf("failed to stat file on FTP: %w", err)
		}

		logger.WarnContext(ctx, "storage does not support the FTP SIZE command", "file_name", fileName, "error", err)

		totalBytes = io_utils.UnknownTotalBytes
	}

	lease.Release(nil)

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

func (f *FTPStorage) DeleteFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) error {
	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation carries its own deadline; the caller's ctx stays for log correlation only.
	deleteCtx, cancel := context.WithTimeout(context.Background(), ftpDeleteTimeout)
	defer cancel()

	lease, err := f.checkoutSession(deleteCtx, encryptor)
	if err != nil {
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}

	err = lease.GetSession().serverConn.Delete(f.getFilePath(fileName))
	lease.Release(ignoreFtpReply(err))

	if err != nil && !isFtpFileUnavailable(err) {
		return fmt.Errorf("failed to delete file from FTP: %w", err)
	}

	logger.DebugContext(ctx, "deleted file from ftp storage", "file_name", fileName)

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

	session, err := f.connectWithContext(ctx, encryptor, ftpTestConnectTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}
	defer closeSession(session)

	if f.Path != "" {
		if err := f.ensureDirectory(session.serverConn, f.Path); err != nil {
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

func (f *FTPStorage) connectWithContext(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	timeout time.Duration,
) (*ftpSession, error) {
	password, err := encryptor.Decrypt(f.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt FTP password: %w", err)
	}

	address := fmt.Sprintf("%s:%d", f.Host, f.Port)

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	connections := &trackedConnections{
		open:       map[*trackedConnection]struct{}{},
		dialer:     net.Dialer{Timeout: timeout},
		controlCtx: dialCtx,
	}

	dialOptions := []ftp.DialOption{ftp.DialWithDialFunc(connections.dial)}

	if f.UseSSL {
		connections.tlsConfig = &tls.Config{
			ServerName:         f.Host,
			InsecureSkipVerify: f.SkipTLSVerify,
		}
		dialOptions = append(dialOptions, ftp.DialWithExplicitTLS(connections.tlsConfig))
	}

	session := &ftpSession{connections: connections}

	// The greeting, the TLS upgrade and the login ignore the context, so a server that accepts the
	// TCP connection and then goes silent would hold the dial forever without this.
	stopCloseOnCancel := context.AfterFunc(dialCtx, func() {
		closeSession(session)
	})
	defer stopCloseOnCancel()

	session.serverConn, err = ftp.Dial(address, dialOptions...)
	if err != nil {
		closeSession(session)
		return nil, fmt.Errorf("failed to dial FTP server: %w", err)
	}

	if err := session.serverConn.Login(f.Username, password); err != nil {
		closeSession(session)
		return nil, fmt.Errorf("failed to login to FTP server: %w", err)
	}

	if !stopCloseOnCancel() {
		closeSession(session)
		return nil, fmt.Errorf("failed to connect to FTP server: %w", dialCtx.Err())
	}

	return session, nil
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

		if err := conn.ChangeDir(part); err == nil {
			continue
		}

		// Concurrent sessions dial at once, and the one that loses the race to create the directory
		// finds it in place, so a failed MKD is judged by whether the directory can be entered.
		makeDirErr := conn.MakeDir(part)

		if err := conn.ChangeDir(part); err != nil {
			return fmt.Errorf("failed to create directory '%s': %w", part, errors.Join(makeDirErr, err))
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
	response *ftp.Response
	lease    *storage_connections.Lease[*ftpSession]
	readErr  error
}

func (r *ftpFileReader) Read(p []byte) (int, error) {
	readBytes, err := r.response.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		r.readErr = err
	}

	return readBytes, err
}

func (r *ftpFileReader) Close() error {
	closeErr := r.response.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("failed to close response: %w", closeErr)
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

// Each attempt takes its own session: an FTP data transfer dies together with the control channel
// that opened it, and a session whose read failed was closed, so a resumed REST gets a live one.
func (f *FTPStorage) retrieveFileFromOffset(
	attemptCtx context.Context,
	encryptor encryption.FieldEncryptor,
	filePath string,
	offsetBytes int64,
) (io.ReadCloser, error) {
	lease, err := f.checkoutSession(attemptCtx, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to FTP: %w", err)
	}

	response, err := lease.GetSession().serverConn.RetrFrom(filePath, uint64(offsetBytes))
	if err != nil {
		lease.Release(ignoreFtpReply(err))

		return nil, fmt.Errorf("failed to retrieve file from FTP: %w", err)
	}

	return &ftpFileReader{response: response, lease: lease}, nil
}

// A protocol reply such as 550 leaves the control channel in a clean state, so only transport
// failures make a session unusable.
func ignoreFtpReply(err error) error {
	var protocolError *textproto.Error
	if errors.As(err, &protocolError) {
		return nil
	}

	return err
}

// 550 is the only reply an FTP server has for a missing file, and also its reply
// for a refused one, so absence is the closest reading the protocol allows.
func isFtpFileUnavailable(err error) bool {
	var protocolError *textproto.Error
	if !errors.As(err, &protocolError) {
		return false
	}

	return protocolError.Code == ftp.StatusFileUnavailable
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
