import type { PhysicalBackupListItem } from './PhysicalBackupListItem';

export interface GetPhysicalBackupsResponse {
  backups: PhysicalBackupListItem[];
  totalUsageMb: number;

  // total counts every backup type for the database (drives pagination); limit and
  // offset echo the page that was served
  total: number;
  limit: number;
  offset: number;
}
