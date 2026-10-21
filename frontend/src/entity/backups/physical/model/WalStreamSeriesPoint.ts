// Every value is null when no sample falls into the point's interval.
export interface WalStreamSeriesPoint {
  startedAt: string;

  sourceWriteBytesPerSecond: number | null;
  archiveBytesPerSecond: number | null;
  archivedStoredBytesPerSecond: number | null;

  maxSlotLagBytes: number | null;
  maxUnarchivedWalBytes: number | null;
  maxLocalQueueBytes: number | null;
  localQueuePauseBytes: number | null;

  restarts: number | null;
  pauses: number | null;
  sessionStarts: number | null;

  highestConcurrentUploads: number | null;
  segmentWriteAvgMillis: number | null;
  segmentWriteMaxMillis: number | null;
}
