import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { MariadbDatabase, MysqlDatabase, PostgresqlDatabase } from '../../databases';
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
  }: {
    backupId: string;
    postgresql?: PostgresqlDatabase;
    mysql?: MysqlDatabase;
    mariadb?: MariadbDatabase;
  }) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(
      JSON.stringify({
        postgresqlDatabase: postgresql,
        mysqlDatabase: mysql,
        mariadbDatabase: mariadb,
      }),
    );

    return apiHelper.fetchPostJson<{ message: string }>(
      `${getApplicationServer()}/api/v1/restores/${backupId}/restore`,
      requestOptions,
    );
  },
};
