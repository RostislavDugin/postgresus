import { Spin } from 'antd';
import { useRef, useState } from 'react';
import { useEffect } from 'react';

import { type Database, databaseApi } from '../../../entity/databases';
import { BackupsComponent } from '../../backups';
import { HealthckeckAttemptsComponent } from '../../healthcheck';
import { DatabaseConfigComponent } from './DatabaseConfigComponent';

interface Props {
  contentHeight: number;
  databaseId: string;
  onDatabaseChanged: (database: Database) => void;
  onDatabaseDeleted: () => void;
  isCanManageDBs: boolean;
}

export const DatabaseComponent = ({
  contentHeight,
  databaseId,
  onDatabaseChanged,
  onDatabaseDeleted,
  isCanManageDBs,
}: Props) => {
  const [currentTab, setCurrentTab] = useState<'config' | 'backups' | 'metrics'>('backups');

  const [database, setDatabase] = useState<Database | undefined>();
  const [editDatabase, setEditDatabase] = useState<Database | undefined>();

  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const loadSettings = () => {
    setDatabase(undefined);
    setEditDatabase(undefined);
    databaseApi.getDatabase(databaseId).then(setDatabase);
  };

  useEffect(() => {
    loadSettings();
  }, [databaseId]);

  if (!database) {
    return <Spin />;
  }

  return (
    <div
      className="w-full overflow-y-auto"
      style={{ maxHeight: contentHeight }}
      ref={scrollContainerRef}
    >
      <div className="flex">
        <div
          className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'config' ? 'bg-white' : 'bg-gray-200'}`}
          onClick={() => setCurrentTab('config')}
        >
          Config
        </div>

        <div
          className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'backups' ? 'bg-white' : 'bg-gray-200'}`}
          onClick={() => setCurrentTab('backups')}
        >
          Backups
        </div>
      </div>

      {currentTab === 'config' && (
        <DatabaseConfigComponent
          database={database}
          setDatabase={setDatabase}
          onDatabaseChanged={onDatabaseChanged}
          onDatabaseDeleted={onDatabaseDeleted}
          editDatabase={editDatabase}
          setEditDatabase={setEditDatabase}
          isCanManageDBs={isCanManageDBs}
        />
      )}
      {currentTab === 'backups' && (
        <>
          <HealthckeckAttemptsComponent database={database} />
          <BackupsComponent
            database={database}
            isCanManageDBs={isCanManageDBs}
            scrollContainerRef={scrollContainerRef}
          />
        </>
      )}
    </div>
  );
};
