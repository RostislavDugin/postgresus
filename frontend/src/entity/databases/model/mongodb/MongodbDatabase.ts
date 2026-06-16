import type { MongodbVersion } from './MongodbVersion';

export interface MongodbDatabase {
  id: string;
  version: MongodbVersion;
  host: string;
  port: number;
  username: string;
  password: string;
  database: string;
  authDatabase: string;
  isHttps: boolean;
  isSrv: boolean;
  isDirectConnection: boolean;
  cpuCount: number;
  excludeCollections?: string[];
  includeCollections?: string[];
  restoreIncludeCollections?: string[];
  restoreExcludeCollections?: string[];
}
