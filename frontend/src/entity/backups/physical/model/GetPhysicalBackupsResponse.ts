import type { PhysicalBackupListItem } from './PhysicalBackupListItem';

export interface GetPhysicalBackupsResponse {
  backups: PhysicalBackupListItem[];
  totalUsageMb: number;

  // continues the list after this page; absent on the last page
  nextCursor?: string;
}
