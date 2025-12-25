import { type Database } from '../../../../entity/databases';

interface Props {
  database: Database;
}

export const ShowMongoDbSpecificDataComponent = ({ database }: Props) => {
  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] break-all">Host</div>
        <div>{database.mongodb?.host || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Port</div>
        <div>{database.mongodb?.port || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Username</div>
        <div>{database.mongodb?.username || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Password</div>
        <div>{'*************'}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">DB name</div>
        <div>{database.mongodb?.database || ''}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Use TLS</div>
        <div>{database.mongodb?.useTls ? 'Yes' : 'No'}</div>
      </div>

      {database.mongodb?.authDatabase && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Auth database</div>
          <div>{database.mongodb.authDatabase}</div>
        </div>
      )}
    </div>
  );
};
