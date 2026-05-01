import type { ClickhouseDatabase, PostgresqlDatabase } from '../../databases';
import { RestoreStatus } from './RestoreStatus';

export interface Restore {
  id: string;
  status: RestoreStatus;

  postgresql?: PostgresqlDatabase;
  clickhouse?: ClickhouseDatabase;

  failMessage?: string;

  restoreDurationMs: number;
  createdAt: string;
}
