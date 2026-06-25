import type { PostgresqlLogicalDatabase } from '../../databases';
import { RestoreStatus } from './RestoreStatus';

export interface Restore {
  id: string;
  status: RestoreStatus;

  postgresql?: PostgresqlLogicalDatabase;

  failMessage?: string;

  restoreDurationMs: number;
  createdAt: string;
}
