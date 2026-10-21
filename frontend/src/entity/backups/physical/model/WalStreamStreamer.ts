import type { WalStreamStreamerState } from './WalStreamStreamerState';

export interface WalStreamStreamer {
  state: WalStreamStreamerState;
  sessionStartedAt: string | null;
  failingSince: string | null;
  restartsLastHour: number;
}
