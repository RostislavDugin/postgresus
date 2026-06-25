import { type Database, DatabaseType } from '../../../../entity/databases';
import { ShowMariaDbSpecificDataComponent } from './ShowMariaDbSpecificDataComponent';
import { ShowMongoDbSpecificDataComponent } from './ShowMongoDbSpecificDataComponent';
import { ShowMySqlSpecificDataComponent } from './ShowMySqlSpecificDataComponent';
import { ShowPostgreSqlLogicalSpecificDataComponent } from './ShowPostgreSqlLogicalSpecificDataComponent';
import { ShowPostgreSqlPhysicalSpecificDataComponent } from './ShowPostgreSqlPhysicalSpecificDataComponent';

interface Props {
  database: Database;
}

export const ShowDatabaseSpecificDataComponent = ({ database }: Props) => {
  switch (database.type) {
    case DatabaseType.POSTGRES_LOGICAL:
      return <ShowPostgreSqlLogicalSpecificDataComponent database={database} />;
    case DatabaseType.POSTGRES_PHYSICAL:
      return <ShowPostgreSqlPhysicalSpecificDataComponent database={database} />;
    case DatabaseType.MYSQL:
      return <ShowMySqlSpecificDataComponent database={database} />;
    case DatabaseType.MARIADB:
      return <ShowMariaDbSpecificDataComponent database={database} />;
    case DatabaseType.MONGODB:
      return <ShowMongoDbSpecificDataComponent database={database} />;
    default:
      return null;
  }
};
