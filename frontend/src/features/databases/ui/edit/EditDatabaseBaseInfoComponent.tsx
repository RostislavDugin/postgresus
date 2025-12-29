import { Button, Input, Select } from 'antd';
import { useEffect, useState } from 'react';

import {
  type Database,
  DatabaseType,
  type MariadbDatabase,
  type MongodbDatabase,
  type MysqlDatabase,
  type PostgresqlDatabase,
  databaseApi,
  getDatabaseLogoFromType,
} from '../../../../entity/databases';

interface Props {
  database: Database;

  isShowName?: boolean;
  isShowType?: boolean;
  isShowCancelButton?: boolean;
  onCancel: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (db: Database) => void;
}

const databaseTypeOptions = [
  { value: DatabaseType.POSTGRES, label: 'PostgreSQL' },
  { value: DatabaseType.MYSQL, label: 'MySQL' },
  { value: DatabaseType.MARIADB, label: 'MariaDB' },
  { value: DatabaseType.MONGODB, label: 'MongoDB' },
];

export const EditDatabaseBaseInfoComponent = ({
  database,
  isShowName,
  isShowType,
  isShowCancelButton,
  onCancel,
  saveButtonText,
  isSaveToApi,
  onSaved,
}: Props) => {
  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const updateDatabase = (patch: Partial<Database>) => {
    setEditingDatabase((prev) => (prev ? { ...prev, ...patch } : prev));
    setIsUnsaved(true);
  };

  const handleTypeChange = (newType: DatabaseType) => {
    if (!editingDatabase) return;

    const updatedDatabase: Database = {
      ...editingDatabase,
      type: newType,
      postgresql: undefined,
      mysql: undefined,
      mariadb: undefined,
      mongodb: undefined,
    };

    switch (newType) {
      case DatabaseType.POSTGRES:
        updatedDatabase.postgresql =
          editingDatabase.postgresql ?? ({ cpuCount: 1 } as PostgresqlDatabase);
        break;
      case DatabaseType.MYSQL:
        updatedDatabase.mysql = editingDatabase.mysql ?? ({} as MysqlDatabase);
        break;
      case DatabaseType.MARIADB:
        updatedDatabase.mariadb = editingDatabase.mariadb ?? ({} as MariadbDatabase);
        break;
      case DatabaseType.MONGODB:
        updatedDatabase.mongodb = editingDatabase.mongodb ?? ({ cpuCount: 1 } as MongodbDatabase);
        break;
    }

    setEditingDatabase(updatedDatabase);
    setIsUnsaved(true);
  };

  const saveDatabase = async () => {
    if (!editingDatabase) return;
    if (isSaveToApi) {
      setIsSaving(true);

      try {
        editingDatabase.name = editingDatabase.name?.trim();
        await databaseApi.updateDatabase(editingDatabase);
        setIsUnsaved(false);
      } catch (e) {
        alert((e as Error).message);
      }

      setIsSaving(false);
    }
    onSaved(editingDatabase);
  };

  useEffect(() => {
    setIsSaving(false);
    setIsUnsaved(false);
    setEditingDatabase({ ...database });
  }, [database]);

  if (!editingDatabase) return null;

  const isAllFieldsFilled = !!editingDatabase.name?.trim();

  return (
    <div>
      {isShowName && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Name</div>
          <Input
            value={editingDatabase.name || ''}
            onChange={(e) => updateDatabase({ name: e.target.value })}
            size="small"
            placeholder="My favourite DB"
            className="max-w-[200px] grow"
          />
        </div>
      )}

      {isShowType && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Database type</div>

          <div className="flex items-center">
            <Select
              value={editingDatabase.type}
              onChange={handleTypeChange}
              options={databaseTypeOptions}
              size="small"
              className="w-[200px] grow"
            />

            <img
              src={getDatabaseLogoFromType(editingDatabase.type)}
              alt="databaseIcon"
              className="ml-2 h-4 w-4"
            />
          </div>
        </div>
      )}

      <div className="mt-5 flex">
        {isShowCancelButton && (
          <Button danger ghost className="mr-1" onClick={onCancel}>
            Cancel
          </Button>
        )}

        <Button
          type="primary"
          className={`${isShowCancelButton ? 'ml-1' : 'ml-auto'} mr-5`}
          onClick={saveDatabase}
          loading={isSaving}
          disabled={(isSaveToApi && !isUnsaved) || !isAllFieldsFilled}
        >
          {saveButtonText || 'Save'}
        </Button>
      </div>
    </div>
  );
};
