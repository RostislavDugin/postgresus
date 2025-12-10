import { Button, Input, InputNumber, Select, Spin, Switch } from 'antd';
import { useState } from 'react';

import { backupsApi, type BackupConfig, backupConfigApi } from '../../../entity/backups';
import { type Database, DatabaseType, type PostgresqlDatabase, databaseApi } from '../../../entity/databases';
import { PostgresqlVersion } from '../../../entity/databases/model/postgresql/PostgresqlVersion';
import type { Notifier } from '../../../entity/notifiers';
import { EditBackupConfigComponent } from '../../backups';
import { EditDatabaseNotifiersComponent } from './edit/EditDatabaseNotifiersComponent';

interface Props {
  workspaceId: string;
  onCreated: () => void;
  onClose: () => void;
}

export const CreateDatabasesFromClusterComponent = ({ workspaceId, onCreated, onClose }: Props) => {
  const [step, setStep] = useState<'cluster-settings' | 'select-databases' | 'backup-config' | 'notifiers' | 'creating'>('cluster-settings');

  const [clusterDb, setClusterDb] = useState<Database>({
    id: undefined as unknown as string,
    name: 'Postgres Cluster',
    workspaceId,
    type: DatabaseType.POSTGRES,
    postgresql: {
      id: undefined as unknown as string,
      version: undefined as unknown as PostgresqlVersion,
      host: '',
      port: 5432,
      username: '',
      password: '',
      isHttps: false,
    } as unknown as PostgresqlDatabase,
    notifiers: [],
  } as Database);

  const [isLoadingDbs, setIsLoadingDbs] = useState(false);
  const [availableDbs, setAvailableDbs] = useState<string[]>([]);
  const [selectedDbNames, setSelectedDbNames] = useState<string[]>([]);

  const [backupConfig, setBackupConfig] = useState<BackupConfig | undefined>();
  const [notifiers, setNotifiers] = useState<Notifier[]>([]);

  const [isCreating, setIsCreating] = useState(false);
  const [createdCount, setCreatedCount] = useState(0);

  // Step 1: load databases from cluster
  const loadDatabasesFromCluster = async () => {
    setIsLoadingDbs(true);
    try {
      const dbs = await databaseApi.listDatabasesDirect(clusterDb);
      setAvailableDbs(dbs);
      // preselect all
      setSelectedDbNames(dbs);
      setStep('select-databases');
    } catch (e) {
      alert((e as Error).message);
    }
    setIsLoadingDbs(false);
  };

  // Step 5: create databases
  const createDatabases = async () => {
    if (!backupConfig) return;

    setIsCreating(true);
    setCreatedCount(0);

    try {
      for (const dbName of selectedDbNames) {
        // build per-db object
        const newDb: Database = {
          id: undefined as unknown as string,
          name: dbName,
          workspaceId,
          type: DatabaseType.POSTGRES,
          postgresql: {
            ...(clusterDb.postgresql as PostgresqlDatabase),
            database: dbName,
          } as PostgresqlDatabase,
          notifiers,
        } as Database;

        const created = await databaseApi.createDatabase(newDb);

        const cfg: BackupConfig = { ...backupConfig, databaseId: created.id } as BackupConfig;
        await backupConfigApi.saveBackupConfig(cfg);
        if (cfg.isBackupsEnabled) {
          await backupsApi.makeBackup(created.id);
        }

        setCreatedCount((c: number) => c + 1);
      }

      onCreated();
      onClose();
    } catch (e) {
      alert((e as Error).message);
    }

    setIsCreating(false);
  };

  // Render steps
  if (step === 'cluster-settings') {
    const pg = clusterDb.postgresql! as PostgresqlDatabase;

    const isAllFieldsFilled = Boolean(pg.version && pg.host && pg.port && pg.username && pg.password);

    return (
      <div>
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">PG version</div>
          <Select
            value={pg.version as unknown as string}
            onChange={(v) => setClusterDb({
              ...clusterDb,
              postgresql: { ...pg, version: v as PostgresqlDatabase['version'] },
            })}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Select PG version"
            options={[
              { label: '12', value: '12' },
              { label: '13', value: '13' },
              { label: '14', value: '14' },
              { label: '15', value: '15' },
              { label: '16', value: '16' },
              { label: '17', value: '17' },
              { label: '18', value: '18' },
            ]}
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Host</div>
          <Input
            value={pg.host}
            onChange={(e) =>
              setClusterDb({
                ...clusterDb,
                postgresql: { ...pg, host: e.target.value.trim().replace('https://', '').replace('http://', '') },
              })
            }
            size="small"
            className="max-w-[200px] grow"
            placeholder="Enter PG host"
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Port</div>
          <InputNumber
            type="number"
            value={pg.port}
            onChange={(v) => typeof v === 'number' && setClusterDb({ ...clusterDb, postgresql: { ...pg, port: v } })}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Enter PG port"
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Username</div>
          <Input
            value={pg.username}
            onChange={(e) => setClusterDb({ ...clusterDb, postgresql: { ...pg, username: e.target.value.trim() } })}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Enter PG username"
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Password</div>
          <Input.Password
            value={pg.password}
            onChange={(e) => setClusterDb({ ...clusterDb, postgresql: { ...pg, password: e.target.value.trim() } })}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Enter PG password"
          />
        </div>

        <div className="mb-3 flex w-full items-center">
          <div className="min-w-[150px]">Use HTTPS</div>
          <Switch
            checked={pg.isHttps}
            onChange={(checked) => setClusterDb({ ...clusterDb, postgresql: { ...pg, isHttps: checked } })}
            size="small"
          />
        </div>

        <div className="mt-5 flex">
          <Button type="primary" className="ml-auto mr-5" onClick={loadDatabasesFromCluster} disabled={!isAllFieldsFilled}>
            {isLoadingDbs ? <Spin /> : 'Load databases'}
          </Button>
        </div>
      </div>
    );
  }

  if (step === 'select-databases') {
    return (
      <div>
        <div className="mb-2 text-gray-500">Select databases to back up</div>
        <div className="mb-2 flex items-center text-xs text-gray-500">
          <span className="mr-auto">{selectedDbNames.length} / {availableDbs.length} selected</span>
          <Button size="small" onClick={() => setSelectedDbNames(availableDbs)}>
            Select all
          </Button>
          <Button size="small" className="ml-1" onClick={() => setSelectedDbNames([])}>
            Clear
          </Button>
          <Button
            size="small"
            className="ml-1"
            onClick={() =>
              setSelectedDbNames(
                availableDbs.filter((d: string) => !selectedDbNames.includes(d))
              )
            }
          >
            Invert
          </Button>
          <Button
            size="small"
            className="ml-1"
            onClick={() => {
              const systemDbNames = ['postgres', 'template0', 'template1'];
              setSelectedDbNames(
                availableDbs.filter((d: string) => !systemDbNames.includes(d))
              );
            }}
          >
            Exclude system
          </Button>
        </div>
        <Select
          mode="multiple"
          value={selectedDbNames}
          onChange={(values) => setSelectedDbNames(values as string[])}
          size="small"
          className="w-full"
          options={availableDbs
            .slice()
            .sort((a: string, b: string) => a.localeCompare(b))
            .map((d) => ({ label: d, value: d }))}
          placeholder="Select databases"
          showSearch
          optionFilterProp="label"
          filterOption={(input: string, option?: { label?: string; value?: string }) =>
            ((option?.label as string) || '').toLowerCase().includes(input.toLowerCase())
          }
          maxTagCount={0}
          listHeight={320}
          dropdownStyle={{ maxHeight: 340, overflowY: 'auto' }}
          popupMatchSelectWidth
          allowClear
        />

        <div className="mt-5 flex">
          <Button className="mr-auto" type="primary" ghost onClick={() => setStep('cluster-settings')}>
            Back
          </Button>
          <Button
            type="primary"
            className="ml-1 mr-5"
            onClick={() => setStep('backup-config')}
            disabled={selectedDbNames.length === 0}
          >
            Continue
          </Button>
        </div>
      </div>
    );
  }

  if (step === 'backup-config') {
    return (
      <EditBackupConfigComponent
        database={clusterDb}
        isShowCancelButton={false}
        onCancel={() => onClose()}
        isShowBackButton
        onBack={() => setStep('select-databases')}
        saveButtonText="Continue"
        isSaveToApi={false}
        onSaved={(cfg) => {
          setBackupConfig(cfg);
          setStep('notifiers');
        }}
      />
    );
  }

  if (step === 'notifiers') {
    return (
      <EditDatabaseNotifiersComponent
        database={{ ...clusterDb, notifiers } as Database}
        workspaceId={workspaceId}
        isShowCancelButton={false}
        onCancel={() => onClose()}
        isShowBackButton
        onBack={() => setStep('backup-config')}
        isShowSaveOnlyForUnsaved={false}
        saveButtonText="Create"
        isSaveToApi={false}
        onSaved={(db) => {
          setNotifiers(db.notifiers);
          setStep('creating');
        }}
      />
    );
  }

  if (step === 'creating') {
    if (!isCreating) createDatabases();

    return (
      <div className="flex flex-col items-center justify-center p-5">
        <Spin />
        <div className="mt-3 text-gray-500">
          Created {createdCount} / {selectedDbNames.length} databases...
        </div>
      </div>
    );
  }

  return null;
};
