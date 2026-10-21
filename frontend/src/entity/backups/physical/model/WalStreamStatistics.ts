import type { WalStreamCurrent } from './WalStreamCurrent';
import type { WalStreamHealth } from './WalStreamHealth';
import type { WalStreamKeepingUp } from './WalStreamKeepingUp';
import type { WalStreamSeries } from './WalStreamSeries';
import type { WalStreamStreamer } from './WalStreamStreamer';
import type { WalStreamTimeRange } from './WalStreamTimeRange';

export interface WalStreamStatistics {
  health: WalStreamHealth;

  // The moment a restore to the latest available point reaches; null while no full backup has
  // completed.
  latestRestorableAt: string | null;

  // The stretches of time the newest chain can restore to, oldest first; the last one ends at
  // latestRestorableAt. Empty while no full backup has completed.
  restorableRanges: WalStreamTimeRange[];

  streamer: WalStreamStreamer;
  current: WalStreamCurrent | null;
  keepingUp: WalStreamKeepingUp;
  series: WalStreamSeries;
}
