import type { Interval } from '../../../intervals';
import type { Storage } from '../../../storages';
import { BackupEncryption } from '../../shared';
import type { ChainsRetention } from './ChainsRetention';
import type { FullBackupsRetention } from './FullBackupsRetention';
import type { PhysicalBackupNotificationType } from './PhysicalBackupNotificationType';
import type { PhysicalRetention } from './PhysicalRetention';

export interface PhysicalBackupConfig {
  databaseId: string;

  isBackupsEnabled: boolean;

  fullBackupInterval?: Interval;
  incrementalBackupInterval?: Interval;

  retention: PhysicalRetention;
  chainsRetention: ChainsRetention;
  fullBackupsRetention: FullBackupsRetention;

  // slot-rebuild trigger for WAL streaming; must be 0 unless backupType is
  // FULL_INCREMENTAL_WAL_STREAM
  walLagThresholdBytes: number;

  storage?: Storage;

  encryption: BackupEncryption;
  sendNotificationsOn: PhysicalBackupNotificationType[];
}
