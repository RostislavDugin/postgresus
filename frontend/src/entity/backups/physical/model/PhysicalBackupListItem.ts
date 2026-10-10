import type { PhysicalBackupStatus } from './PhysicalBackupStatus';
import type { PhysicalBackupType } from './PhysicalBackupType';

// One row of the flat physical-backup list: a FULL, an incremental, or a WAL range
// of consecutive committed segments. Type-only fields are optional so unrelated
// rows omit them rather than carrying a misleading zero value (chain links exist
// on incrementals only, the range summary on WAL only). A WAL range's createdAt
// and completedAt are the receive times of its first and last segments.
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

  // WAL range summary, WAL rows only: the first and last segment names and how
  // many segments the range holds. hasGapBefore marks a missing segment between
  // this range and the older one on the same timeline.
  fromWalFilename?: string;
  toWalFilename?: string;
  walSegmentsCount?: number;
  hasGapBefore?: boolean;

  sizeMb: number;

  // human-readable failure detail, ERROR / CHAIN_BROKEN rows only
  failMessage?: string;

  createdAt: Date;
  completedAt?: Date;
}
