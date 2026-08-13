import type { SshTunnelConfig } from '../../sshtunnel/SshTunnelConfig';
import type { PostgresSslMode } from '../PostgresSslMode';
import type { PostgresqlVersion } from '../PostgresqlVersion';
import type { PhysicalDatabaseBackupType } from './PhysicalDatabaseBackupType';

export interface PostgresqlPhysicalDatabase {
  id: string;
  databaseId?: string;

  // physical backups run against PostgreSQL 17 or 18 only; the server detects the
  // exact version on connect, so this is read-only after the first test-connection
  version: PostgresqlVersion;
  backupType: PhysicalDatabaseBackupType;

  // connection data (physical connects to the cluster, not a single database)
  host: string;
  port: number;
  username: string;
  password: string;

  // SSL / TLS
  sslMode: PostgresSslMode;
  sslClientCert?: string;
  sslClientKey?: string;
  sslRootCert?: string;

  // When enabled, host and port above address the cluster as the bastion sees it.
  sshTunnel?: SshTunnelConfig;

  // server-managed identity, read-only - set by the backend on first connect and
  // immutable afterwards (a changed value means the cluster was swapped)
  systemIdentifier?: string;
  walSegmentSizeBytes?: number;
}
