import { describe, expect, it, vi } from 'vitest';

import { WalStreamHealthLevel } from '../../../../../entity/backups/physical';
import { MonitoringTone } from '../model/MonitoringTone';
import { VerdictIcon } from '../model/VerdictIcon';
import { getVerdictBanner } from './verdictBanner';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const now = new Date('2026-10-18T12:00:00Z');

describe('getVerdictBanner', () => {
  it.each([
    [
      WalStreamHealthLevel.HEALTHY,
      MonitoringTone.SUCCESS,
      VerdictIcon.CHECK,
      'backups.physical.monitoring.levels.healthy',
    ],
    [
      WalStreamHealthLevel.NEEDS_ATTENTION,
      MonitoringTone.WARNING,
      VerdictIcon.EXCLAMATION,
      'backups.physical.monitoring.levels.needsAttention',
    ],
    [
      WalStreamHealthLevel.BROKEN,
      MonitoringTone.ERROR,
      VerdictIcon.CLOSE,
      'backups.physical.monitoring.levels.broken',
    ],
    [
      WalStreamHealthLevel.COLLECTING_DATA,
      MonitoringTone.NEUTRAL,
      VerdictIcon.CLOCK,
      'backups.physical.monitoring.levels.collectingData',
    ],
    [
      WalStreamHealthLevel.NOT_STREAMING,
      MonitoringTone.NEUTRAL,
      VerdictIcon.PAUSE,
      'backups.physical.monitoring.levels.notStreaming',
    ],
  ])('shows %s with its own color, icon and headline', (level, tone, icon, headlineKey) => {
    const banner = getVerdictBanner(level, now, now);

    expect(banner.tone).toBe(tone);
    expect(banner.icon).toBe(icon);
    expect(banner.headlineKey).toBe(headlineKey);
  });

  it('links the not streaming headline to the configuration', () => {
    expect(getVerdictBanner(WalStreamHealthLevel.NOT_STREAMING, now, now).isConfigLinkShown).toBe(
      true,
    );
    expect(getVerdictBanner(WalStreamHealthLevel.HEALTHY, now, now).isConfigLinkShown).toBe(false);
  });

  it('cannot confirm the state when refreshes have failed for 3 minutes after a healthy response', () => {
    const lastSuccessfulRefreshAt = new Date(now.getTime() - 3 * 60 * 1000);

    const banner = getVerdictBanner(WalStreamHealthLevel.HEALTHY, lastSuccessfulRefreshAt, now);

    expect(banner.isStateConfirmed).toBe(false);
    expect(banner.tone).toBe(MonitoringTone.NEUTRAL);
    expect(banner.icon).toBe(VerdictIcon.QUESTION);
    expect(banner.headlineKey).toBe('backups.physical.monitoring.banner.cannotConfirm');
    expect(banner.description).toEqual({
      key: 'backups.physical.monitoring.banner.cannotConfirmDescription',
      params: {
        level: { kind: 'text', text: { key: 'backups.physical.monitoring.levels.healthy' } },
        duration: {
          kind: 'text',
          text: { key: 'backups.physical.monitoring.durations.minutes', params: { minutes: 3 } },
        },
      },
    });
  });

  it('keeps the verdict while the last refresh is under 2 minutes old', () => {
    const lastSuccessfulRefreshAt = new Date(now.getTime() - 90 * 1000);

    expect(getVerdictBanner(WalStreamHealthLevel.HEALTHY, lastSuccessfulRefreshAt, now).tone).toBe(
      MonitoringTone.SUCCESS,
    );
  });
});
