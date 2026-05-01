import type { ClickhouseVersion } from './ClickhouseVersion';

export interface ClickhouseDatabase {
  id: string;
  version: ClickhouseVersion;

  host: string;
  port: number;
  username: string;
  password: string;
  database: string;
  isHttps: boolean;
  isStrictTls?: boolean;

  isDropExisting?: boolean;
  isKeepReplicatedDDL?: boolean;
}
