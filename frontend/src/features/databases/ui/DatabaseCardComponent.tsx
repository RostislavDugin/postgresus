import { InfoCircleOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';

import { logicalBackupConfigApi } from '../../../entity/backups/logical';
import { physicalBackupConfigApi } from '../../../entity/backups/physical';
import { type Database, DatabaseType } from '../../../entity/databases';
import { HealthStatus } from '../../../entity/databases/model/HealthStatus';
import type { Storage } from '../../../entity/storages';
import { getStorageLogoFromType } from '../../../entity/storages/models/getStorageLogoFromType';

interface Props {
  database: Database;
  selectedDatabaseId?: string;
  setSelectedDatabaseId: (databaseId: string) => void;
}

export const DatabaseCardComponent = ({
  database,
  selectedDatabaseId,
  setSelectedDatabaseId,
}: Props) => {
  const [storage, setStorage] = useState<Storage | undefined>();

  useEffect(() => {
    if (!database.id) return;

    const backupConfigRequest =
      database.type === DatabaseType.POSTGRES_PHYSICAL
        ? physicalBackupConfigApi.getPhysicalBackupConfigByDbId(database.id)
        : logicalBackupConfigApi.getBackupConfigByDbID(database.id);

    backupConfigRequest.then((config) => setStorage(config?.storage));
  }, [database.id, database.type]);

  return (
    <div
      className={`mb-3 cursor-pointer rounded p-3 shadow ${selectedDatabaseId === database.id ? 'bg-blue-100 dark:bg-blue-800' : 'bg-white dark:bg-gray-800'}`}
      onClick={() => setSelectedDatabaseId(database.id)}
    >
      <div className="flex">
        <div className="mb-1 min-w-0 font-bold break-words">{database.name}</div>

        {database.healthStatus && (
          <div className="ml-auto shrink-0 pl-1">
            <div
              className={`rounded px-[6px] py-[2px] text-[10px] text-white ${
                database.healthStatus === HealthStatus.AVAILABLE ? 'bg-green-500' : 'bg-red-500'
              }`}
            >
              {database.healthStatus === HealthStatus.AVAILABLE ? 'Available' : 'Unavailable'}
            </div>
          </div>
        )}
      </div>

      {storage && (
        <div className="text-sm text-gray-500 dark:text-gray-400">
          <span>Storage: </span>
          <span className="inline-flex items-center">
            {storage.name}{' '}
            {storage.type && (
              <img
                src={getStorageLogoFromType(storage.type)}
                alt="storageIcon"
                className="ml-1 h-4 w-4"
              />
            )}
          </span>
        </div>
      )}

      {database.lastBackupTime && (
        <div className="text-gray-500 dark:text-gray-400">
          Last backup {dayjs(database.lastBackupTime).fromNow()}
        </div>
      )}

      {database.lastBackupErrorMessage && (
        <div className="mt-1 flex items-center text-sm text-red-600 underline dark:text-red-400">
          <InfoCircleOutlined className="mr-1" style={{ color: 'red' }} />
          Has backup error
        </div>
      )}
    </div>
  );
};
