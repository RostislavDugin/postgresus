import type { PhysicalBackupStatus } from './PhysicalBackupStatus';
import type { PhysicalBackupType } from './PhysicalBackupType';

// One row of the flat physical-backup list: a FULL, an incremental, or a committed
// WAL segment. Type-only fields are optional so unrelated rows omit them rather
// than carrying a misleading zero value (chain links exist on incrementals only,
// walFilename on WAL only).
export interface PhysicalBackupListItem {
  id: string;
  type: PhysicalBackupType;
  status: PhysicalBackupStatus;
  timelineId: number;

  startLsn: string;
  stopLsn: string;

  // chain links, incremental rows only
  rootFullBackupId?: string;
  parentIncrementalBackupId?: string;

  // bare PostgreSQL segment name, WAL rows only
  walFilename?: string;

  sizeMb: number;

  // human-readable failure detail, ERROR / CHAIN_BROKEN rows only
  failMessage?: string;

  createdAt: Date;
  completedAt?: Date;
}
