import type { WalStreamReasonCode } from './WalStreamReasonCode';
import type { WalStreamReasonLevel } from './WalStreamReasonLevel';

export interface WalStreamReason {
  code: WalStreamReasonCode;
  level: WalStreamReasonLevel;
  since?: string;
  valueBytes?: number;
  thresholdBytes?: number;
  count?: number;
  estimatedSeconds?: number;
}
