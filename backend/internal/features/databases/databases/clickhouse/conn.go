package clickhouse

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

const (
	connDialTimeout = 5 * time.Second
	connReadTimeout = 30 * time.Second
)

// OpenConn opens a native-protocol connection to the configured ClickHouse
// server. Caller is responsible for Close.
func OpenConn(ctx context.Context, c *ClickhouseDatabase, password string) (driver.Conn, error) {
	if c == nil {
		return nil, errors.New("nil ClickhouseDatabase")
	}

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)

	opts := &clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: c.Database,
			Username: c.Username,
			Password: password,
		},
		DialTimeout: connDialTimeout,
		ReadTimeout: connReadTimeout,
	}
	if c.IsHttps {
		opts.TLS = &tls.Config{
			MinVersion: tls.VersionTLS12,
			// Default to skipping cert verification (parity with PostgreSQL's
			// `sslmode=require`). Operators can opt into strict verification
			// against system trust roots via the IsStrictTls toggle.
			InsecureSkipVerify: !c.IsStrictTls,
		}
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open clickhouse: %w", err)
	}

	// Verify the connection now rather than on first query — surfaces auth/TLS
	// failures with a deterministic error site.
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}

	return conn, nil
}

// DetectClickhouseVersion queries `SELECT version()` and parses the major.minor.
func DetectClickhouseVersion(ctx context.Context, conn driver.Conn) (tools.ClickhouseVersion, error) {
	var raw string
	if err := conn.QueryRow(ctx, "SELECT version()").Scan(&raw); err != nil {
		return "", fmt.Errorf("query version: %w", err)
	}

	return tools.GetClickhouseVersionEnum(raw), nil
}

// decryptPasswordIfNeeded centralises the encrypt-or-passthrough pattern used
// by every method that needs the cleartext password.
func decryptPasswordIfNeeded(
	password string,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) (string, error) {
	if encryptor == nil {
		return password, nil
	}

	return encryptor.Decrypt(databaseID, password)
}
