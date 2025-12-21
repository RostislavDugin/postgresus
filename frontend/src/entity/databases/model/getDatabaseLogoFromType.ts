import { DatabaseType } from './DatabaseType';

export const getDatabaseLogoFromType = (type: DatabaseType) => {
  switch (type) {
    case DatabaseType.POSTGRES:
      return '/icons/databases/postgresql.svg';
    case DatabaseType.MYSQL:
      return '/icons/databases/mysql.svg';
    case DatabaseType.MARIADB:
      return '/icons/databases/mariadb.svg';
    default:
      return '';
  }
};
