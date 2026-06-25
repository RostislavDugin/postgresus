import { type Database, PostgresSslMode, PostgresqlVersion } from '../../../../entity/databases';

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

const sslModeLabels: Record<string, string> = {
  [PostgresSslMode.Disable]: 'Disable',
  [PostgresSslMode.Require]: 'Require',
  [PostgresSslMode.VerifyCa]: 'Verify CA',
  [PostgresSslMode.VerifyFull]: 'Verify full',
};

export const ShowPostgreSqlLogicalSpecificDataComponent = ({ database }: Props) => {
  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">PG version</div>
        <div>
          {database.postgresqlLogical?.version
            ? postgresqlVersionLabels[database.postgresqlLogical.version]
            : ''}
        </div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] break-all">Host</div>
        <div>{database.postgresqlLogical?.host || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Port</div>
        <div>{database.postgresqlLogical?.port || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Username</div>
        <div>{database.postgresqlLogical?.username || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Password</div>
        <div>{'*************'}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">DB name</div>
        <div>{database.postgresqlLogical?.database || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">SSL mode</div>
        <div>{sslModeLabels[database.postgresqlLogical?.sslMode ?? PostgresSslMode.Disable]}</div>
      </div>

      {!!database.postgresqlLogical?.sslClientCert &&
        database.postgresqlLogical?.sslMode !== PostgresSslMode.Disable && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Client certificate</div>
            <div>*************</div>
          </div>
        )}

      {!!database.postgresqlLogical?.includeSchemas?.length && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Include schemas</div>
          <div>{database.postgresqlLogical.includeSchemas.join(', ')}</div>
        </div>
      )}

      {!!database.postgresqlLogical?.excludeTables?.length && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Exclude tables</div>
          <div>{database.postgresqlLogical.excludeTables.join(', ')}</div>
        </div>
      )}

      {!!database.postgresqlLogical?.isSkipUserMappings && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Skip user mappings</div>
          <div>Yes</div>
        </div>
      )}
    </div>
  );
};
