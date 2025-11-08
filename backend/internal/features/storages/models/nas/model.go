package nas_storage

import (
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

func (n *NASStorage) SaveFile(logger *slog.Logger, fileID uuid.UUID, file io.Reader) error {
	logger.Info("Starting to save file to NAS storage", "fileId", fileID.String(), "host", n.Host)

	session, err := n.createSession()
	if err != nil {
		logger.Error("Failed to create NAS session", "fileId", fileID.String(), "error", err)
		return fmt.Errorf("failed to create NAS session: %w", err)
	}
	defer func() {
		if logoffErr := session.Logoff(); logoffErr != nil {
			logger.Error(
				"Failed to logoff NAS session",
				"fileId",
				fileID.String(),
				"error",
				logoffErr,
			)
		}
	}()

	fs, err := session.Mount(n.Share)
	if err != nil {
		logger.Error(
			"Failed to mount NAS share",
			"fileId",
			fileID.String(),
			"share",
			n.Share,
			"error",
			err,
		)
		return fmt.Errorf("failed to mount share '%s': %w", n.Share, err)
	}
	defer func() {
		if umountErr := fs.Umount(); umountErr != nil {
			logger.Error(
				"Failed to unmount NAS share",
				"fileId",
				fileID.String(),
				"error",
				umountErr,
			)
		}
	}()

	// Ensure the directory exists
	if n.Path != "" {
		if err := n.ensureDirectory(fs, n.Path); err != nil {
			logger.Error(
				"Failed to ensure directory",
				"fileId",
				fileID.String(),
				"path",
				n.Path,
				"error",
				err,
			)
			return fmt.Errorf("failed to ensure directory: %w", err)
		}
	}

	filePath := n.getFilePath(fileID.String())
	logger.Debug("Creating file on NAS", "fileId", fileID.String(), "filePath", filePath)

	nasFile, err := fs.Create(filePath)
	if err != nil {
		logger.Error(
			"Failed to create file on NAS",
			"fileId",
			fileID.String(),
			"filePath",
			filePath,
			"error",
			err,
		)
		return fmt.Errorf("failed to create file on NAS: %w", err)
	}
	defer func() {
		if closeErr := nasFile.Close(); closeErr != nil {
			logger.Error("Failed to close NAS file", "fileId", fileID.String(), "error", closeErr)
		}
	}()

	logger.Debug("Copying file data to NAS", "fileId", fileID.String())
	_, err = io.Copy(nasFile, file)
	if err != nil {
		logger.Error("Failed to write file to NAS", "fileId", fileID.String(), "error", err)
		return fmt.Errorf("failed to write file to NAS: %w", err)
	}

	logger.Info(
		"Successfully saved file to NAS storage",
		"fileId",
		fileID.String(),
		"filePath",
		filePath,
	)
	return nil
}

func (n *NASStorage) GetFile(fileID uuid.UUID) (io.ReadCloser, error) {
	session, err := n.createSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create NAS session: %w", err)
	}

	fs, err := session.Mount(n.Share)
	if err != nil {
		_ = session.Logoff()
		return nil, fmt.Errorf("failed to mount share '%s': %w", n.Share, err)
	}

	filePath := n.getFilePath(fileID.String())

	// Check if file exists
	_, err = fs.Stat(filePath)
	if err != nil {
		_ = fs.Umount()
		_ = session.Logoff()
		return nil, fmt.Errorf("file not found: %s", fileID.String())
	}

	nasFile, err := fs.Open(filePath)
	if err != nil {
		_ = fs.Umount()
		_ = session.Logoff()
		return nil, fmt.Errorf("failed to open file from NAS: %w", err)
	}

	// Return a wrapped reader that cleans up resources when closed
	return &nasFileReader{
		file:    nasFile,
		fs:      fs,
		session: session,
	}, nil
}

func (n *NASStorage) DeleteFile(fileID uuid.UUID) error {
	session, err := n.createSession()
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

	filePath := n.getFilePath(fileID.String())

	// Check if file exists before trying to delete
	_, err = fs.Stat(filePath)
	if err != nil {
		// File doesn't exist, consider it already deleted
		return nil
	}

	err = fs.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file from NAS: %w", err)
	}

	return nil
}

func (n *NASStorage) Validate() error {
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

	// Test the configuration by creating a session
	return n.TestConnection()
}

func (n *NASStorage) TestConnection() error {
	session, err := n.createSession()
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

func (n *NASStorage) createSession() (*smb2.Session, error) {
	// Create connection with timeout
	conn, err := n.createConnection()
	if err != nil {
		return nil, err
	}

	// Create SMB2 dialer
	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     n.Username,
			Password: n.Password,
			Domain:   n.Domain,
		},
	}

	// Create session
	session, err := d.Dial(conn)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to create SMB session: %w", err)
	}

	return session, nil
}

func (n *NASStorage) createConnection() (net.Conn, error) {
	address := net.JoinHostPort(n.Host, fmt.Sprintf("%d", n.Port))

	// Create connection with timeout
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	if n.UseSSL {
		// Use TLS connection
		tlsConfig := &tls.Config{
			ServerName:         n.Host,
			InsecureSkipVerify: false, // Change to true if you want to skip cert verification
		}

		conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create SSL connection to %s: %w", address, err)
		}
		return conn, nil
	} else {
		// Use regular TCP connection
		conn, err := dialer.Dial("tcp", address)
		if err != nil {
			return nil, fmt.Errorf("failed to create connection to %s: %w", address, err)
		}
		return conn, nil
	}
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
			err = fs.Mkdir(currentPath, 0755)
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

// nasFileReader wraps the NAS file and handles cleanup of resources
type nasFileReader struct {
	file    *smb2.File
	fs      *smb2.Share
	session *smb2.Session
}

func (r *nasFileReader) Read(p []byte) (n int, err error) {
	return r.file.Read(p)
}

func (r *nasFileReader) Close() error {
	// Close resources in reverse order
	var errors []error

	if r.file != nil {
		if err := r.file.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close file: %w", err))
		}
	}

	if r.fs != nil {
		if err := r.fs.Umount(); err != nil {
			errors = append(errors, fmt.Errorf("failed to unmount share: %w", err))
		}
	}

	if r.session != nil {
		if err := r.session.Logoff(); err != nil {
			errors = append(errors, fmt.Errorf("failed to logoff session: %w", err))
		}
	}

	if len(errors) > 0 {
		// Return the first error, but log others if needed
		return errors[0]
	}

	return nil
}
