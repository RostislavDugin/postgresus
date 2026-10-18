import { useTranslation } from 'react-i18next';

import {
  type Database,
  POSTGRESQL_SHORT_NAME,
  POSTGRES_SSL_MODE_LABEL_KEYS,
  PostgresSslMode,
  PostgresqlVersion,
} from '../../../../entity/databases';
import { ShowSshTunnelComponent } from './ShowSshTunnelComponent';

interface Props {
  database: Database;
}

const postgresqlVersionLabels = {
  [PostgresqlVersion.PostgresqlVersion12]: '12',
  [PostgresqlVersion.PostgresqlVersion13]: '13',
  [PostgresqlVersion.PostgresqlVersion14]: '14',
  [PostgresqlVersion.PostgresqlVersion15]: '15',
  [PostgresqlVersion.PostgresqlVersion16]: '16',
  [PostgresqlVersion.PostgresqlVersion17]: '17',
  [PostgresqlVersion.PostgresqlVersion18]: '18',
};

export const ShowPostgreSqlLogicalSpecificDataComponent = ({ database }: Props) => {
  const { t } = useTranslation();

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">
          {t('databases.fields.version', { engine: POSTGRESQL_SHORT_NAME })}
        </div>
        <div>
          {database.postgresqlLogical?.version
            ? postgresqlVersionLabels[database.postgresqlLogical.version]
            : ''}
        </div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2 break-all">{t('common.fields.host')}</div>
        <div>{database.postgresqlLogical?.host || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('common.fields.port')}</div>
        <div>{database.postgresqlLogical?.port || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('common.fields.username')}</div>
        <div>{database.postgresqlLogical?.username || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('common.fields.password')}</div>
        <div>{'*************'}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.fields.databaseName')}</div>
        <div>{database.postgresqlLogical?.database || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.fields.sslMode')}</div>
        <div>
          {t(
            POSTGRES_SSL_MODE_LABEL_KEYS[
              database.postgresqlLogical?.sslMode ?? PostgresSslMode.Disable
            ],
          )}
        </div>
      </div>

      {!!database.postgresqlLogical?.sslClientCert &&
        database.postgresqlLogical?.sslMode !== PostgresSslMode.Disable && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.fields.clientCertificate')}</div>
            <div>*************</div>
          </div>
        )}
      <ShowSshTunnelComponent sshTunnel={database.postgresqlLogical?.sshTunnel} />

      {!!database.postgresqlLogical?.includeSchemas?.length && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.includeSchemas')}</div>
          <div>{database.postgresqlLogical.includeSchemas.join(', ')}</div>
        </div>
      )}

      {!!database.postgresqlLogical?.excludeTables?.length && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.excludeWholeTables')}</div>
          <div>{database.postgresqlLogical.excludeTables.join(', ')}</div>
        </div>
      )}

      {!!database.postgresqlLogical?.excludeTableData?.length && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.excludeTableData')}</div>
          <div>{database.postgresqlLogical.excludeTableData.join(', ')}</div>
        </div>
      )}

      {!!database.postgresqlLogical?.isSkipUserMappings && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.skipUserMappings')}</div>
          <div>{t('common.answers.yes')}</div>
        </div>
      )}
    </div>
  );
};
