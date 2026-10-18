import { describe, expect, it } from 'vitest';

import { getMeasurementAge, getRoundedDuration } from './roundedDuration';

describe('getRoundedDuration', () => {
  it.each([
    [20, 'backups.physical.monitoring.durations.minutes', { minutes: 1 }],
    [40 * 60 + 20, 'backups.physical.monitoring.durations.minutes', { minutes: 40 }],
    [59 * 60 + 10, 'backups.physical.monitoring.durations.minutes', { minutes: 59 }],
    [59 * 60 + 40, 'backups.physical.monitoring.durations.hoursMinutes', { hours: 1, minutes: 0 }],
    [
      2 * 3600 + 15 * 60 + 10,
      'backups.physical.monitoring.durations.hoursMinutes',
      { hours: 2, minutes: 15 },
    ],
    [
      23 * 3600 + 59 * 60,
      'backups.physical.monitoring.durations.hoursMinutes',
      { hours: 23, minutes: 59 },
    ],
    [
      3 * 86400 + 4 * 3600 + 20 * 60,
      'backups.physical.monitoring.durations.daysHours',
      { days: 3, hours: 4 },
    ],
  ])('rounds %d seconds for reading', (seconds, expectedKey, expectedParams) => {
    expect(getRoundedDuration(seconds)).toEqual({ key: expectedKey, params: expectedParams });
  });
});

describe('getMeasurementAge', () => {
  it('counts seconds under a minute', () => {
    expect(getMeasurementAge(12.7)).toEqual({
      key: 'backups.physical.monitoring.durations.seconds',
      params: { seconds: 12 },
    });
  });

  it('rounds like any other duration from a minute on', () => {
    expect(getMeasurementAge(3 * 60)).toEqual(getRoundedDuration(3 * 60));
  });
});
