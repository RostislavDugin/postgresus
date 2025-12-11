import type { PostgresqlVersion } from './PostgresqlVersion';

export interface PostgresqlDatabase {
  id: string;
  version: PostgresqlVersion;

  // connection data
  host: string;
  port: number;
  username: string;
  password: string;
  database?: string;
  isHttps: boolean;

  // backup settings
  includeSchemas?: string[];
}
