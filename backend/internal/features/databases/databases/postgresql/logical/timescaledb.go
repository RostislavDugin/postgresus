package postgresql_logical

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/util/encryption"
)

// GetTimescaleDBVersion returns the installed timescaledb extension version on this database,
// or "" when the extension is absent. Captured at backup time so restore knows whether the
// TimescaleDB restore procedure is required, and the verification agent knows which engine
// image version to pull.
func (p *PostgresqlLogicalDatabase) GetTimescaleDBVersion(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) (string, error) {
	conn, err := openPgConn(ctx, p, *p.Database, encryptor)
	if err != nil {
		return "", fmt.Errorf("failed to connect to database '%s': %w", *p.Database, err)
	}
	defer func() { _ = conn.Close(ctx) }()

	var version string

	err = conn.QueryRow(
		ctx, "SELECT extversion FROM pg_extension WHERE extname = 'timescaledb'",
	).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("failed to query timescaledb extension version: %w", err)
	}

	return version, nil
}

// TimescaleDB restore procedure, executed against the restore target.
//
// timescaledb_pre_restore() runs `ALTER DATABASE <db> SET timescaledb.restoring='on'` — a
// database-level GUC that persists across new connections — and stops background workers. Because
// the GUC lives on the database (not the session), the separate pg_restore process that runs next
// inherits restoring mode; that is the whole reason these hooks can run on connections distinct
// from pg_restore's. timescaledb_post_restore() resets the GUC and restarts the workers, so it MUST
// run even when pg_restore fails — otherwise the database is left stuck in restoring mode with its
// workers stopped. Callers run RunTimescaleDBPostRestore via defer.

// RunTimescaleDBPreRestore installs the extension (a freshly created restore target has none, and
// pre_restore needs its functions present) and enters restoring mode.
func (p *PostgresqlLogicalDatabase) RunTimescaleDBPreRestore(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) error {
	return p.execOnTarget(ctx, encryptor,
		"CREATE EXTENSION IF NOT EXISTS timescaledb",
		"SELECT timescaledb_pre_restore()",
	)
}

func (p *PostgresqlLogicalDatabase) RunTimescaleDBPostRestore(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) error {
	return p.execOnTarget(ctx, encryptor, "SELECT timescaledb_post_restore()")
}

func (p *PostgresqlLogicalDatabase) execOnTarget(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	statements ...string,
) error {
	conn, err := openPgConn(ctx, p, *p.Database, encryptor)
	if err != nil {
		return fmt.Errorf("failed to connect to database '%s': %w", *p.Database, err)
	}
	defer func() { _ = conn.Close(ctx) }()

	for _, statement := range statements {
		if _, err := conn.Exec(ctx, statement); err != nil {
			return fmt.Errorf("failed to run %q: %w", statement, err)
		}
	}

	return nil
}
