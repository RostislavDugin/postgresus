import { Spin } from 'antd';
import { useState } from 'react';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, databaseApi } from '../../../entity/databases';
import { BackupsComponent } from '../../backups';
import { HealthckeckAttemptsComponent } from '../../healthcheck';
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
  const { t } = useTranslation(['database']);
  const [currentTab, setCurrentTab] = useState<'config' | 'backups' | 'metrics'>('backups');

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
          {t('database:tabs.config')}
        </div>

        <div
          className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'backups' ? 'bg-white' : 'bg-gray-200'}`}
          onClick={() => setCurrentTab('backups')}
        >
          {t('database:tabs.backups')}
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
    </div>
  );
};
