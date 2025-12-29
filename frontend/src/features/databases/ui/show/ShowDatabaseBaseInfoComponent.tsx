import { type Database, getDatabaseLogoFromType } from '../../../../entity/databases';

interface Props {
  database: Database;
  isShowName?: boolean;
  isShowType?: boolean;
}

export const ShowDatabaseBaseInfoComponent = ({ database, isShowName, isShowType }: Props) => {
  return (
    <div>
      {isShowName && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Name</div>
          <div>{database.name || ''}</div>
        </div>
      )}

      {isShowType && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Database type</div>
          <div className="flex items-center">
            <span>{database.type === 'POSTGRES' ? 'PostgreSQL' : 'MySQL'}</span>
            <img
              src={getDatabaseLogoFromType(database.type)}
              alt="databaseIcon"
              className="ml-2 h-4 w-4"
            />
          </div>
        </div>
      )}
    </div>
  );
};
