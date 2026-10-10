import dayjs from 'dayjs';

import type { ScheduledBackupTime } from './ScheduledBackupTime';

export const isScheduledBackupDue = (
  scheduledBackupTime: ScheduledBackupTime,
  now: Date = new Date(),
): boolean => scheduledBackupTime.isDue || !dayjs(scheduledBackupTime.scheduledAt).isAfter(now);
