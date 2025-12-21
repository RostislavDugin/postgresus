import { useState } from 'react';

import { type BackupConfig, backupConfigApi, backupsApi } from '../../../entity/backups';
import {
  type Database,
  DatabaseType,
  type MariadbDatabase,
  type MysqlDatabase,
  Period,
  type PostgresqlDatabase,
  databaseApi,
} from '../../../entity/databases';
import { EditBackupConfigComponent } from '../../backups';
import { CreateReadOnlyComponent } from './edit/CreateReadOnlyComponent';
import { EditDatabaseBaseInfoComponent } from './edit/EditDatabaseBaseInfoComponent';
import { EditDatabaseNotifiersComponent } from './edit/EditDatabaseNotifiersComponent';
import { EditDatabaseSpecificDataComponent } from './edit/EditDatabaseSpecificDataComponent';

interface Props {
  workspaceId: string;

  onCreated: (databaseId: string) => void;
  onClose: () => void;
}

const createInitialDatabase = (workspaceId: string): Database =>
  ({
    id: undefined as unknown as string,
    name: '',
    workspaceId,
    storePeriod: Period.MONTH,

    postgresql: {} as PostgresqlDatabase,

    type: DatabaseType.POSTGRES,

    storage: {} as unknown as Storage,

    notifiers: [],
    sendNotificationsOn: [],
  }) as Database;

const initializeDatabaseTypeData = (db: Database): Database => {
  const base = { ...db, postgresql: undefined, mysql: undefined, mariadb: undefined };

  switch (db.type) {
    case DatabaseType.POSTGRES:
      return { ...base, postgresql: db.postgresql ?? ({} as PostgresqlDatabase) };
    case DatabaseType.MYSQL:
      return { ...base, mysql: db.mysql ?? ({} as MysqlDatabase) };
    case DatabaseType.MARIADB:
      return { ...base, mariadb: db.mariadb ?? ({} as MariadbDatabase) };
    default:
      return db;
  }
};

export const CreateDatabaseComponent = ({ workspaceId, onCreated, onClose }: Props) => {
  const [isCreating, setIsCreating] = useState(false);
  const [backupConfig, setBackupConfig] = useState<BackupConfig | undefined>();
  const [database, setDatabase] = useState<Database>(createInitialDatabase(workspaceId));

  const [step, setStep] = useState<
    'base-info' | 'db-settings' | 'create-readonly-user' | 'backup-config' | 'notifiers'
  >('base-info');

  const createDatabase = async (database: Database, backupConfig: BackupConfig) => {
    setIsCreating(true);

    try {
      const createdDatabase = await databaseApi.createDatabase(database);
      setDatabase({ ...createdDatabase });

      backupConfig.databaseId = createdDatabase.id;
      await backupConfigApi.saveBackupConfig(backupConfig);
      if (backupConfig.isBackupsEnabled) {
        await backupsApi.makeBackup(createdDatabase.id);
      }

      onCreated(createdDatabase.id);
      onClose();
    } catch (error) {
      alert(error);
    }

    setIsCreating(false);
  };

  if (step === 'base-info') {
    return (
      <div>
        <EditDatabaseBaseInfoComponent
          database={database}
          isShowName
          isShowType
          isSaveToApi={false}
          saveButtonText="Continue"
          onCancel={() => onClose()}
          onSaved={(db) => {
            const initializedDb = initializeDatabaseTypeData(db);
            setDatabase({ ...initializedDb });
            setStep('db-settings');
          }}
        />
      </div>
    );
  }

  if (step === 'db-settings') {
    return (
      <EditDatabaseSpecificDataComponent
        database={database}
        isShowCancelButton={false}
        onCancel={() => onClose()}
        isShowBackButton
        onBack={() => setStep('base-info')}
        saveButtonText="Continue"
        isSaveToApi={false}
        onSaved={(database) => {
          setDatabase({ ...database });
          setStep('create-readonly-user');
        }}
      />
    );
  }

  if (step === 'create-readonly-user') {
    return (
      <CreateReadOnlyComponent
        database={database}
        onReadOnlyUserUpdated={(database) => {
          setDatabase({ ...database });
        }}
        onGoBack={() => setStep('db-settings')}
        onContinue={() => setStep('backup-config')}
      />
    );
  }

  if (step === 'backup-config') {
    return (
      <EditBackupConfigComponent
        database={database}
        isShowCancelButton={false}
        onCancel={() => onClose()}
        isShowBackButton
        onBack={() => setStep('db-settings')}
        saveButtonText="Continue"
        isSaveToApi={false}
        onSaved={(backupConfig) => {
          setBackupConfig(backupConfig);
          setStep('notifiers');
        }}
      />
    );
  }

  if (step === 'notifiers') {
    return (
      <EditDatabaseNotifiersComponent
        database={database}
        isShowCancelButton={false}
        workspaceId={workspaceId}
        onCancel={() => onClose()}
        isShowBackButton
        onBack={() => setStep('backup-config')}
        isShowSaveOnlyForUnsaved={false}
        saveButtonText="Complete"
        isSaveToApi={false}
        onSaved={(database) => {
          if (isCreating) return;

          setDatabase({ ...database });
          createDatabase(database, backupConfig!);
        }}
      />
    );
  }
};
