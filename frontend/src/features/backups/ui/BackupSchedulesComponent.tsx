import { Button, Spin } from 'antd';
import { useEffect, useState } from 'react';

import {
  BackupEncryption,
  RetentionPolicyType,
  type BackupConfig,
  backupConfigApi,
} from '../../../entity/backups';
import { BackupNotificationType } from '../../../entity/backups/model/BackupNotificationType';
import type { Database } from '../../../entity/databases';
import { Period } from '../../../entity/databases/model/Period';
import { IntervalType } from '../../../entity/intervals';
import type { UserProfile } from '../../../entity/users';
import { ConfirmationComponent } from '../../../shared/ui';
import { EditBackupConfigComponent } from './EditBackupConfigComponent';

interface Props {
  database: Database;
  user: UserProfile;
  isCanManageDBs: boolean;
  onChanged: () => void;
}

const intervalLabels: Record<string, string> = {
  [IntervalType.HOURLY]: 'Hourly',
  [IntervalType.DAILY]: 'Daily',
  [IntervalType.WEEKLY]: 'Weekly',
  [IntervalType.MONTHLY]: 'Monthly',
  [IntervalType.CRON]: 'Cron',
};

const buildDefaultNewConfig = (databaseId: string): BackupConfig => ({
  databaseId,
  name: '',
  isBackupsEnabled: true,
  backupInterval: {
    id: undefined as unknown as string,
    interval: IntervalType.DAILY,
    timeOfDay: '00:00',
  },
  storage: undefined,
  retentionPolicyType: RetentionPolicyType.GFS,
  retentionTimePeriod: Period.THREE_MONTH,
  retentionCount: 100,
  retentionGfsHours: 24,
  retentionGfsDays: 7,
  retentionGfsWeeks: 4,
  retentionGfsMonths: 12,
  retentionGfsYears: 3,
  sendNotificationsOn: [BackupNotificationType.BackupFailed],
  isRetryIfFailed: true,
  maxFailedTriesCount: 3,
  encryption: BackupEncryption.ENCRYPTED,
  maxBackupSizeMb: 0,
  maxBackupsTotalSizeMb: 0,
});

export const BackupSchedulesComponent = ({ database, user, isCanManageDBs, onChanged }: Props) => {
  const [configs, setConfigs] = useState<BackupConfig[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [editingConfig, setEditingConfig] = useState<BackupConfig | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  const loadConfigs = async () => {
    setIsLoading(true);
    try {
      const result = await backupConfigApi.getBackupConfigsByDbID(database.id);
      setConfigs(result);
    } catch (e) {
      alert((e as Error).message);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadConfigs();
  }, [database.id]);

  const handleDelete = async (id: string) => {
    setConfirmDeleteId(null);
    setDeletingId(id);
    try {
      await backupConfigApi.deleteBackupConfig(id);
      await loadConfigs();
      onChanged();
    } catch (e) {
      alert((e as Error).message);
    } finally {
      setDeletingId(null);
    }
  };

  const handleSaved = () => {
    setEditingConfig(null);
    loadConfigs();
    onChanged();
  };

  if (editingConfig !== null) {
    return (
      <EditBackupConfigComponent
        database={database}
        user={user}
        initialConfig={editingConfig}
        isShowCancelButton
        onCancel={() => {
          setEditingConfig(null);
          loadConfigs();
        }}
        isSaveToApi
        onSaved={handleSaved}
        isShowBackButton={false}
        onBack={() => {}}
      />
    );
  }

  return (
    <div>
      {isLoading ? (
        <Spin size="small" />
      ) : (
        <div className="flex flex-col gap-2">
          {configs.length === 0 && (
            <div className="text-sm text-gray-500 dark:text-gray-400">
              No backup schedules configured.
            </div>
          )}

          {configs.map((config) => (
            <div
              key={config.id}
              className="flex items-center justify-between rounded border border-gray-200 px-3 py-2 dark:border-gray-600"
            >
              <div className="flex flex-col gap-0.5">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">{config.name}</span>
                  <span
                    className={`text-xs font-medium ${
                      config.isBackupsEnabled ? 'text-green-600' : 'text-red-600'
                    }`}
                  >
                    {config.isBackupsEnabled ? '● Enabled' : '● Disabled'}
                  </span>
                </div>
                <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                  {config.backupInterval?.interval && (
                    <span>{intervalLabels[config.backupInterval.interval]}</span>
                  )}
                  {config.storage?.name && (
                    <>
                      <span>·</span>
                      <span>{config.storage.name}</span>
                    </>
                  )}
                </div>
              </div>

              {isCanManageDBs && (
                <div className="ml-2 flex shrink-0 items-center gap-1">
                  <Button size="small" onClick={() => setEditingConfig(config)}>
                    Edit
                  </Button>
                  {config.id && (
                    <Button
                      size="small"
                      danger
                      ghost
                      onClick={() => setConfirmDeleteId(config.id!)}
                      loading={deletingId === config.id}
                    >
                      Delete
                    </Button>
                  )}
                </div>
              )}
            </div>
          ))}

          {isCanManageDBs && (
            <Button
              type="dashed"
              size="small"
              className="mt-1 w-full"
              onClick={() => setEditingConfig(buildDefaultNewConfig(database.id))}
            >
              + Add schedule
            </Button>
          )}
        </div>
      )}

      {confirmDeleteId && (
        <ConfirmationComponent
          onConfirm={() => handleDelete(confirmDeleteId)}
          onDecline={() => setConfirmDeleteId(null)}
          description="Are you sure you want to delete this backup schedule? This action cannot be undone."
          actionText="Delete"
          actionButtonColor="red"
        />
      )}
    </div>
  );
};
