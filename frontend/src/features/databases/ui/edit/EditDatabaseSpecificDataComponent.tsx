import { Modal } from 'antd';
import { useState } from 'react';

import { type Database, DatabaseType, databaseApi } from '../../../../entity/databases';
import { CreateReadOnlyComponent } from './CreateReadOnlyComponent';
import { EditMariaDbSpecificDataComponent } from './EditMariaDbSpecificDataComponent';
import { EditMongoDbSpecificDataComponent } from './EditMongoDbSpecificDataComponent';
import { EditMySqlSpecificDataComponent } from './EditMySqlSpecificDataComponent';
import { EditPostgreSqlLogicalSpecificDataComponent } from './EditPostgreSqlLogicalSpecificDataComponent';
import { EditPostgreSqlPhysicalSpecificDataComponent } from './EditPostgreSqlPhysicalSpecificDataComponent';

interface Props {
  database: Database;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (database: Database) => void;

  isShowDbName?: boolean;
  isRestoreMode?: boolean;

  onConnectionErrorChange?: (hasConnectionError: boolean) => void;
}

export const EditDatabaseSpecificDataComponent = ({
  database,

  isShowCancelButton,
  onCancel,

  isShowBackButton,
  onBack,

  saveButtonText,
  isSaveToApi,
  onSaved,
  isShowDbName = true,
  isRestoreMode = false,
  onConnectionErrorChange,
}: Props) => {
  const [isShowReadOnlyDialog, setIsShowReadOnlyDialog] = useState(false);
  const [editingDatabase, setEditingDatabase] = useState<Database>(database);

  const saveDb = async (databaseToSave: Database) => {
    setEditingDatabase(databaseToSave);

    if (!isSaveToApi) {
      onSaved(databaseToSave);
      return;
    }

    try {
      const result = await databaseApi.isUserReadOnly(databaseToSave);

      if (result.isReadOnly) {
        onSaved(databaseToSave);
      } else {
        setIsShowReadOnlyDialog(true);
      }
    } catch (e) {
      alert((e as Error).message);
    }
  };

  const onReadOnlyUserCreated = (updatedDatabase: Database) => {
    setEditingDatabase(updatedDatabase);
    setIsShowReadOnlyDialog(false);
  };

  const skipReadOnlyUser = () => {
    setIsShowReadOnlyDialog(false);
    onSaved(editingDatabase);
  };

  if (isShowReadOnlyDialog) {
    return (
      <Modal
        title="Create read-only user"
        footer={<div />}
        open={isShowReadOnlyDialog}
        onCancel={() => setIsShowReadOnlyDialog(false)}
        maskClosable={false}
        width={450}
      >
        <CreateReadOnlyComponent
          database={editingDatabase}
          onReadOnlyUserUpdated={(db) => {
            console.log('onReadOnlyUserUpdated', db);
            onReadOnlyUserCreated(db);
          }}
          onGoBack={() => {
            setIsShowReadOnlyDialog(false);
          }}
          onSkipped={() => {
            skipReadOnlyUser();
          }}
          onAlreadyExists={() => {
            console.log('onAlreadyExists');
            onSaved(editingDatabase);
          }}
        />
      </Modal>
    );
  }

  const commonProps = {
    database: editingDatabase,
    isShowCancelButton,
    onCancel,
    isShowBackButton,
    onBack,
    saveButtonText,
    isSaveToApi,
    onSaved: saveDb,
    isShowDbName,
  };

  switch (editingDatabase.type) {
    case DatabaseType.POSTGRES_LOGICAL:
      return (
        <EditPostgreSqlLogicalSpecificDataComponent
          {...commonProps}
          isRestoreMode={isRestoreMode}
        />
      );
    case DatabaseType.POSTGRES_PHYSICAL:
      return (
        <EditPostgreSqlPhysicalSpecificDataComponent
          {...commonProps}
          onConnectionErrorChange={onConnectionErrorChange}
        />
      );
    case DatabaseType.MYSQL:
      return <EditMySqlSpecificDataComponent {...commonProps} />;
    case DatabaseType.MARIADB:
      return <EditMariaDbSpecificDataComponent {...commonProps} />;
    case DatabaseType.MONGODB:
      return <EditMongoDbSpecificDataComponent {...commonProps} />;
    default:
      return null;
  }
};
