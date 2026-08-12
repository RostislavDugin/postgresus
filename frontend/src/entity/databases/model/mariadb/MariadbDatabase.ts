import type { SshTunnelConfig } from '../sshtunnel/SshTunnelConfig';
import type { MariadbVersion } from './MariadbVersion';

export interface MariadbDatabase {
  id: string;
  version: MariadbVersion;
  host: string;
  port: number;
  username: string;
  password: string;
  database?: string;
  isHttps: boolean;
  sshTunnel?: SshTunnelConfig;
  isExcludeEvents?: boolean;
  isSkipGaleraDisable?: boolean;
  excludeTables?: string[];
}
