import { useState } from 'react';

import { type BackupConfig, backupConfigApi, backupsApi } from '../../../entity/backups';
import {
  type Database,
  DatabaseType,
  Period,
  type PostgresqlDatabase,
  databaseApi,
} from '../../../entity/databases';
import { EditBackupConfigComponent } from '../../backups';
import { EditDatabaseBaseInfoComponent } from './edit/EditDatabaseBaseInfoComponent';
import { EditDatabaseNotifiersComponent } from './edit/EditDatabaseNotifiersComponent';
import { EditDatabaseSpecificDataComponent } from './edit/EditDatabaseSpecificDataComponent';

interface Props {
  workspaceId: string;

  onCreated: () => void;
  onClose: () => void;
}

export const CreateDatabaseComponent = ({ workspaceId, onCreated, onClose }: Props) => {
  const [isCreating, setIsCreating] = useState(false);
  const [backupConfig, setBackupConfig] = useState<BackupConfig | undefined>();
  const [database, setDatabase] = useState<Database>({
    id: undefined as unknown as string,
    name: '',
    workspaceId,
    storePeriod: Period.MONTH,

    postgresql: {
      cpuCount: 1,
    } as unknown as PostgresqlDatabase,

    type: DatabaseType.POSTGRES,

    storage: {} as unknown as Storage,

    notifiers: [],
    sendNotificationsOn: [],
  } as Database);

  const [step, setStep] = useState<'base-info' | 'db-settings' | 'backup-config' | 'notifiers'>(
    'base-info',
  );

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

      onCreated();
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
          isSaveToApi={false}
          saveButtonText="Continue"
          onCancel={() => onClose()}
          onSaved={(database) => {
            setDatabase({ ...database });
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
          setStep('backup-config');
        }}
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
