import type { WalStreamCurrent } from './WalStreamCurrent';
import type { WalStreamHealth } from './WalStreamHealth';
import type { WalStreamKeepingUp } from './WalStreamKeepingUp';
import type { WalStreamSeries } from './WalStreamSeries';
import type { WalStreamStreamer } from './WalStreamStreamer';

export interface WalStreamStatistics {
  health: WalStreamHealth;
  streamer: WalStreamStreamer;
  current: WalStreamCurrent | null;
  keepingUp: WalStreamKeepingUp;
  series: WalStreamSeries;
}
