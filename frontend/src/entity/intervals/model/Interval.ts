import type { IntervalType } from './IntervalType';

export interface Interval {
  id: string;
  interval: IntervalType;
  timeOfDay: string;
  // only for WEEKLY
  weekday?: number;
  // only for MONTHLY
  dayOfMonth?: number;
  // only for CRON
  cronExpression?: string;
}
