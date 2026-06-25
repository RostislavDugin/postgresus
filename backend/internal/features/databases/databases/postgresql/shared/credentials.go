package postgresql_shared

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

type CredentialSpec struct {
	Host          string
	Port          int
	Username      string
	SslMode       PostgresSslMode
	SslClientCert string
	SslClientKey  string
	SslRootCert   string
}

type CredentialTempFiles struct {
	Dir            string
	PgpassPath     string
	ClientCertPath string
	ClientKeyPath  string
	RootCertPath   string
}

func WriteCredentialFilesToTempDir(
	spec CredentialSpec,
	password string,
	encryptor encryption.FieldEncryptor,
) (*CredentialTempFiles, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "pgcreds_"+uuid.New().String())
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)

		return nil, fmt.Errorf("failed to set temporary directory permissions: %w", err)
	}

	files := &CredentialTempFiles{Dir: dir}

	if err := files.writePgpass(spec, password); err != nil {
		_ = os.RemoveAll(dir)

		return nil, err
	}

	if spec.SslMode != PostgresSslModeDisable && spec.SslMode != "" {
		if err := files.writeCertFiles(spec, encryptor); err != nil {
			_ = os.RemoveAll(dir)

			return nil, err
		}
	}

	return files, nil
}

func (f *CredentialTempFiles) Remove() {
	if f == nil || f.Dir == "" {
		return
	}

	_ = os.RemoveAll(f.Dir)
}

func BuildConnString(
	spec CredentialSpec,
	password, dbName string,
	files *CredentialTempFiles,
) string {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password='%s' dbname=%s sslmode=%s",
		spec.Host,
		spec.Port,
		spec.Username,
		escapeConnStringValue(password),
		dbName,
		sslModeOrDefault(spec),
	)

	connStr += " default_query_exec_mode=simple_protocol standard_conforming_strings=on client_encoding=UTF8"

	return appendSslFilePaths(connStr, files)
}

// BuildPhysicalReplicationConnString builds a libpq conninfo for a PHYSICAL replication connection
// (replication=true) — the mode pg_basebackup / pg_receivewal use, and the one pg_hba matches via
// "host replication" rules rather than ordinary "host all" rules. It omits the pgx-only query-exec
// params so the string is consumable by the low-level pgconn.Connect used for the connectivity probe.
func BuildPhysicalReplicationConnString(
	spec CredentialSpec,
	password, dbName string,
	files *CredentialTempFiles,
) string {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password='%s' dbname=%s sslmode=%s replication=true",
		spec.Host,
		spec.Port,
		spec.Username,
		escapeConnStringValue(password),
		dbName,
		sslModeOrDefault(spec),
	)

	return appendSslFilePaths(connStr, files)
}

func sslModeOrDefault(spec CredentialSpec) PostgresSslMode {
	if spec.SslMode == "" {
		return PostgresSslModeDisable
	}

	return spec.SslMode
}

func appendSslFilePaths(connStr string, files *CredentialTempFiles) string {
	if files == nil {
		return connStr
	}

	if files.ClientCertPath != "" {
		connStr += fmt.Sprintf(" sslcert='%s'", escapeConnStringValue(files.ClientCertPath))
	}

	if files.ClientKeyPath != "" {
		connStr += fmt.Sprintf(" sslkey='%s'", escapeConnStringValue(files.ClientKeyPath))
	}

	if files.RootCertPath != "" {
		connStr += fmt.Sprintf(" sslrootcert='%s'", escapeConnStringValue(files.RootCertPath))
	}

	return connStr
}

// DecryptFieldIfNeeded decrypts an encrypted field, or returns it unchanged when
// encryptor is nil (plaintext input, e.g. a restore target config never persisted).
func DecryptFieldIfNeeded(value string, encryptor encryption.FieldEncryptor) (string, error) {
	if encryptor == nil {
		return value, nil
	}

	return encryptor.Decrypt(value)
}

func (f *CredentialTempFiles) writePgpass(spec CredentialSpec, password string) error {
	content := fmt.Sprintf("%s:%d:*:%s:%s",
		tools.EscapePgpassField(spec.Host),
		spec.Port,
		tools.EscapePgpassField(spec.Username),
		tools.EscapePgpassField(password),
	)

	path := filepath.Join(f.Dir, ".pgpass")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write .pgpass file: %w", err)
	}

	f.PgpassPath = path

	return nil
}

func (f *CredentialTempFiles) writeCertFiles(spec CredentialSpec, encryptor encryption.FieldEncryptor) error {
	var err error

	if f.ClientCertPath, err = f.writeCert("client.crt", spec.SslClientCert, encryptor); err != nil {
		return fmt.Errorf("failed to write client certificate: %w", err)
	}

	if f.ClientKeyPath, err = f.writeCert("client.key", spec.SslClientKey, encryptor); err != nil {
		return fmt.Errorf("failed to write client key: %w", err)
	}

	if f.RootCertPath, err = f.writeCert("root.crt", spec.SslRootCert, encryptor); err != nil {
		return fmt.Errorf("failed to write server CA certificate: %w", err)
	}

	return nil
}

func (f *CredentialTempFiles) writeCert(
	fileName, encryptedPEM string,
	encryptor encryption.FieldEncryptor,
) (string, error) {
	if encryptedPEM == "" {
		return "", nil
	}

	pem, err := DecryptFieldIfNeeded(encryptedPEM, encryptor)
	if err != nil {
		return "", err
	}

	path := filepath.Join(f.Dir, fileName)
	if err := os.WriteFile(path, []byte(pem), 0o600); err != nil {
		return "", err
	}

	return path, nil
}

func escapeConnStringValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)

	return value
}
