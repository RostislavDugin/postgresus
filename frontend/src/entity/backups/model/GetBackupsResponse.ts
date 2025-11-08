import type { Backup } from './Backup';

export interface GetBackupsResponse {
  backups: Backup[];
  total: number;
  limit: number;
  offset: number;
}
