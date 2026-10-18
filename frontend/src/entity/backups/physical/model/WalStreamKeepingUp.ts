import type { WalStreamKeepingUpState } from './WalStreamKeepingUpState';

export interface WalStreamKeepingUp {
  state: WalStreamKeepingUpState;

  sourceWriteBytesPerSecond: number | null;
  archiveBytesPerSecond: number | null;

  unarchivedWalBytes: number | null;
  unarchivedWalChangeBytesPerSecond: number | null;

  slotLagBytes: number | null;
  slotLagChangeBytesPerSecond: number | null;

  slotRebuildLagBytes: number;

  estimatedCatchUpSeconds: number | null;
  estimatedSlotRebuildSeconds: number | null;
  estimatedSlotLimitSeconds: number | null;
}
