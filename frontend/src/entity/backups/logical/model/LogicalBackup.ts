import type { Database } from '../../../databases/model/Database';
import type { Storage } from '../../../storages';
import { BackupEncryption } from '../../shared';
import { LogicalBackupStatus } from './LogicalBackupStatus';
import { RestoreVerificationStatus } from './RestoreVerificationStatus';

export interface LogicalBackup {
  id: string;
  database: Database;
  storage: Storage;
  status: LogicalBackupStatus;
  failMessage?: string;
  backupSizeMb: number;
  backupRawDbSizeMb: number;
  backupDurationMs: number;
  encryption: BackupEncryption;
  restoreVerificationStatus?: RestoreVerificationStatus;
  createdAt: Date;
}
