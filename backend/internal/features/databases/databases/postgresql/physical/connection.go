package postgresql_physical

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

// The `postgres` database always exists, and the physical model has no per-DB selection.
func openConn(
	ctx context.Context,
	p *PostgresqlPhysicalDatabase,
	encryptor encryption.FieldEncryptor,
) (*pgx.Conn, error) {
	password, err := postgresql_shared.DecryptFieldIfNeeded(p.Password, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	files, err := postgresql_shared.WriteCredentialFilesToTempDir(p.CredentialSpec(), password, encryptor)
	if err != nil {
		return nil, err
	}
	defer files.Remove()

	connConfig, err := postgresql_shared.BuildConnConfig(p.CredentialSpec(), password, "postgres", files)
	if err != nil {
		return nil, err
	}

	return pgx.ConnectConfig(ctx, connConfig)
}

// This is what exercises the "host replication" pg_hba path and the REPLICATION privilege at connect
// time; an ordinary "host all" rule does NOT cover it, so a logical (replication=database) probe
// would wrongly accept a cluster that real backups cannot stream. It uses the low-level pgconn
// because no ordinary SQL is allowed on a physical replication connection.
func openPhysicalReplicationConn(
	ctx context.Context,
	p *PostgresqlPhysicalDatabase,
	encryptor encryption.FieldEncryptor,
) (*pgconn.PgConn, error) {
	password, err := postgresql_shared.DecryptFieldIfNeeded(p.Password, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	files, err := postgresql_shared.WriteCredentialFilesToTempDir(p.CredentialSpec(), password, encryptor)
	if err != nil {
		return nil, err
	}
	defer files.Remove()

	connConfig, err := postgresql_shared.BuildPhysicalReplicationConnConfig(
		p.CredentialSpec(), password, "postgres", files,
	)
	if err != nil {
		return nil, err
	}

	return pgconn.ConnectConfig(ctx, connConfig)
}

func closeConnQuietly(ctx context.Context, conn *pgx.Conn, logger *slog.Logger) {
	if err := conn.Close(ctx); err != nil {
		logger.WarnContext(ctx, "failed to close connection", "error", err)
	}
}

var versionRegexp = regexp.MustCompile(`PostgreSQL (\d+)`)

func detectVersion(ctx context.Context, conn *pgx.Conn) (tools.PostgresqlVersion, error) {
	var rawServerVersion string
	if err := conn.QueryRow(ctx, "SELECT version()").Scan(&rawServerVersion); err != nil {
		return "", fmt.Errorf("failed to query version(): %w", err)
	}

	matches := versionRegexp.FindStringSubmatch(rawServerVersion)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse version from: %s", rawServerVersion)
	}

	switch matches[1] {
	case "17":
		return tools.PostgresqlVersion17, nil
	case "18":
		return tools.PostgresqlVersion18, nil
	default:
		return "", fmt.Errorf("physical backup requires PostgreSQL 17 or 18, detected %s", matches[1])
	}
}
