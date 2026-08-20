package nas_storage

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hirochachacha/go-smb2"

	"databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
)

const (
	nasDeleteTimeout = 30 * time.Second

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

	session, err := n.createSessionWithContext(ctx, encryptor)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create NAS session", "file_name", fileName, "error", err)
		return fmt.Errorf("failed to create NAS session: %w", err)
	}
	defer func() {
		if logoffErr := session.Logoff(); logoffErr != nil {
			logger.ErrorContext(
				ctx,
				"failed to logoff NAS session",
				"file_name",
				fileName,
				"error",
				logoffErr,
			)
		}
	}()

	fs, err := session.Mount(n.Share)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to mount NAS share",
			"file_name",
			fileName,
			"share",
			n.Share,
			"error",
			err,
		)
		return fmt.Errorf("failed to mount share '%s': %w", n.Share, err)
	}
	defer func() {
		if umountErr := fs.Umount(); umountErr != nil {
			logger.ErrorContext(
				ctx,
				"failed to unmount NAS share",
				"file_name",
				fileName,
				"error",
				umountErr,
			)
		}
	}()

	// Ensure the directory exists
	if n.Path != "" {
		if err := n.ensureDirectory(fs, n.Path); err != nil {
			logger.ErrorContext(
				ctx,
				"failed to ensure directory",
				"file_name",
				fileName,
				"path",
				n.Path,
				"error",
				err,
			)
			return fmt.Errorf("failed to ensure directory: %w", err)
		}
	}

	filePath := n.getFilePath(fileName)
	logger.DebugContext(ctx, "creating file on NAS", "file_name", fileName, "file_path", filePath)

	nasFile, err := fs.Create(filePath)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to create file on NAS",
			"file_name",
			fileName,
			"file_path",
			filePath,
			"error",
			err,
		)
		return fmt.Errorf("failed to create file on NAS: %w", err)
	}
	defer func() {
		if closeErr := nasFile.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close NAS file", "file_name", fileName, "error", closeErr)
		}
	}()

	logger.DebugContext(ctx, "copying file data to NAS", "file_name", fileName)
	_, err = copyWithContext(ctx, nasFile, file)
	if err != nil {
		logger.ErrorContext(ctx, "failed to write file to NAS", "file_name", fileName, "error", err)
		return fmt.Errorf("failed to write file to NAS: %w", err)
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

func (n *NASStorage) GetFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) (io.ReadCloser, error) {
	session, err := n.createSession(encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to create NAS session: %w", err)
	}

	share, err := session.Mount(n.Share)
	if err != nil {
		_ = session.Logoff()

		return nil, fmt.Errorf("failed to mount share '%s': %w", n.Share, err)
	}

	filePath := n.getFilePath(fileName)

	fileInfo, err := share.Stat(filePath)
	if err != nil {
		_ = share.Umount()
		_ = session.Logoff()

		return nil, fmt.Errorf("file not found: %s", fileName)
	}

	totalBytes := fileInfo.Size()

	_ = share.Umount()
	_ = session.Logoff()

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

	session, err := n.createSessionWithContext(deleteCtx, encryptor)
	if err != nil {
		return fmt.Errorf("failed to create NAS session: %w", err)
	}
	defer func() {
		_ = session.Logoff()
	}()

	fs, err := session.Mount(n.Share)
	if err != nil {
		return fmt.Errorf("failed to mount share '%s': %w", n.Share, err)
	}
	defer func() {
		_ = fs.Umount()
	}()

	filePath := n.getFilePath(fileName)

	_, err = fs.Stat(filePath)
	if err != nil {
		return nil
	}

	err = fs.Remove(filePath)
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

func (n *NASStorage) TestConnection(encryptor encryption.FieldEncryptor) error {
	session, err := n.createSession(encryptor)
	if err != nil {
		return fmt.Errorf("failed to connect to NAS: %w", err)
	}
	defer func() {
		_ = session.Logoff()
	}()

	// Try to mount the share to verify access
	fs, err := session.Mount(n.Share)
	if err != nil {
		return fmt.Errorf("failed to access share '%s': %w", n.Share, err)
	}
	defer func() {
		_ = fs.Umount()
	}()

	// If path is specified, check if it exists or can be created
	if n.Path != "" {
		if err := n.ensureDirectory(fs, n.Path); err != nil {
			return fmt.Errorf("failed to access or create path '%s': %w", n.Path, err)
		}
	}

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

func (n *NASStorage) createSession(encryptor encryption.FieldEncryptor) (*smb2.Session, error) {
	return n.createSessionWithContext(context.Background(), encryptor)
}

func (n *NASStorage) createSessionWithContext(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) (*smb2.Session, error) {
	conn, err := n.createConnectionWithContext(ctx)
	if err != nil {
		return nil, err
	}

	password, err := encryptor.Decrypt(n.Password)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to decrypt NAS password: %w", err)
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     n.Username,
			Password: password,
			Domain:   n.Domain,
		},
	}

	session, err := d.Dial(conn)
	if err != nil {
		_ = conn.Close()
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

func (n *NASStorage) ensureDirectory(fs *smb2.Share, path string) error {
	// Clean and normalize the path
	path = filepath.Clean(path)
	path = strings.ReplaceAll(path, "\\", "/")

	// Check if directory already exists
	_, err := fs.Stat(path)
	if err == nil {
		return nil // Directory exists
	}

	// Try to create the directory (including parent directories)
	parts := strings.Split(path, "/")
	currentPath := ""

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}

		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		// Check if this part of the path exists
		_, err := fs.Stat(currentPath)
		if err != nil {
			// Directory doesn't exist, try to create it
			err = fs.Mkdir(currentPath, 0o755)
			if err != nil {
				return fmt.Errorf("failed to create directory '%s': %w", currentPath, err)
			}
		}
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
	file              *smb2.File
	share             *smb2.Share
	session           *smb2.Session
	stopCloseOnCancel func() bool
}

func (r *nasFileReader) Read(p []byte) (n int, err error) {
	return r.file.Read(p)
}

func (r *nasFileReader) Close() error {
	var closeErrors []error

	if r.stopCloseOnCancel != nil {
		r.stopCloseOnCancel()
	}

	if r.file != nil {
		if err := r.file.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("failed to close file: %w", err))
		}
	}

	if r.share != nil {
		if err := r.share.Umount(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("failed to unmount share: %w", err))
		}
	}

	if r.session != nil {
		if err := r.session.Logoff(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("failed to logoff session: %w", err))
		}
	}

	if len(closeErrors) > 0 {
		return closeErrors[0]
	}

	return nil
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

// Each attempt logs in again: a resumed read follows a dropped one, and the SMB session that
// carried it is no longer usable.
func (n *NASStorage) openFileAtOffset(
	attemptCtx context.Context,
	encryptor encryption.FieldEncryptor,
	filePath string,
	offsetBytes int64,
) (io.ReadCloser, error) {
	session, err := n.createSessionWithContext(attemptCtx, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to create NAS session: %w", err)
	}

	share, err := session.Mount(n.Share)
	if err != nil {
		_ = session.Logoff()

		return nil, fmt.Errorf("failed to mount share '%s': %w", n.Share, err)
	}

	nasFile, err := share.Open(filePath)
	if err != nil {
		_ = share.Umount()
		_ = session.Logoff()

		return nil, fmt.Errorf("failed to open file from NAS: %w", err)
	}

	if offsetBytes > 0 {
		if _, err := nasFile.Seek(offsetBytes, io.SeekStart); err != nil {
			_ = nasFile.Close()
			_ = share.Umount()
			_ = session.Logoff()

			return nil, fmt.Errorf("failed to resume NAS read at offset %d: %w", offsetBytes, err)
		}
	}

	// The SMB file takes no context, so a read blocked on a silent server would ignore the
	// caller's cancellation; closing the file is what unblocks it.
	stopCloseOnCancel := context.AfterFunc(attemptCtx, func() {
		_ = nasFile.Close()
	})

	return &nasFileReader{
		file:              nasFile,
		share:             share,
		session:           session,
		stopCloseOnCancel: stopCloseOnCancel,
	}, nil
}
