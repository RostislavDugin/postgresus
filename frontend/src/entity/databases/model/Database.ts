import type { Notifier } from '../../notifiers';
import type { DatabaseType } from './DatabaseType';
import type { HealthStatus } from './HealthStatus';
import type { MariadbDatabase } from './mariadb/MariadbDatabase';
import type { MongodbDatabase } from './mongodb/MongodbDatabase';
import type { MysqlDatabase } from './mysql/MysqlDatabase';
import type { PostgresqlDatabase } from './postgresql/PostgresqlDatabase';

export interface Database {
  id: string;
  name: string;
  workspaceId: string;
  type: DatabaseType;

  postgresql?: PostgresqlDatabase;
  mysql?: MysqlDatabase;
  mariadb?: MariadbDatabase;
  mongodb?: MongodbDatabase;

  notifiers: Notifier[];

  lastBackupTime?: Date;
  lastBackupErrorMessage?: string;

  healthStatus?: HealthStatus;
}
