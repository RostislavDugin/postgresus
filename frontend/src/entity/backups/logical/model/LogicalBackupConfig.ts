import type { Period } from '../../../databases/model/Period';
import type { Interval } from '../../../intervals';
import type { Storage } from '../../../storages';
import { BackupEncryption } from '../../shared';
import type { LogicalBackupNotificationType } from './LogicalBackupNotificationType';
import type { LogicalRetentionPolicyType } from './LogicalRetentionPolicyType';

export interface LogicalBackupConfig {
  databaseId: string;

  isBackupsEnabled: boolean;

  retentionPolicyType: LogicalRetentionPolicyType;
  retentionTimePeriod: Period;
  retentionCount: number;
  retentionGfsHours: number;
  retentionGfsDays: number;
  retentionGfsWeeks: number;
  retentionGfsMonths: number;
  retentionGfsYears: number;

  backupInterval?: Interval;
  storage?: Storage;
  sendNotificationsOn: LogicalBackupNotificationType[];
  isRetryIfFailed: boolean;
  maxFailedTriesCount: number;
  encryption: BackupEncryption;
}
