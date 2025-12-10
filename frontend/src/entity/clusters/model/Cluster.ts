import type { Notifier } from '../../notifiers';
import type { Interval } from '../../intervals';
import type { PostgresqlCluster } from './PostgresqlCluster';

export interface Cluster {
  id?: string;
  workspaceId: string;
  name: string;
  postgresql: PostgresqlCluster;

  isBackupsEnabled?: boolean;
  storePeriod?: string;
  backupIntervalId?: string;
  backupInterval?: Interval;
  storageId?: string;
  sendNotificationsOn?: string;
  cpuCount?: number;

  notifiers?: Notifier[];
  excludedDatabases?: { id?: string; clusterId?: string; name: string }[];
}
