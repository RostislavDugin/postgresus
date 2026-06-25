package postgresql_physical

import (
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
)

// CredentialSpec maps this database into the strategy-agnostic credential inputs
// shared by every libpq path: the pgx inspection / replication connections here
// and pg_basebackup in the backup usecase.
func (p *PostgresqlPhysicalDatabase) CredentialSpec() postgresql_shared.CredentialSpec {
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
