package ftp_storage

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"postgresus-backend/internal/util/encryption"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jlaffaye/ftp"
)

const (
	ftpConnectTimeout = 30 * time.Second
	ftpChunkSize      = 16 * 1024 * 1024
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
	PassiveMode   bool      `json:"passiveMode"   gorm:"not null;default:true;column:passive_mode"`
}

func (f *FTPStorage) TableName() string {
	return "ftp_storages"
}

func (f *FTPStorage) SaveFile(
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

	logger.Info("Starting to save file to FTP storage", "fileId", fileID.String(), "host", f.Host)

	conn, err := f.connect(encryptor)
	if err != nil {
		logger.Error("Failed to connect to FTP", "fileId", fileID.String(), "error", err)
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}
	defer func() {
		if quitErr := conn.Quit(); quitErr != nil {
			logger.Error(
				"Failed to close FTP connection",
				"fileId",
				fileID.String(),
				"error",
				quitErr,
			)
		}
	}()

	if f.Path != "" {
		if err := f.ensureDirectory(conn, f.Path); err != nil {
			logger.Error(
				"Failed to ensure directory",
				"fileId",
				fileID.String(),
				"path",
				f.Path,
				"error",
				err,
			)
			return fmt.Errorf("failed to ensure directory: %w", err)
		}
	}

	filePath := f.getFilePath(fileID.String())
	logger.Debug("Uploading file to FTP", "fileId", fileID.String(), "filePath", filePath)

	ctxReader := &contextReader{ctx: ctx, reader: file}

	err = conn.Stor(filePath, ctxReader)
	if err != nil {
		select {
		case <-ctx.Done():
			logger.Info("FTP upload cancelled", "fileId", fileID.String())
			return ctx.Err()
		default:
			logger.Error("Failed to upload file to FTP", "fileId", fileID.String(), "error", err)
			return fmt.Errorf("failed to upload file to FTP: %w", err)
		}
	}

	logger.Info(
		"Successfully saved file to FTP storage",
		"fileId",
		fileID.String(),
		"filePath",
		filePath,
	)
	return nil
}

func (f *FTPStorage) GetFile(
	encryptor encryption.FieldEncryptor,
	fileID uuid.UUID,
) (io.ReadCloser, error) {
	conn, err := f.connect(encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to FTP: %w", err)
	}

	filePath := f.getFilePath(fileID.String())

	resp, err := conn.Retr(filePath)
	if err != nil {
		_ = conn.Quit()
		return nil, fmt.Errorf("failed to retrieve file from FTP: %w", err)
	}

	return &ftpFileReader{
		response: resp,
		conn:     conn,
	}, nil
}

func (f *FTPStorage) DeleteFile(encryptor encryption.FieldEncryptor, fileID uuid.UUID) error {
	conn, err := f.connect(encryptor)
	if err != nil {
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}
	defer func() {
		_ = conn.Quit()
	}()

	filePath := f.getFilePath(fileID.String())

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
	conn, err := f.connect(encryptor)
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
		encrypted, err := encryptor.Encrypt(f.StorageID, f.Password)
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
	f.PassiveMode = incoming.PassiveMode
	f.Path = incoming.Path

	if incoming.Password != "" {
		f.Password = incoming.Password
	}
}

func (f *FTPStorage) connect(encryptor encryption.FieldEncryptor) (*ftp.ServerConn, error) {
	password, err := encryptor.Decrypt(f.StorageID, f.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt FTP password: %w", err)
	}

	address := fmt.Sprintf("%s:%d", f.Host, f.Port)

	var conn *ftp.ServerConn
	if f.UseSSL {
		tlsConfig := &tls.Config{
			ServerName:         f.Host,
			InsecureSkipVerify: f.SkipTLSVerify,
		}
		conn, err = ftp.Dial(address,
			ftp.DialWithTimeout(ftpConnectTimeout),
			ftp.DialWithExplicitTLS(tlsConfig),
		)
	} else {
		conn, err = ftp.Dial(address, ftp.DialWithTimeout(ftpConnectTimeout))
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

		err := conn.ChangeDir(currentPath)
		if err != nil {
			err = conn.MakeDir(currentPath)
			if err != nil {
				return fmt.Errorf("failed to create directory '%s': %w", currentPath, err)
			}
		}

		err = conn.ChangeDirToParent()
		if err != nil {
			return fmt.Errorf("failed to change to parent directory: %w", err)
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
	conn     *ftp.ServerConn
}

func (r *ftpFileReader) Read(p []byte) (n int, err error) {
	return r.response.Read(p)
}

func (r *ftpFileReader) Close() error {
	var errs []error

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
