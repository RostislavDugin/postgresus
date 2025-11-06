import { useState } from 'react';
import { useTranslation } from 'react-i18next';

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
  onCreated: () => void;

  onClose: () => void;
}

export const CreateDatabaseComponent = ({ onCreated, onClose }: Props) => {
  const { t } = useTranslation('common');
  const [isCreating, setIsCreating] = useState(false);
  const [backupConfig, setBackupConfig] = useState<BackupConfig | undefined>();
  const [database, setDatabase] = useState<Database>({
    id: undefined as unknown as string,
    name: '',
    storePeriod: Period.WEEK,

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
          saveButtonText={t('button.continue')}
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
        saveButtonText={t('button.continue')}
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
        saveButtonText={t('button.continue')}
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
        onCancel={() => onClose()}
        isShowBackButton
        onBack={() => setStep('backup-config')}
        isShowSaveOnlyForUnsaved={false}
        saveButtonText={t('button.complete')}
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
