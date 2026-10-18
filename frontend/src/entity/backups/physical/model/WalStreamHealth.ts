import type { WalStreamHealthLevel } from './WalStreamHealthLevel';
import type { WalStreamNotice } from './WalStreamNotice';
import type { WalStreamReason } from './WalStreamReason';

export interface WalStreamHealth {
  level: WalStreamHealthLevel;
  reasons: WalStreamReason[];
  notices: WalStreamNotice[];
  isFullBackupPending: boolean;
}
