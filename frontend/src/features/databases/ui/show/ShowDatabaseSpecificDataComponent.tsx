import { type Database, DatabaseType } from '../../../../entity/databases';
import { ShowClickhouseSpecificDataComponent } from './ShowClickhouseSpecificDataComponent';
import { ShowMariaDbSpecificDataComponent } from './ShowMariaDbSpecificDataComponent';
import { ShowMongoDbSpecificDataComponent } from './ShowMongoDbSpecificDataComponent';
import { ShowMySqlSpecificDataComponent } from './ShowMySqlSpecificDataComponent';
import { ShowPostgreSqlSpecificDataComponent } from './ShowPostgreSqlSpecificDataComponent';

interface Props {
  database: Database;
}

export const ShowDatabaseSpecificDataComponent = ({ database }: Props) => {
  switch (database.type) {
    case DatabaseType.POSTGRES:
      return <ShowPostgreSqlSpecificDataComponent database={database} />;
    case DatabaseType.MYSQL:
      return <ShowMySqlSpecificDataComponent database={database} />;
    case DatabaseType.MARIADB:
      return <ShowMariaDbSpecificDataComponent database={database} />;
    case DatabaseType.MONGODB:
      return <ShowMongoDbSpecificDataComponent database={database} />;
    case DatabaseType.CLICKHOUSE:
      return <ShowClickhouseSpecificDataComponent database={database} />;
    default:
      return null;
  }
};
