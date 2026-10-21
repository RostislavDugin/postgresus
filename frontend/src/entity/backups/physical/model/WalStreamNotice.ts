import type { WalStreamNoticeCode } from './WalStreamNoticeCode';
import type { WalStreamReasonLevel } from './WalStreamReasonLevel';

export interface WalStreamNotice {
  code: WalStreamNoticeCode;
  level: WalStreamReasonLevel;
  since?: string;
}
