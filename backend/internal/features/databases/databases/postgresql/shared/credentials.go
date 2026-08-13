package postgresql_shared

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

type CredentialSpec struct {
	Host string
	// Named after the libpq parameter it maps to: when set, the connection is made to this address
	// while Host is what the certificate and .pgpass are matched against. That split is what lets an
	// SSH tunnel forward the traffic without breaking sslmode=verify-full.
	HostAddr      string
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

func buildConnString(
	spec CredentialSpec,
	password, dbName string,
	files *CredentialTempFiles,
) string {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password='%s' dbname=%s sslmode=%s",
		getConnectHost(spec),
		spec.Port,
		spec.Username,
		escapeConnStringValue(password),
		dbName,
		sslModeOrDefault(spec),
	)

	connStr += " default_query_exec_mode=simple_protocol standard_conforming_strings=on client_encoding=UTF8"

	return appendSslFilePaths(connStr, files)
}

// replication=true is the mode pg_basebackup / pg_receivewal use, and the one pg_hba matches via
// "host replication" rules rather than ordinary "host all" rules. The pgx-only query-exec params are
// omitted so the string stays consumable by the low-level pgconn used for the connectivity probe.
func buildPhysicalReplicationConnString(
	spec CredentialSpec,
	password, dbName string,
	files *CredentialTempFiles,
) string {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password='%s' dbname=%s sslmode=%s replication=true",
		getConnectHost(spec),
		spec.Port,
		spec.Username,
		escapeConnStringValue(password),
		dbName,
		sslModeOrDefault(spec),
	)

	return appendSslFilePaths(connStr, files)
}

// pgx has no hostaddr parameter, so through a tunnel the address goes in as the host and the real
// name is restored on the TLS config, which is what sslmode=verify-full matches against.
func BuildConnConfig(
	spec CredentialSpec,
	password, dbName string,
	files *CredentialTempFiles,
) (*pgx.ConnConfig, error) {
	connConfig, err := pgx.ParseConfig(buildConnString(spec, password, dbName, files))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the PostgreSQL connection string: %w", err)
	}

	restoreServerNameForTunneledHost(&connConfig.Config, spec)

	return connConfig, nil
}

// pgconn has no hostaddr either, and unlike the pgx path there is no ParseConfig step in the caller,
// so the ServerName restoration has to happen here or verify-full would match the tunnel's loopback.
func BuildPhysicalReplicationConnConfig(
	spec CredentialSpec,
	password, dbName string,
	files *CredentialTempFiles,
) (*pgconn.Config, error) {
	connConfig, err := pgconn.ParseConfig(buildPhysicalReplicationConnString(spec, password, dbName, files))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the PostgreSQL replication connection string: %w", err)
	}

	restoreServerNameForTunneledHost(connConfig, spec)

	return connConfig, nil
}

// pgx builds one fallback config per sslmode it may retry with, so skipping those would silently
// verify against the tunnel address on the retry.
func restoreServerNameForTunneledHost(config *pgconn.Config, spec CredentialSpec) {
	if spec.HostAddr == "" {
		return
	}

	if config.TLSConfig != nil {
		config.TLSConfig.ServerName = spec.Host
	}

	for _, fallback := range config.Fallbacks {
		if fallback.TLSConfig != nil {
			fallback.TLSConfig.ServerName = spec.Host
		}
	}
}

// libpq resolves and connects to hostaddr while still matching the certificate and .pgpass against
// the host argument, so the CLI tools need no rewritten -h.
func GetPgHostAddrEnv(spec CredentialSpec) []string {
	if spec.HostAddr == "" {
		return nil
	}

	return []string{"PGHOSTADDR=" + spec.HostAddr}
}

func getConnectHost(spec CredentialSpec) string {
	if spec.HostAddr != "" {
		return spec.HostAddr
	}

	return spec.Host
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
