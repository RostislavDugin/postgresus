import type { Period } from '../../databases/model/Period';
import type { Interval } from '../../intervals';
import type { Storage } from '../../storages';
import { BackupEncryption } from './BackupEncryption';
import type { BackupNotificationType } from './BackupNotificationType';
import type { RetentionPolicyType } from './RetentionPolicyType';

export interface BackupConfig {
  id?: string;
  databaseId: string;
  name: string;

  isBackupsEnabled: boolean;

  retentionPolicyType: RetentionPolicyType;
  retentionTimePeriod: Period;
  retentionCount: number;
  retentionGfsHours: number;
  retentionGfsDays: number;
  retentionGfsWeeks: number;
  retentionGfsMonths: number;
  retentionGfsYears: number;

  backupInterval?: Interval;
  storage?: Storage;
  sendNotificationsOn: BackupNotificationType[];
  isRetryIfFailed: boolean;
  maxFailedTriesCount: number;
  encryption: BackupEncryption;

  maxBackupSizeMb: number;
  maxBackupsTotalSizeMb: number;
}
