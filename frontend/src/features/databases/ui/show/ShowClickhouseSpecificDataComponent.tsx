import { ClickhouseVersion, type Database } from '../../../../entity/databases';

interface Props {
  database: Database;
}

const clickhouseVersionLabels = {
  [ClickhouseVersion.ClickhouseVersion238]: '23.8 LTS',
  [ClickhouseVersion.ClickhouseVersion244]: '24.4',
  [ClickhouseVersion.ClickhouseVersion248]: '24.8 LTS',
  [ClickhouseVersion.ClickhouseVersion254]: '25.4',
};

export const ShowClickhouseSpecificDataComponent = ({ database }: Props) => {
  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">ClickHouse version</div>
        <div>
          {database.clickhouse?.version ? clickhouseVersionLabels[database.clickhouse.version] : ''}
        </div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] break-all">Host</div>
        <div>{database.clickhouse?.host || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Port</div>
        <div>{database.clickhouse?.port || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Username</div>
        <div>{database.clickhouse?.username || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Password</div>
        <div>{'*************'}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">DB name</div>
        <div>{database.clickhouse?.database || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Use TLS</div>
        <div>{database.clickhouse?.isHttps ? 'Yes' : 'No'}</div>
      </div>
    </div>
  );
};
