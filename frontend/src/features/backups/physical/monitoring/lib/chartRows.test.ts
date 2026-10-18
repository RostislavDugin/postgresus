import { describe, expect, it, vi } from 'vitest';

import {
  type WalStreamSeries,
  type WalStreamSeriesPoint,
  WalStreamStatisticsPeriod,
} from '../../../../../entity/backups/physical';
import { getDataSinceMs, hasAnyChartValues, toChartRows } from './chartRows';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const MB = 1000 * 1000;
const seriesStartMs = Date.parse('2026-10-18T11:00:00Z');

const buildEmptyPoint = (minute: number): WalStreamSeriesPoint => ({
  startedAt: new Date(seriesStartMs + minute * 60_000).toISOString(),
  sourceWriteBytesPerSecond: null,
  archiveBytesPerSecond: null,
  archivedStoredBytesPerSecond: null,
  maxSlotLagBytes: null,
  maxUnarchivedWalBytes: null,
  maxLocalQueueBytes: null,
  localQueuePauseBytes: null,
  restarts: null,
  pauses: null,
  sessionStarts: null,
  highestConcurrentUploads: null,
  segmentWriteAvgMillis: null,
  segmentWriteMaxMillis: null,
});

const buildFallingBehindPoint = (minute: number): WalStreamSeriesPoint => ({
  ...buildEmptyPoint(minute),
  sourceWriteBytesPerSecond: 21 * MB,
  archiveBytesPerSecond: 12 * MB,
  maxUnarchivedWalBytes: minute * 9 * MB * 60,
  maxLocalQueueBytes: 64 * MB,
  localQueuePauseBytes: 512 * MB,
  restarts: 0,
  pauses: 0,
});

const buildSeries = (points: WalStreamSeriesPoint[]): WalStreamSeries => ({
  period: WalStreamStatisticsPeriod.HOUR,
  bucketSeconds: 60,
  from: points[0].startedAt,
  to: new Date(seriesStartMs + points.length * 60_000).toISOString(),
  points,
});

describe('toChartRows', () => {
  it('keeps archiving below the source and the backlog rising while archiving falls behind', () => {
    const rows = toChartRows(
      Array.from({ length: 60 }, (_, minute) => buildFallingBehindPoint(minute)),
    );

    for (const row of rows) {
      expect(row.archiveBytesPerSecond ?? 0).toBeLessThan(row.sourceWriteBytesPerSecond ?? 0);
    }

    for (let rowIndex = 1; rowIndex < rows.length; rowIndex++) {
      expect(rows[rowIndex].maxUnarchivedWalBytes ?? 0).toBeGreaterThan(
        rows[rowIndex - 1].maxUnarchivedWalBytes ?? 0,
      );
    }
  });

  it('keeps an interval without samples as a gap, not as zero', () => {
    const [row] = toChartRows([buildEmptyPoint(0)]);

    expect(row.sourceWriteBytesPerSecond).toBeNull();
    expect(row.archiveBytesPerSecond).toBeNull();
    expect(row.maxUnarchivedWalBytes).toBeNull();
    expect(row.maxLocalQueueBytes).toBeNull();
    expect(row.restartMarkerBytes).toBeNull();
  });

  it('puts restart and pause markers on the buffer line', () => {
    const [row] = toChartRows([{ ...buildFallingBehindPoint(0), restarts: 1, pauses: 2 }]);

    expect(row.restartMarkerBytes).toBe(64 * MB);
    expect(row.pauseMarkerBytes).toBe(64 * MB);
  });
});

describe('hasAnyChartValues', () => {
  it('reports no history when no point has values', () => {
    expect(hasAnyChartValues(buildSeries([buildEmptyPoint(0), buildEmptyPoint(1)]))).toBe(false);
  });

  it('reports history once one point has values', () => {
    expect(hasAnyChartValues(buildSeries([buildEmptyPoint(0), buildFallingBehindPoint(1)]))).toBe(
      true,
    );
  });
});

describe('getDataSinceMs', () => {
  it('names the first interval with data when the data starts inside the period', () => {
    const series = buildSeries([
      buildEmptyPoint(0),
      buildEmptyPoint(1),
      buildFallingBehindPoint(2),
    ]);

    expect(getDataSinceMs(series)).toBe(seriesStartMs + 2 * 60_000);
  });

  it('says nothing when the data covers the whole period', () => {
    expect(
      getDataSinceMs(buildSeries([buildFallingBehindPoint(0), buildFallingBehindPoint(1)])),
    ).toBeNull();
  });
});
