import { Button, Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { databaseApi } from '../../../entity/databases';
import type { Database } from '../../../entity/databases';
import { CreateDatabaseComponent } from './CreateDatabaseComponent';
import { DatabaseCardComponent } from './DatabaseCardComponent';
import { DatabaseComponent } from './DatabaseComponent';

interface Props {
  contentHeight: number;
}
export const DatabasesComponent = ({ contentHeight }: Props) => {
  const { t } = useTranslation('database');
  const [isLoading, setIsLoading] = useState(true);
  const [databases, setDatabases] = useState<Database[]>([]);
  const [searchQuery, setSearchQuery] = useState('');

  const [isShowAddDatabase, setIsShowAddDatabase] = useState(false);
  const [selectedDatabaseId, setSelectedDatabaseId] = useState<string | undefined>(undefined);

  const loadDatabases = (isSilent = false) => {
    if (!isSilent) {
      setIsLoading(true);
    }

    databaseApi
      .getDatabases()
      .then((databases) => {
        setDatabases(databases);
        if (!selectedDatabaseId && !isSilent) {
          setSelectedDatabaseId(databases[0]?.id);
        }
      })
      .catch((e) => alert(e.message))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadDatabases();

    const interval = setInterval(() => {
      loadDatabases(true);
    }, 5 * 60_000);

    return () => clearInterval(interval);
  }, []);

  if (isLoading) {
    return (
      <div className="mx-3 my-3 flex w-[250px] justify-center">
        <Spin />
      </div>
    );
  }

  const addDatabaseButton = (
    <Button type="primary" className="mb-2 w-full" onClick={() => setIsShowAddDatabase(true)}>
      {t('add_button')}
    </Button>
  );

  const filteredDatabases = databases.filter((database) =>
    database.name.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  return (
    <>
      <div className="flex grow">
        <div
          className="mx-3 w-[250px] min-w-[250px] overflow-y-auto pr-2"
          style={{ height: contentHeight }}
        >
          {databases.length >= 5 && (
            <>
              {addDatabaseButton}

              <div className="mb-2">
                <input
                  placeholder={t('search_placeholder')}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full border-b border-gray-300 p-1 text-gray-500 outline-none"
                />
              </div>
            </>
          )}

          {filteredDatabases.length > 0
            ? filteredDatabases.map((database) => (
                <DatabaseCardComponent
                  key={database.id}
                  database={database}
                  selectedDatabaseId={selectedDatabaseId}
                  setSelectedDatabaseId={setSelectedDatabaseId}
                />
              ))
            : searchQuery && (
                <div className="mb-4 text-center text-sm text-gray-500">
                  {t('message.no_databases')} &quot;{searchQuery}&quot;
                </div>
              )}

          {databases.length < 5 && addDatabaseButton}

          <div className="mx-3 text-center text-xs text-gray-500">
            {t('modal.description')}
          </div>
        </div>

        {selectedDatabaseId && (
          <DatabaseComponent
            contentHeight={contentHeight}
            databaseId={selectedDatabaseId}
            onDatabaseChanged={() => {
              loadDatabases();
            }}
            onDatabaseDeleted={() => {
              loadDatabases();
              setSelectedDatabaseId(
                databases.filter((database) => database.id !== selectedDatabaseId)[0]?.id,
              );
            }}
          />
        )}
      </div>

      {isShowAddDatabase && (
        <Modal
          title={t('modal.add_title')}
          footer={<div />}
          open={isShowAddDatabase}
          onCancel={() => setIsShowAddDatabase(false)}
          width={420}
        >
          <div className="mt-5" />

          <CreateDatabaseComponent
            onCreated={() => {
              loadDatabases();
              setIsShowAddDatabase(false);
            }}
            onClose={() => setIsShowAddDatabase(false)}
          />
        </Modal>
      )}
    </>
  );
};
