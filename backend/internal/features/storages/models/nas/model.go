package nas_storage

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hirochachacha/go-smb2"

	storage_connections "databasus-backend/internal/features/storages/connections"
	"databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
)

const (
	nasDeleteTimeout      = 30 * time.Second
	nasTestConnectTimeout = 30 * time.Second

	// Chunk size for NAS uploads - 16MB provides good balance between
	// memory usage and upload efficiency. This creates backpressure to pg_dump
	// by only reading one chunk at a time and waiting for NAS to confirm receipt.
	nasChunkSize = 16 * 1024 * 1024
)

type NASStorage struct {
	StorageID uuid.UUID `json:"storageId" gorm:"primaryKey;type:uuid;column:storage_id"`
	Host      string    `json:"host"      gorm:"not null;type:text;column:host"`
	Port      int       `json:"port"      gorm:"not null;default:445;column:port"`
	Share     string    `json:"share"     gorm:"not null;type:text;column:share"`
	Username  string    `json:"username"  gorm:"not null;type:text;column:username"`
	Password  string    `json:"password"  gorm:"not null;type:text;column:password"`
	UseSSL    bool      `json:"useSsl"    gorm:"not null;default:false;column:use_ssl"`
	Domain    string    `json:"domain"    gorm:"type:text;column:domain"`
	Path      string    `json:"path"      gorm:"type:text;column:path"`
}

func (n *NASStorage) TableName() string {
	return "nas_storages"
}

func (n *NASStorage) SaveFile(
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

	logger.InfoContext(ctx, "starting to save file to NAS storage", "file_name", fileName, "host", n.Host)

	lease, err := n.checkoutSession(ctx, encryptor)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create NAS session", "file_name", fileName, "error", err)
		return fmt.Errorf("failed to create NAS session: %w", err)
	}

	filePath := n.getFilePath(fileName)
	logger.DebugContext(ctx, "creating file on NAS", "file_name", fileName, "file_path", filePath)

	err = writeNasFile(ctx, lease.GetSession().getShare(ctx), filePath, file)
	lease.Release(err)

	if err != nil {
		logger.ErrorContext(ctx, "failed to write file to NAS", "file_name", fileName, "error", err)
		return err
	}

	logger.InfoContext(
		ctx,
		"successfully saved file to NAS storage",
		"file_name",
		fileName,
		"file_path",
		filePath,
	)
	return nil
}

func writeNasFile(ctx context.Context, share *smb2.Share, filePath string, source io.Reader) error {
	nasFile, err := share.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file on NAS: %w", err)
	}

	if _, err := copyWithContext(ctx, nasFile, source); err != nil {
		_ = nasFile.Close()
		return fmt.Errorf("failed to write file to NAS: %w", err)
	}

	if err := nasFile.Close(); err != nil {
		return fmt.Errorf("failed to close file on NAS: %w", err)
	}

	return nil
}

func (n *NASStorage) GetFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) (io.ReadCloser, error) {
	lease, err := n.checkoutSession(ctx, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to create NAS session: %w", err)
	}

	filePath := n.getFilePath(fileName)

	fileInfo, err := lease.GetSession().getShare(ctx).Stat(filePath)
	lease.Release(ignoreNotExist(err))

	if err != nil {
		return nil, fmt.Errorf("file not found: %s", fileName)
	}

	totalBytes := fileInfo.Size()

	return io_utils.NewResumingReader(io_utils.ResumingReaderSpec{
		StreamCtx:  ctx,
		Logger:     logger,
		FileName:   fileName,
		TotalBytes: totalBytes,
		OpenAtOffset: func(attemptCtx context.Context, offsetBytes int64) (io.ReadCloser, error) {
			return n.openFileAtOffset(attemptCtx, encryptor, filePath, offsetBytes)
		},
	}), nil
}

func (n *NASStorage) DeleteFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) error {
	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation carries its own deadline; the caller's ctx stays for log correlation only.
	deleteCtx, cancel := context.WithTimeout(context.Background(), nasDeleteTimeout)
	defer cancel()

	lease, err := n.checkoutSession(deleteCtx, encryptor)
	if err != nil {
		return fmt.Errorf("failed to create NAS session: %w", err)
	}

	err = ignoreNotExist(lease.GetSession().getShare(deleteCtx).Remove(n.getFilePath(fileName)))
	lease.Release(err)

	if err != nil {
		return fmt.Errorf("failed to delete file from NAS: %w", err)
	}

	logger.DebugContext(ctx, "deleted file from nas storage", "file_name", fileName)

	return nil
}

func (n *NASStorage) Validate(encryptor encryption.FieldEncryptor) error {
	if n.Host == "" {
		return errors.New("NAS host is required")
	}
	if n.Share == "" {
		return errors.New("NAS share is required")
	}
	if n.Username == "" {
		return errors.New("NAS username is required")
	}
	if n.Password == "" {
		return errors.New("NAS password is required")
	}
	if n.Port <= 0 || n.Port > 65535 {
		return errors.New("NAS port must be between 1 and 65535")
	}

	return nil
}

// A connection test often runs for settings that are not saved yet, so it dials its own session
// instead of placing one in the pool under a storage ID it may not have.
func (n *NASStorage) TestConnection(encryptor encryption.FieldEncryptor) error {
	ctx, cancel := context.WithTimeout(context.Background(), nasTestConnectTimeout)
	defer cancel()

	session, err := n.dialSession(ctx, encryptor)
	if err != nil {
		return fmt.Errorf("failed to connect to NAS: %w", err)
	}
	defer closeSession(session)

	return nil
}

func (n *NASStorage) HideSensitiveData() {
	n.Password = ""
}

func (n *NASStorage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if n.Password != "" {
		encrypted, err := encryptor.Encrypt(n.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt NAS password: %w", err)
		}
		n.Password = encrypted
	}

	return nil
}

func (n *NASStorage) Update(incoming *NASStorage) {
	n.Host = incoming.Host
	n.Port = incoming.Port
	n.Share = incoming.Share
	n.Username = incoming.Username
	n.UseSSL = incoming.UseSSL
	n.Domain = incoming.Domain
	n.Path = incoming.Path

	if incoming.Password != "" {
		n.Password = incoming.Password
	}
}

func (n *NASStorage) dialSmbSession(
	ctx context.Context,
	conn net.Conn,
	encryptor encryption.FieldEncryptor,
) (*smb2.Session, error) {
	password, err := encryptor.Decrypt(n.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt NAS password: %w", err)
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     n.Username,
			Password: password,
			Domain:   n.Domain,
		},
	}

	session, err := d.DialContext(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMB session: %w", err)
	}

	return session, nil
}

func (n *NASStorage) createConnectionWithContext(ctx context.Context) (net.Conn, error) {
	address := net.JoinHostPort(n.Host, fmt.Sprintf("%d", n.Port))

	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
	}

	if n.UseSSL {
		tlsConfig := &tls.Config{
			ServerName:         n.Host,
			InsecureSkipVerify: false,
		}
		conn, err := (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, fmt.Errorf("failed to create SSL connection to %s: %w", address, err)
		}
		return conn, nil
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection to %s: %w", address, err)
	}
	return conn, nil
}

// MkdirAll also succeeds when a concurrent session created the directory first, which happens when
// several sessions dial at once.
func (n *NASStorage) ensureDirectory(share *smb2.Share, path string) error {
	path = strings.ReplaceAll(filepath.Clean(path), "\\", "/")

	if err := share.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("failed to create directory '%s': %w", path, err)
	}

	return nil
}

func (n *NASStorage) getFilePath(filename string) string {
	if n.Path == "" {
		return filename
	}

	// Clean path and use forward slashes for SMB
	cleanPath := filepath.Clean(n.Path)
	cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")

	return cleanPath + "/" + filename
}

type nasFileReader struct {
	file    *smb2.File
	lease   *storage_connections.Lease[*nasSession]
	readErr error
}

func (r *nasFileReader) Read(p []byte) (int, error) {
	readBytes, err := r.file.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		r.readErr = err
	}

	return readBytes, err
}

func (r *nasFileReader) Close() error {
	closeErr := r.file.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("failed to close file: %w", closeErr)
	}

	r.lease.Release(errors.Join(r.readErr, closeErr))

	return closeErr
}

type writeResult struct {
	bytesWritten int
	writeErr     error
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, nasChunkSize)
	var written int64

	for {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		nr, readErr := io.ReadFull(src, buf)

		if nr == 0 && readErr == io.EOF {
			break
		}

		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return written, readErr
		}

		writeResultCh := make(chan writeResult, 1)
		go func() {
			nw, writeErr := dst.Write(buf[0:nr])
			writeResultCh <- writeResult{nw, writeErr}
		}()

		var nw int
		var writeErr error

		select {
		case <-ctx.Done():
			return written, ctx.Err()
		case result := <-writeResultCh:
			nw = result.bytesWritten
			writeErr = result.writeErr
		}

		if nw < 0 || nr < nw {
			nw = 0
			if writeErr == nil {
				writeErr = errors.New("invalid write result")
			}
		}

		if writeErr != nil {
			return written, writeErr
		}

		if nr != nw {
			return written, io.ErrShortWrite
		}

		written += int64(nw)

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	return written, nil
}

// Each attempt takes its own session. A resumed read follows a dropped one, and the session that
// carried it was closed when its read failed, so the retry gets a live one.
func (n *NASStorage) openFileAtOffset(
	attemptCtx context.Context,
	encryptor encryption.FieldEncryptor,
	filePath string,
	offsetBytes int64,
) (io.ReadCloser, error) {
	lease, err := n.checkoutSession(attemptCtx, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to create NAS session: %w", err)
	}

	nasFile, err := lease.GetSession().getShare(attemptCtx).Open(filePath)
	if err != nil {
		lease.Release(ignoreNotExist(err))

		return nil, fmt.Errorf("failed to open file from NAS: %w", err)
	}

	if offsetBytes > 0 {
		if _, err := nasFile.Seek(offsetBytes, io.SeekStart); err != nil {
			_ = nasFile.Close()
			lease.Release(err)

			return nil, fmt.Errorf("failed to resume NAS read at offset %d: %w", offsetBytes, err)
		}
	}

	return &nasFileReader{file: nasFile, lease: lease}, nil
}

func ignoreNotExist(err error) error {
	if errors.Is(err, iofs.ErrNotExist) {
		return nil
	}

	return err
}
