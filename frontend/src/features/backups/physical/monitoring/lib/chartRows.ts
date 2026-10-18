import type { WalStreamSeries, WalStreamSeriesPoint } from '../../../../../entity/backups/physical';
import type { WalStreamChartRow } from '../model/WalStreamChartRow';

export const toChartRows = (points: WalStreamSeriesPoint[]): WalStreamChartRow[] =>
  points.map((point) => ({
    timestampMs: new Date(point.startedAt).getTime(),
    sourceWriteBytesPerSecond: point.sourceWriteBytesPerSecond,
    archiveBytesPerSecond: point.archiveBytesPerSecond,
    maxUnarchivedWalBytes: point.maxUnarchivedWalBytes,
    maxSlotLagBytes: point.maxSlotLagBytes,
    maxLocalQueueBytes: point.maxLocalQueueBytes,
    localQueuePauseBytes: point.localQueuePauseBytes,
    restartMarkerBytes: getMarkerBytes(point.restarts, point),
    pauseMarkerBytes: getMarkerBytes(point.pauses, point),
  }));

// A marker sits on the buffer line at its interval, so it needs no axis of its own.
const getMarkerBytes = (eventCount: number | null, point: WalStreamSeriesPoint): number | null =>
  eventCount !== null && eventCount > 0 ? (point.maxLocalQueueBytes ?? 0) : null;

const hasPointValues = (point: WalStreamSeriesPoint): boolean =>
  point.archiveBytesPerSecond !== null ||
  point.sourceWriteBytesPerSecond !== null ||
  point.maxLocalQueueBytes !== null;

export const hasAnyChartValues = (series: WalStreamSeries): boolean =>
  series.points.some(hasPointValues);

// Null when the data covers the whole period, so the caption only appears when the charts start
// later than their axis.
export const getDataSinceMs = (series: WalStreamSeries): number | null => {
  const firstPointWithValues = series.points.find(hasPointValues);

  if (!firstPointWithValues || firstPointWithValues === series.points[0]) {
    return null;
  }

  return new Date(firstPointWithValues.startedAt).getTime();
};
