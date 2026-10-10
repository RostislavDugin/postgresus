import { describe, expect, it } from 'vitest';

import { isScheduledBackupDue } from './isScheduledBackupDue';

const now = new Date('2026-01-15T12:00:00Z');

describe('isScheduledBackupDue', () => {
  it('is due when the server reports it as due, even with a future time', () => {
    expect(
      isScheduledBackupDue({ scheduledAt: new Date('2026-01-15T13:00:00Z'), isDue: true }, now),
    ).toBe(true);
  });

  it('is not due when the scheduled time is still ahead', () => {
    expect(
      isScheduledBackupDue({ scheduledAt: new Date('2026-01-15T12:00:01Z'), isDue: false }, now),
    ).toBe(false);
  });

  it('is due when the scheduled time is exactly now', () => {
    expect(isScheduledBackupDue({ scheduledAt: now, isDue: false }, now)).toBe(true);
  });

  it('is due when the scheduled time has passed', () => {
    expect(
      isScheduledBackupDue({ scheduledAt: new Date('2026-01-15T11:59:59Z'), isDue: false }, now),
    ).toBe(true);
  });

  it('accepts the ISO string the API actually returns', () => {
    const scheduledAt = '2026-01-15T11:00:00Z' as unknown as Date;

    expect(isScheduledBackupDue({ scheduledAt, isDue: false }, now)).toBe(true);
  });
});
