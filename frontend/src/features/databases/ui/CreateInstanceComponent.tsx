import { Button, Progress } from 'antd';
import { useState } from 'react';

import { type BackupConfig, backupConfigApi } from '../../../entity/backups';
import {
  type Database,
  DatabaseType,
  type PostgresqlDatabase,
  buildInstanceKey,
  databaseApi,
} from '../../../entity/databases';
import { EditBackupConfigComponent } from '../../backups';
import { EditDatabaseNotifiersComponent } from './edit/EditDatabaseNotifiersComponent';
import { EditDatabaseSpecificDataComponent } from './edit/EditDatabaseSpecificDataComponent';
import { SelectServerDatabasesComponent } from './edit/SelectServerDatabasesComponent';

interface Props {
  workspaceId: string;
  existingDatabases: Database[];
  onCreatingChanged: (isCreating: boolean) => void;

  onCreated: (databaseId: string | undefined) => void;
  onClose: () => void;
}

interface CreationFailure {
  databaseName: string;
  message: string;
}

export const CreateInstanceComponent = ({
  workspaceId,
  existingDatabases,
  onCreatingChanged,
  onCreated,
  onClose,
}: Props) => {
  const [step, setStep] = useState<
    'connection' | 'select-databases' | 'backup-config' | 'notifiers' | 'creating'
  >('connection');

  const [connection, setConnection] = useState<Database>({
    id: undefined as unknown as string,
    name: 'instance',
    workspaceId,
    type: DatabaseType.POSTGRES,
    postgresql: { port: 5432, database: 'postgres' } as unknown as PostgresqlDatabase,
    notifiers: [],
  } as unknown as Database);

  const [selectedNames, setSelectedNames] = useState<string[]>([]);
  const [backupConfig, setBackupConfig] = useState<BackupConfig | undefined>();

  const [createdCount, setCreatedCount] = useState(0);
  const [failures, setFailures] = useState<CreationFailure[]>([]);
  const [configFailures, setConfigFailures] = useState<CreationFailure[]>([]);
  const [isFinished, setIsFinished] = useState(false);
  const [firstCreatedId, setFirstCreatedId] = useState<string | undefined>();
  const [creatingName, setCreatingName] = useState<string>('');

  const alreadyAddedNames = existingDatabases
    .filter(
      (database) =>
        buildInstanceKey(database.postgresql?.host, database.postgresql?.port) ===
        buildInstanceKey(connection.postgresql?.host, connection.postgresql?.port),
    )
    .map((database) => database.postgresql?.database)
    .filter((name): name is string => !!name);

  const createDatabases = async (notifiersSource: Database, config: BackupConfig) => {
    setStep('creating');
    setCreatedCount(0);
    setFailures([]);
    setConfigFailures([]);
    setIsFinished(false);
    onCreatingChanged(true);

    let firstId: string | undefined;
    const collectedFailures: CreationFailure[] = [];
    const collectedConfigFailures: CreationFailure[] = [];

    try {
      for (const databaseName of selectedNames) {
        let created: Database;

        setCreatingName(databaseName);

        try {
          created = await databaseApi.createDatabase({
            ...connection,
            id: undefined as unknown as string,
            name: databaseName,
            workspaceId,
            notifiers: notifiersSource.notifiers,
            postgresql: {
              ...(connection.postgresql as PostgresqlDatabase),
              database: databaseName,
            },
          } as Database);
        } catch (e) {
          collectedFailures.push({ databaseName, message: toMessage(e) });
          setFailures([...collectedFailures]);
          continue;
        }

        firstId = firstId ?? created.id;
        setCreatedCount((current) => current + 1);

        try {
          await backupConfigApi.saveBackupConfig({ ...config, databaseId: created.id });
        } catch (e) {
          collectedConfigFailures.push({ databaseName, message: toMessage(e) });
          setConfigFailures([...collectedConfigFailures]);
        }
      }
    } finally {
      setFirstCreatedId(firstId);
      setIsFinished(true);
      onCreatingChanged(false);
    }
  };

  if (step === 'connection') {
    return (
      <EditDatabaseSpecificDataComponent
        database={connection}
        isShowCancelButton
        onCancel={() => onClose()}
        isShowBackButton={false}
        onBack={() => onClose()}
        saveButtonText="Continue"
        isSaveToApi={false}
        dbNameLabel="Connect to DB"
        dbNameHint="Database used only to read the list of databases on this server. Usually postgres."
        onSaved={(database) => {
          setConnection({ ...database });
          setStep('select-databases');
        }}
      />
    );
  }

  if (step === 'select-databases') {
    return (
      <SelectServerDatabasesComponent
        database={connection}
        alreadyAddedNames={alreadyAddedNames}
        onBack={() => setStep('connection')}
        onSelected={(names) => {
          setSelectedNames(names);
          setStep('backup-config');
        }}
      />
    );
  }

  if (step === 'backup-config') {
    return (
      <EditBackupConfigComponent
        database={connection}
        isShowCancelButton={false}
        onCancel={() => onClose()}
        isShowBackButton
        onBack={() => setStep('select-databases')}
        saveButtonText="Continue"
        isSaveToApi={false}
        onSaved={(config) => {
          setBackupConfig(config);
          setStep('notifiers');
        }}
      />
    );
  }

  if (step === 'notifiers') {
    return (
      <EditDatabaseNotifiersComponent
        database={connection}
        workspaceId={workspaceId}
        isShowCancelButton={false}
        onCancel={() => onClose()}
        isShowBackButton
        onBack={() => setStep('backup-config')}
        isShowSaveOnlyForUnsaved={false}
        saveButtonText={`Create ${selectedNames.length} ${
          selectedNames.length === 1 ? 'database' : 'databases'
        }`}
        isSaveToApi={false}
        onSaved={(database) => {
          if (!backupConfig) return;

          createDatabases(database, backupConfig);
        }}
      />
    );
  }

  const processedCount = createdCount + failures.length;

  return (
    <div>
      <div className="mb-3">
        {isFinished
          ? `Done: ${createdCount} of ${selectedNames.length} databases created`
          : `Creating ${creatingName}… (${processedCount} of ${selectedNames.length})`}
      </div>

      <Progress
        percent={Math.round((processedCount / selectedNames.length) * 100)}
        status={isFinished && failures.length > 0 ? 'exception' : undefined}
      />

      {failures.length > 0 && (
        <div className="mt-3 max-h-[200px] overflow-y-auto">
          <div className="mb-1 font-bold text-red-600 dark:text-red-400">
            Failed: {failures.length}
          </div>

          {failures.map((failure) => (
            <div key={failure.databaseName} className="mb-1 text-sm">
              <span className="font-bold">{failure.databaseName}</span>: {failure.message}
            </div>
          ))}
        </div>
      )}

      {configFailures.length > 0 && (
        <div className="mt-3 max-h-[200px] overflow-y-auto">
          <div className="mb-1 font-bold text-orange-600 dark:text-orange-400">
            Created without backup config: {configFailures.length}
          </div>

          <div className="mb-1 text-sm text-gray-500 dark:text-gray-400">
            These databases exist but will not be backed up until you configure them.
          </div>

          {configFailures.map((failure) => (
            <div key={failure.databaseName} className="mb-1 text-sm">
              <span className="font-bold">{failure.databaseName}</span>: {failure.message}
            </div>
          ))}
        </div>
      )}

      {isFinished && (
        <div className="mt-5 flex">
          <Button
            className="ml-auto"
            type="primary"
            onClick={() => {
              onCreated(firstCreatedId);
              onClose();
            }}
          >
            Close
          </Button>
        </div>
      )}
    </div>
  );
};

function toMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
