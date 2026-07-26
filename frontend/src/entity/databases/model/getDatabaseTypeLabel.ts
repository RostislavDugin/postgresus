import { DatabaseType } from './DatabaseType';

export const getDatabaseTypeLabel = (type: DatabaseType): string => {
  switch (type) {
    case DatabaseType.POSTGRES_LOGICAL:
    case DatabaseType.POSTGRES_PHYSICAL:
      return 'PostgreSQL';
    case DatabaseType.MYSQL:
      return 'MySQL';
    case DatabaseType.MARIADB:
      return 'MariaDB';
    case DatabaseType.MONGODB:
      return 'MongoDB';
    default:
      return 'database';
  }
};
