import { type Database, DatabaseType } from '../../../../entity/databases';
import { ShowMySqlSpecificDataComponent } from './ShowMySqlSpecificDataComponent';
import { ShowPostgreSqlSpecificDataComponent } from './ShowPostgreSqlSpecificDataComponent';

interface Props {
  database: Database;
}

export const ShowDatabaseSpecificDataComponent = ({ database }: Props) => {
  if (database.type === DatabaseType.POSTGRES) {
    return <ShowPostgreSqlSpecificDataComponent database={database} />;
  }

  if (database.type === DatabaseType.MYSQL) {
    return <ShowMySqlSpecificDataComponent database={database} />;
  }

  return null;
};
