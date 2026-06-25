package postgresql_logical

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/util/encryption"
)

// CredentialSpec maps this database into the strategy-agnostic credential inputs
// shared by every libpq path: the pgx connections here and pg_dump / pg_restore
// in the backup and restore usecases.
func (p *PostgresqlLogicalDatabase) CredentialSpec() postgresql_shared.CredentialSpec {
	return postgresql_shared.CredentialSpec{
		Host:          p.Host,
		Port:          p.Port,
		Username:      p.Username,
		SslMode:       p.SslMode,
		SslClientCert: p.SslClientCert,
		SslClientKey:  p.SslClientKey,
		SslRootCert:   p.SslRootCert,
	}
}

// openPgConn writes p's credential files, opens a pgx connection to dbName, and
// removes the files once the TLS handshake has completed.
func openPgConn(
	ctx context.Context,
	p *PostgresqlLogicalDatabase,
	dbName string,
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

	return pgx.Connect(ctx, postgresql_shared.BuildConnString(p.CredentialSpec(), password, dbName, files))
}
