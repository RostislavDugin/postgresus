import type { WalStreamLive } from './WalStreamLive';

// Current is null while no streamer runs for the database.
export interface WalStreamLiveResponse {
  isStreamerRunning: boolean;
  current: WalStreamLive | null;
}
