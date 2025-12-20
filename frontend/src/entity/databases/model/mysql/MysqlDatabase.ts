import type { MysqlVersion } from './MysqlVersion';

export interface MysqlDatabase {
  id: string;
  version: MysqlVersion;

  host: string;
  port: number;
  username: string;
  password: string;
  database?: string;
  isHttps: boolean;
}
