import type { WalStreamSourceState } from './WalStreamSourceState';

// Source-side values are null while the source is unreachable or without the slot. On a reachable
// sample a null slotSafeWalSizeBytes means the source sets no limit.
export interface WalStreamCurrent {
  sampledAt: string;
  lastArchivedWalAt: string | null;
  sessionStartedAt: string;

  sourceState: WalStreamSourceState;
  sourceUnreachableSince: string | null;

  sourceCurrentLsn: string | null;
  archiveProgressLsn: string | null;
  sourceWrittenBytes: number | null;

  archivedSegments: number;
  archivedWalBytes: number;
  archivedStoredBytes: number;
  walSegmentSizeBytes: number;
  highestConcurrentUploads: number;
  segmentWriteAvgMillis: number | null;
  segmentWriteMaxMillis: number | null;

  slotWalStatus: string | null;
  slotExtraWalSince: string | null;
  slotLagBytes: number | null;
  slotSafeWalSizeBytes: number | null;

  unarchivedWalBytes: number | null;
  archiveStalledSince: string | null;

  localQueueBytes: number;
  localQueuePauseBytes: number;

  receiverRestarts: number;
  receiverPauses: number;
  isReceiverRunning: boolean;
  isForcedRotationDenied: boolean;
}
