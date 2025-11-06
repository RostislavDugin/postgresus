import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, DatabaseType, PostgresqlVersion } from '../../../../entity/databases';

interface Props {
  database: Database;
}

export const ShowDatabaseSpecificDataComponent = ({ database }: Props) => {
  const { t } = useTranslation(['database', 'common']);

  const databaseTypeLabels = useMemo(
    () => ({
      [DatabaseType.POSTGRES]: 'PostgreSQL',
    }),
    []
  );

  const postgresqlVersionLabels = useMemo(
    () => ({
      [PostgresqlVersion.PostgresqlVersion13]: '13',
      [PostgresqlVersion.PostgresqlVersion14]: '14',
      [PostgresqlVersion.PostgresqlVersion15]: '15',
      [PostgresqlVersion.PostgresqlVersion16]: '16',
      [PostgresqlVersion.PostgresqlVersion17]: '17',
      [PostgresqlVersion.PostgresqlVersion18]: '18',
    }),
    []
  );

  return (
    <div>
      <div className="mb-5 flex w-full items-center">
        <div className="min-w-[150px]">{t('database:form.type_label')}</div>
        <div>{database.type ? databaseTypeLabels[database.type] : ''}</div>
      </div>

      {database.type === DatabaseType.POSTGRES && (
        <>
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.version_label')}</div>
            <div>
              {database.postgresql?.version
                ? postgresqlVersionLabels[database.postgresql.version]
                : ''}
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] break-all">{t('database:form.host_label')}</div>
            <div>{database.postgresql?.host || ''}</div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.port_label')}</div>
            <div>{database.postgresql?.port || ''}</div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.username_label')}</div>
            <div>{database.postgresql?.username || ''}</div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.password_label')}</div>
            <div>{database.postgresql?.password ? '*********' : ''}</div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.database_label')}</div>
            <div>{database.postgresql?.database || ''}</div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.use_https_label')}</div>
            <div>{database.postgresql?.isHttps ? t('common:common.yes') : t('common:common.no')}</div>
          </div>
        </>
      )}
    </div>
  );
};
