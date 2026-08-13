package postgresql_physical

import (
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
)

// Every libpq path goes through here: the pgx inspection and replication connections, pg_basebackup
// and pg_receivewal.
func (p *PostgresqlPhysicalDatabase) CredentialSpec() postgresql_shared.CredentialSpec {
	hostAddr := ""
	if p.LocalTunnelEndpoint != nil {
		hostAddr = p.LocalTunnelEndpoint.Host
	}

	return postgresql_shared.CredentialSpec{
		Host:          p.Host,
		HostAddr:      hostAddr,
		Port:          p.Port,
		Username:      p.Username,
		SslMode:       p.SslMode,
		SslClientCert: p.SslClientCert,
		SslClientKey:  p.SslClientKey,
		SslRootCert:   p.SslRootCert,
	}
}
