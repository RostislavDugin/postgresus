package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"

	"databasus-backend/internal/util/encryption"
)

// ClickHouse server error codes (from src/Common/ErrorCodes.cpp upstream).
const (
	chErrAccessDenied              = 497
	chErrAccessEntityAlreadyExists = 493
	chErrAccessStorageReadOnly     = 495
)

// readOnlyWriteAccessTypes lists every access_type in system.grants that
// indicates the connecting user is NOT a strict read-only user. Membership
// in any of these for any database (or globally with database = '') triggers
// rejection.
var readOnlyWriteAccessTypes = []string{
	"INSERT",
	"ALTER",
	"DROP",
	"CREATE",
	"TRUNCATE",
	"OPTIMIZE",
	"SYSTEM",
	"KILL QUERY",
	"ALL",
}

// IsUserReadOnly returns (true, nil, nil) when the configured connection user
// has zero write privileges in system.grants — INSERT/ALTER/DROP/CREATE/
// TRUNCATE/OPTIMIZE/SYSTEM/KILL QUERY/ALL anywhere. Returns (false, list, nil)
// with the matching access_types if any write privilege is present.
func (c *ClickhouseDatabase) IsUserReadOnly(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) (bool, []string, error) {
	password, err := decryptPasswordIfNeeded(c.Password, encryptor, databaseID)
	if err != nil {
		return false, nil, fmt.Errorf("decrypt password: %w", err)
	}

	conn, err := OpenConn(ctx, c, password)
	if err != nil {
		return false, nil, err
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			logger.Error("failed to close clickhouse connection", "error", closeErr)
		}
	}()

	rows, err := conn.Query(ctx, `
		SELECT access_type
		FROM system.grants
		WHERE user_name = currentUser()
		  AND access_type IN (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		readOnlyWriteAccessTypes[0],
		readOnlyWriteAccessTypes[1],
		readOnlyWriteAccessTypes[2],
		readOnlyWriteAccessTypes[3],
		readOnlyWriteAccessTypes[4],
		readOnlyWriteAccessTypes[5],
		readOnlyWriteAccessTypes[6],
		readOnlyWriteAccessTypes[7],
		readOnlyWriteAccessTypes[8],
	)
	if err != nil {
		return false, nil, fmt.Errorf("query system.grants: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var found []string
	seen := map[string]bool{}
	for rows.Next() {
		var accessType string
		if err := rows.Scan(&accessType); err != nil {
			return false, nil, fmt.Errorf("scan system.grants row: %w", err)
		}
		if !seen[accessType] {
			seen[accessType] = true
			found = append(found, accessType)
		}
	}
	if err := rows.Err(); err != nil {
		return false, nil, fmt.Errorf("iterate system.grants: %w", err)
	}

	return len(found) == 0, found, nil
}

// CreateReadOnlyUser provisions a new ClickHouse user `databasus-XXXXXXXX`
// scoped to the configured database with strict read-only privileges, and
// returns the new credentials. The caller is responsible for storing them
// (encrypted) on the Database record so subsequent backups connect as the
// new user instead of the admin.
//
// SQL pattern (verified against CH 23.8 - 25.x docs):
//
//	CREATE USER `databasus-xxxxxxxx`
//	  IDENTIFIED WITH sha256_password BY '<generated>'
//	  SETTINGS readonly = 2 READONLY;
//	GRANT SELECT, SHOW DATABASES, SHOW TABLES, SHOW COLUMNS, SHOW DICTIONARIES
//	  ON `<database>`.*
//	  TO `databasus-xxxxxxxx`;
//
// readonly = 2 disallows every write (INSERT, ALTER, DROP, CREATE, TRUNCATE,
// OPTIMIZE, SYSTEM) but permits per-query SET commands. The clickhouse-go
// driver sets query-level settings such as `max_execution_time` on every
// connection; readonly = 1 blocks those and produces a confusing
// "Cannot modify 'max_execution_time'" failure on the very first query.
// The trailing READONLY keyword pins the binding so the user cannot lower
// readonly at runtime.
//
// Permission preflight is implicit: if the connecting user lacks ACCESS
// MANAGEMENT, CREATE USER returns ACCESS_DENIED (497) and we surface a
// descriptive error. CHECK GRANT (CH 24.10+) would be cleaner but is not
// available on every supported version, so the error-code path is the
// portable choice.
func (c *ClickhouseDatabase) CreateReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) (string, string, error) {
	password, err := decryptPasswordIfNeeded(c.Password, encryptor, databaseID)
	if err != nil {
		return "", "", fmt.Errorf("decrypt password: %w", err)
	}

	conn, err := OpenConn(ctx, c, password)
	if err != nil {
		return "", "", err
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			logger.Error("failed to close clickhouse connection", "error", closeErr)
		}
	}()

	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		username := fmt.Sprintf("databasus-%s", uuid.New().String()[:8])
		newPassword := encryption.GenerateComplexPassword()

		createSQL := fmt.Sprintf(
			"CREATE USER %s IDENTIFIED WITH sha256_password BY '%s' SETTINGS readonly = 2 READONLY",
			quoteIdent(username),
			escapeSQLString(newPassword),
		)
		if err := conn.Exec(ctx, createSQL); err != nil {
			switch chErrCode(err) {
			case chErrAccessEntityAlreadyExists:
				if attempt < maxRetries-1 {
					continue
				}
				return "", "", fmt.Errorf("could not generate a unique username after %d attempts", maxRetries)
			case chErrAccessDenied:
				return "", "", fmt.Errorf(
					"admin user lacks ACCESS MANAGEMENT privilege required to create users: %w",
					err,
				)
			case chErrAccessStorageReadOnly:
				return "", "", fmt.Errorf(
					"ClickHouse access storage is read-only; this managed service does not "+
						"allow user creation via SQL: %w",
					err,
				)
			default:
				return "", "", fmt.Errorf("create user: %w", err)
			}
		}

		grantSQL := fmt.Sprintf(
			"GRANT SELECT, SHOW DATABASES, SHOW TABLES, SHOW COLUMNS, SHOW DICTIONARIES "+
				"ON %s.* TO %s",
			quoteIdent(c.Database),
			quoteIdent(username),
		)
		if grantErr := conn.Exec(ctx, grantSQL); grantErr != nil {
			// CH has no DDL transactions; explicitly drop the partially
			// provisioned user so we don't leak orphans.
			if dropErr := conn.Exec(
				ctx,
				fmt.Sprintf("DROP USER IF EXISTS %s", quoteIdent(username)),
			); dropErr != nil {
				logger.Error(
					"failed to drop user after grant failure; orphan may remain",
					"username", username,
					"drop_error", dropErr,
				)
			}
			return "", "", fmt.Errorf("grant privileges to %s: %w", username, grantErr)
		}

		logger.Info("read-only ClickHouse user created", "username", username)
		return username, newPassword, nil
	}

	return "", "", errors.New("exhausted retries creating read-only user")
}

// chErrCode extracts the ClickHouse server error code from an error
// returned by clickhouse-go/v2. Returns 0 if the error is not a
// clickhouse.Exception (e.g. network or driver-side failure).
func chErrCode(err error) int32 {
	var exc *clickhouse.Exception
	if errors.As(err, &exc) {
		return exc.Code
	}
	return 0
}

// escapeSQLString escapes a value to be safely interpolated into a CH single-
// quoted string literal. CH doubles single quotes to escape; backslashes are
// not special in standard CH single-quoted strings.
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
