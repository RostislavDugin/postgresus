import type { PostgresBackupType } from './PostgresBackupType';
import type { PostgresqlVersion } from './PostgresqlVersion';

export interface PostgresqlDatabase {
  id: string;
  version: PostgresqlVersion;
  backupType?: PostgresBackupType;

  // connection data
  host: string;
  port: number;
  username: string;
  password: string;
  database?: string;
  isHttps: boolean;
  isStrictTls?: boolean;

  // backup settings
  includeSchemas?: string[];
  cpuCount: number;

  // restore settings (not saved to DB)
  isExcludeExtensions?: boolean;
}
