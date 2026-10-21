import {
  type Database,
  DatabaseType,
  PhysicalDatabaseBackupType,
} from '../../../../../entity/databases';

export const isMonitoringTabShown = (database: Database): boolean =>
  database.type === DatabaseType.POSTGRES_PHYSICAL &&
  database.postgresqlPhysical?.backupType ===
    PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM;
