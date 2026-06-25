import type { Database } from './Database';
import { DatabaseType } from './DatabaseType';
import type { MariadbDatabase } from './mariadb/MariadbDatabase';
import type { MongodbDatabase } from './mongodb/MongodbDatabase';
import type { MysqlDatabase } from './mysql/MysqlDatabase';
import { PostgresSslMode } from './postgresql/PostgresSslMode';
import type { PostgresqlLogicalDatabase } from './postgresql/PostgresqlLogicalDatabase';
import { PhysicalDatabaseBackupType } from './postgresql/physical/PhysicalDatabaseBackupType';
import type { PostgresqlPhysicalDatabase } from './postgresql/physical/PostgresqlPhysicalDatabase';

// Resets every engine-specific sub-object, then seeds defaults for the active type.
export const initializeDatabaseTypeData = (database: Database): Database => {
  const base = {
    ...database,
    postgresqlLogical: undefined,
    postgresqlPhysical: undefined,
    mysql: undefined,
    mariadb: undefined,
    mongodb: undefined,
  };

  switch (database.type) {
    case DatabaseType.POSTGRES_LOGICAL:
      return {
        ...base,
        postgresqlLogical:
          database.postgresqlLogical ?? ({ cpuCount: 1 } as PostgresqlLogicalDatabase),
      };
    case DatabaseType.POSTGRES_PHYSICAL:
      return {
        ...base,
        postgresqlPhysical:
          database.postgresqlPhysical ??
          ({
            backupType: PhysicalDatabaseBackupType.FULL,
            sslMode: PostgresSslMode.Disable,
          } as PostgresqlPhysicalDatabase),
      };
    case DatabaseType.MYSQL:
      return { ...base, mysql: database.mysql ?? ({} as MysqlDatabase) };
    case DatabaseType.MARIADB:
      return { ...base, mariadb: database.mariadb ?? ({} as MariadbDatabase) };
    case DatabaseType.MONGODB:
      return { ...base, mongodb: database.mongodb ?? ({ cpuCount: 1 } as MongodbDatabase) };
    default:
      return base;
  }
};
