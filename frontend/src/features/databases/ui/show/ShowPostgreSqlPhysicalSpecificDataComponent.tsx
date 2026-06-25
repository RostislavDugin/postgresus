import {
  type Database,
  PhysicalDatabaseBackupType,
  PostgresSslMode,
  PostgresqlVersion,
} from '../../../../entity/databases';

interface Props {
  database: Database;
}

const postgresqlVersionLabels: Record<string, string> = {
  [PostgresqlVersion.PostgresqlVersion12]: '12',
  [PostgresqlVersion.PostgresqlVersion13]: '13',
  [PostgresqlVersion.PostgresqlVersion14]: '14',
  [PostgresqlVersion.PostgresqlVersion15]: '15',
  [PostgresqlVersion.PostgresqlVersion16]: '16',
  [PostgresqlVersion.PostgresqlVersion17]: '17',
  [PostgresqlVersion.PostgresqlVersion18]: '18',
};

const backupTypeLabels: Record<string, string> = {
  [PhysicalDatabaseBackupType.FULL]: 'Full backups only',
  [PhysicalDatabaseBackupType.FULL_INCREMENTAL]: 'Full + incremental',
  [PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM]: 'Full + incremental + WAL streaming',
};

const sslModeLabels: Record<string, string> = {
  [PostgresSslMode.Disable]: 'Disable',
  [PostgresSslMode.Require]: 'Require',
  [PostgresSslMode.VerifyCa]: 'Verify CA',
  [PostgresSslMode.VerifyFull]: 'Verify full',
};

export const ShowPostgreSqlPhysicalSpecificDataComponent = ({ database }: Props) => {
  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">PG version</div>
        <div>
          {database.postgresqlPhysical?.version
            ? postgresqlVersionLabels[database.postgresqlPhysical.version]
            : ''}
        </div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Backup type</div>
        <div>
          {database.postgresqlPhysical?.backupType
            ? backupTypeLabels[database.postgresqlPhysical.backupType]
            : ''}
        </div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] break-all">Host</div>
        <div>{database.postgresqlPhysical?.host || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Port</div>
        <div>{database.postgresqlPhysical?.port || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Username</div>
        <div>{database.postgresqlPhysical?.username || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Password</div>
        <div>{'*************'}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">SSL mode</div>
        <div>{sslModeLabels[database.postgresqlPhysical?.sslMode ?? PostgresSslMode.Disable]}</div>
      </div>

      {!!database.postgresqlPhysical?.sslClientCert &&
        database.postgresqlPhysical?.sslMode !== PostgresSslMode.Disable && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Client certificate</div>
            <div>*************</div>
          </div>
        )}
    </div>
  );
};
