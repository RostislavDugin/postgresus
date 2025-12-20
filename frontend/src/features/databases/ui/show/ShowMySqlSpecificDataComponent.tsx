import { type Database, MysqlVersion } from '../../../../entity/databases';

interface Props {
  database: Database;
}

const mysqlVersionLabels = {
  [MysqlVersion.MysqlVersion57]: '5.7',
  [MysqlVersion.MysqlVersion80]: '8.0',
  [MysqlVersion.MysqlVersion84]: '8.4',
};

export const ShowMySqlSpecificDataComponent = ({ database }: Props) => {
  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">MySQL version</div>
        <div>{database.mysql?.version ? mysqlVersionLabels[database.mysql.version] : ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] break-all">Host</div>
        <div>{database.mysql?.host || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Port</div>
        <div>{database.mysql?.port || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Username</div>
        <div>{database.mysql?.username || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Password</div>
        <div>{'*************'}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">DB name</div>
        <div>{database.mysql?.database || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Use HTTPS</div>
        <div>{database.mysql?.isHttps ? 'Yes' : 'No'}</div>
      </div>
    </div>
  );
};
