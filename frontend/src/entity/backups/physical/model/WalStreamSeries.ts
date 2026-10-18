import type { WalStreamSeriesPoint } from './WalStreamSeriesPoint';
import type { WalStreamStatisticsPeriod } from './WalStreamStatisticsPeriod';

export interface WalStreamSeries {
  period: WalStreamStatisticsPeriod;
  bucketSeconds: number;
  from: string;
  to: string;
  points: WalStreamSeriesPoint[];
}
