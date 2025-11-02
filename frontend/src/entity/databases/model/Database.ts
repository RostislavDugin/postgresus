import type { Notifier } from '../../notifiers';
import type { DatabaseType } from './DatabaseType';
import type { HealthStatus } from './HealthStatus';
import type { PostgresqlDatabase } from './postgresql/PostgresqlDatabase';

export interface Database {
  id: string;
  name: string;
  workspaceId: string;
  type: DatabaseType;

  postgresql?: PostgresqlDatabase;

  notifiers: Notifier[];

  lastBackupTime?: Date;
  lastBackupErrorMessage?: string;

  healthStatus?: HealthStatus;
}
