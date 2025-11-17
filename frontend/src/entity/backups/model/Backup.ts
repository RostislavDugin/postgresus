import type { Database } from '../../databases/model/Database';
import type { Storage } from '../../storages';
import { BackupEncryption } from './BackupEncryption';
import { BackupStatus } from './BackupStatus';

export interface Backup {
  id: string;

  database: Database;
  storage: Storage;

  status: BackupStatus;
  failMessage?: string;

  backupSizeMb: number;

  backupDurationMs: number;

  encryption: BackupEncryption;

  createdAt: Date;
}
