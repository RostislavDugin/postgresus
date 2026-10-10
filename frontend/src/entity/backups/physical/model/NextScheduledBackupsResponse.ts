import type { ScheduledBackupTime } from '../../shared';

export interface NextScheduledBackupsResponse {
  // each is null when no such automatic backup will run
  nextFullBackup: ScheduledBackupTime | null;
  nextIncrementalBackup: ScheduledBackupTime | null;
}
