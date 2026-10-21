import type { WalStreamSourceState } from './WalStreamSourceState';

// The running streamer's values at one moment, never stored. The source-side values come from the
// streamer's newest probe of the source, made at least every 10 seconds; they are null until that
// probe found the source reachable with its slot.
export interface WalStreamLive {
  measuredAt: string;

  sourceProbedAt: string | null;
  sourceState: WalStreamSourceState | null;
  sourceCurrentLsn: string | null;
  slotWalStatus: string | null;
  slotLagBytes: number | null;
  slotSafeWalSizeBytes: number | null;

  archiveProgressLsn: string | null;
  lastArchivedWalAt: string | null;
  unarchivedWalBytes: number | null;

  localQueueBytes: number;
  localQueuePauseBytes: number;
  isReceiverRunning: boolean;
}
