import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type {
  MariadbDatabase,
  MongodbDatabase,
  MysqlDatabase,
  PostgresqlDatabase,
} from '../../databases';
import type { Restore } from '../model/Restore';

export const restoreApi = {
  async getRestores(backupId: string) {
    return apiHelper.fetchGetJson<Restore[]>(
      `${getApplicationServer()}/api/v1/restores/${backupId}`,
      undefined,
      true,
    );
  },

  async restoreBackup({
    backupId,
    postgresql,
    mysql,
    mariadb,
    mongodb,
  }: {
    backupId: string;
    postgresql?: PostgresqlDatabase;
    mysql?: MysqlDatabase;
    mariadb?: MariadbDatabase;
    mongodb?: MongodbDatabase;
  }) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(
      JSON.stringify({
        postgresqlDatabase: postgresql,
        mysqlDatabase: mysql,
        mariadbDatabase: mariadb,
        mongodbDatabase: mongodb,
      }),
    );

    return apiHelper.fetchPostJson<{ message: string }>(
      `${getApplicationServer()}/api/v1/restores/${backupId}/restore`,
      requestOptions,
    );
  },
};
