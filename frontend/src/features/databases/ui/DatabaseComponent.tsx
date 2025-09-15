import { Spin } from 'antd';
import { useState } from 'react';
import { useEffect } from 'react';

import { type Database, databaseApi } from '../../../entity/databases';
import { BackupsComponent } from '../../backups';
import { HealthckeckAttemptsComponent } from '../../healthcheck';
import { MetricsComponent } from '../../monitoring/metrics';
//import { SqlQueryComponent } from '../../query/postgresql/SqlQueryComponent';
import { SqlQueryComponent } from '../../sqlquery';
import { DatabaseConfigComponent } from './DatabaseConfigComponent';

interface Props {
  contentHeight: number;
  databaseId: string;
  onDatabaseChanged: (database: Database) => void;
  onDatabaseDeleted: () => void;
}

export const DatabaseComponent = ({
  contentHeight,
  databaseId,
  onDatabaseChanged,
  onDatabaseDeleted,
}: Props) => {
  const [currentTab, setCurrentTab] = useState<'config' | 'backups' | 'metrics' | 'sqlquery'>(
    'backups',
  );

  const [database, setDatabase] = useState<Database | undefined>();
  const [editDatabase, setEditDatabase] = useState<Database | undefined>();

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
    <div className="w-full overflow-y-auto" style={{ maxHeight: contentHeight }}>
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

        <div
          className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'metrics' ? 'bg-white' : 'bg-gray-200'}`}
          onClick={() => setCurrentTab('metrics')}
        >
          Metrics
        </div>

        <div
          className={`cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'sqlquery' ? 'bg-white' : 'bg-gray-200'}`}
          onClick={() => setCurrentTab('sqlquery')}
        >
          SqlQuery
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
        />
      )}
      {currentTab === 'backups' && (
        <>
          <HealthckeckAttemptsComponent database={database} />
          <BackupsComponent database={database} />
        </>
      )}
      {currentTab === 'metrics' && <MetricsComponent databaseId={database.id} />}
      {currentTab === 'sqlquery' && (
        <div className="mt-4">
          <SqlQueryComponent databaseId={database.id} />
        </div>
      )}
    </div>
  );
};
