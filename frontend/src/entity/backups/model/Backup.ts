import type { Database } from '../../databases/model/Database';
import type { Storage } from '../../storages';
import { BackupEncryption } from './BackupEncryption';
import { BackupStatus } from './BackupStatus';
import type { PgWalBackupType } from './PgWalBackupType';
import type { TableHealthReport } from './TableHealthReport';

export interface Backup {
  id: string;
  database: Database;
  storage: Storage;
  status: BackupStatus;
  failMessage?: string;
  backupSizeMb: number;
  backupDurationMs: number;
  encryption: BackupEncryption;
  pgWalBackupType?: PgWalBackupType;
  tableHealthReport?: TableHealthReport;
  createdAt: Date;
}
