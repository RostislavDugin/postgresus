import type { ScheduledBackupTime } from '../../shared';

export interface NextScheduledBackupResponse {
  // null when no automatic backup will run
  nextBackup: ScheduledBackupTime | null;
}
