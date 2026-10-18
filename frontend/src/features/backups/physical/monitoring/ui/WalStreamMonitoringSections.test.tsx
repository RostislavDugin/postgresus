import type { JSX } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import {
  WalStreamHealthLevel,
  WalStreamKeepingUpState,
  WalStreamNoticeCode,
  WalStreamReasonCode,
  WalStreamReasonLevel,
  WalStreamSourceState,
  type WalStreamStatistics,
  WalStreamStatisticsPeriod,
  WalStreamStreamerState,
} from '../../../../../entity/backups/physical';
import { LocaleProvider } from '../../../../../shared/i18n';
import { getKeepingUpSentences } from '../lib/keepingUpSentences';
import { getStageTones } from '../lib/pathStages';
import { getNoticeTexts, getReasonTexts } from '../lib/reasonTexts';
import { getVerdictBanner } from '../lib/verdictBanner';
import { ChangesPathComponent } from './ChangesPathComponent';
import { MonitoringReasonsComponent } from './MonitoringReasonsComponent';
import { VerdictBannerComponent } from './VerdictBannerComponent';
import { WalStreamDetailsComponent } from './WalStreamDetailsComponent';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window. LocaleProvider then reads
// localStorage, which here reports no saved language.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
  vi.stubGlobal('localStorage', { getItem: () => null });
  vi.stubGlobal('navigator', { language: 'en-US', languages: ['en-US'] });
});

const MiB = 1024 * 1024;
const now = new Date('2026-10-18T12:00:00Z');

const statistics: WalStreamStatistics = {
  health: {
    level: WalStreamHealthLevel.NEEDS_ATTENTION,
    reasons: [
      {
        code: WalStreamReasonCode.LOCAL_QUEUE_FILLING,
        level: WalStreamReasonLevel.NEEDS_ATTENTION,
        valueBytes: 600 * MiB,
        thresholdBytes: 512 * MiB,
      },
    ],
    notices: [{ code: WalStreamNoticeCode.NO_SLOT_WAL_LIMIT, level: WalStreamReasonLevel.INFO }],
    isFullBackupPending: false,
  },
  streamer: {
    state: WalStreamStreamerState.RUNNING,
    sessionStartedAt: '2026-10-18T10:00:00Z',
    failingSince: null,
    restartsLastHour: 1,
  },
  current: {
    sampledAt: '2026-10-18T11:59:48Z',
    lastArchivedWalAt: '2026-10-18T11:59:20Z',
    sessionStartedAt: '2026-10-18T10:00:00Z',
    sourceState: WalStreamSourceState.REACHABLE,
    sourceUnreachableSince: null,
    sourceCurrentLsn: '0/5000000',
    archiveProgressLsn: '0/4F00000',
    sourceWrittenBytes: 16 * MiB,
    archivedSegments: 2,
    archivedWalBytes: 32 * MiB,
    archivedStoredBytes: 8 * MiB,
    walSegmentSizeBytes: 16 * MiB,
    highestConcurrentUploads: 3,
    segmentWriteAvgMillis: 120,
    segmentWriteMaxMillis: 300,
    slotWalStatus: 'reserved',
    slotExtraWalSince: null,
    slotLagBytes: 2 * MiB,
    slotSafeWalSizeBytes: null,
    unarchivedWalBytes: 24 * MiB,
    archiveStalledSince: null,
    localQueueBytes: 600 * MiB,
    localQueuePauseBytes: 512 * MiB,
    receiverRestarts: 0,
    receiverPauses: 0,
    isReceiverRunning: false,
    isForcedRotationDenied: false,
  },
  keepingUp: {
    state: WalStreamKeepingUpState.KEEPING_UP,
    sourceWriteBytesPerSecond: 10 * MiB,
    archiveBytesPerSecond: 10 * MiB,
    unarchivedWalBytes: 24 * MiB,
    unarchivedWalChangeBytesPerSecond: 0,
    slotLagBytes: 2 * MiB,
    slotLagChangeBytesPerSecond: 0,
    slotRebuildLagBytes: 256 * MiB,
    estimatedCatchUpSeconds: null,
    estimatedSlotRebuildSeconds: null,
    estimatedSlotLimitSeconds: null,
  },
  series: {
    period: WalStreamStatisticsPeriod.HOUR,
    bucketSeconds: 60,
    from: '2026-10-18T11:01:00Z',
    to: '2026-10-18T12:01:00Z',
    points: [],
  },
};

const renderWithLocale = (element: JSX.Element) =>
  renderToStaticMarkup(<LocaleProvider>{element}</LocaleProvider>);

describe('the Monitoring tab sections', () => {
  it('leads with the verdict, the keeping-up sentence and the age of the newest sample', () => {
    const markup = renderWithLocale(
      <VerdictBannerComponent
        banner={getVerdictBanner(statistics.health.level, now, now)}
        keepingUpSentences={getKeepingUpSentences(statistics.keepingUp)}
        sampledAt={statistics.current?.sampledAt ?? null}
        now={now}
        isReducedMotion
        onOpenConfig={() => {}}
      />,
    );

    expect(markup).toContain('Something needs attention');
    expect(markup).toContain('aria-live="polite"');
    expect(markup).toContain(
      'Databasus saves changes as fast as the database writes them: 10 MB/s written',
    );
    expect(markup).toContain('Measured 12 s ago');
  });

  it('lists each reason with what happened and what to do, and names the setting in the notice', () => {
    const markup = renderWithLocale(
      <MonitoringReasonsComponent
        reasons={statistics.health.reasons.map((reason) => getReasonTexts(reason, false, now))}
        notices={statistics.health.notices.map((notice) => ({
          code: notice.code,
          texts: getNoticeTexts(notice),
        }))}
      />,
    );

    expect(markup).toContain('The Databasus buffer is filling up');
    expect(markup).toContain('The Databasus buffer holds 600 MB of 512 MB.');
    expect(markup).toContain('No limit on kept changes');
  });

  it('shows an amount above its limit as over the limit and the receiver as paused', () => {
    const markup = renderWithLocale(
      <ChangesPathComponent
        statistics={statistics}
        stageTones={getStageTones(statistics.health.level, statistics.health.reasons)}
        isReducedMotion
      />,
    );

    expect(markup).toContain('Over the limit');
    expect(markup).toContain('Receiving paused');
    expect(markup).toContain('Receiving pauses at 512 MB');
    expect(markup).toContain('No limit on kept changes');
  });

  it('keeps the technical names in details that start collapsed', () => {
    const markup = renderWithLocale(<WalStreamDetailsComponent statistics={statistics} />);

    expect(markup).toContain('Details');
    expect(markup).not.toContain('wal_status');
  });
});
