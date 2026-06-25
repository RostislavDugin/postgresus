import type { PhysicalFullBackupsPolicy } from './PhysicalFullBackupsPolicy';

export interface FullBackupsRetention {
  policy: PhysicalFullBackupsPolicy;
  count: number;

  gfsHours: number;
  gfsDays: number;
  gfsWeeks: number;
  gfsMonths: number;
  gfsYears: number;
}
