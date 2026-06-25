import type { LogicalBackup } from './LogicalBackup';

export interface GetLogicalBackupsResponse {
  backups: LogicalBackup[];
  total: number;
  limit: number;
  offset: number;
}
