import { ExclamationCircleOutlined, SyncOutlined } from '@ant-design/icons';
import { CheckCircleOutlined } from '@ant-design/icons';
import { Button, Modal, Spin, Tooltip } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { Backup } from '../../../entity/backups';
import { type Database, DatabaseType, type PostgresqlDatabase } from '../../../entity/databases';
import { type Restore, RestoreStatus, restoreApi } from '../../../entity/restores';
import { getUserTimeFormat } from '../../../shared/time';
import { EditDatabaseSpecificDataComponent } from '../../databases/ui/edit/EditDatabaseSpecificDataComponent';

interface Props {
  database: Database;
  backup: Backup;
}

export const RestoresComponent = ({ database, backup }: Props) => {
  const { t } = useTranslation(['restore', 'common']);
  const [editingDatabase, setEditingDatabase] = useState<Database>({
    ...database,
    postgresql: database.postgresql
      ? ({
          ...database.postgresql,
          host: undefined,
          port: undefined,
          password: undefined,
        } as unknown as PostgresqlDatabase)
      : undefined,
  });

  const [restores, setRestores] = useState<Restore[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const [showingRestoreError, setShowingRestoreError] = useState<Restore | undefined>();

  const [isShowRestore, setIsShowRestore] = useState(false);

  const isReloadInProgress = useRef(false);

  const loadRestores = async () => {
    if (isReloadInProgress.current) {
      return;
    }

    isReloadInProgress.current = true;

    try {
      const restores = await restoreApi.getRestores(backup.id);
      setRestores(restores);
    } catch (e) {
      alert((e as Error).message);
    }

    isReloadInProgress.current = false;
  };

  const restore = async (editingDatabase: Database) => {
    try {
      await restoreApi.restoreBackup({
        backupId: backup.id,
        postgresql: editingDatabase.postgresql as PostgresqlDatabase,
      });
      await loadRestores();

      setIsShowRestore(false);
    } catch (e) {
      alert((e as Error).message);
    }
  };

  useEffect(() => {
    setIsLoading(true);
    loadRestores().finally(() => setIsLoading(false));

    const interval = setInterval(() => {
      loadRestores();
    }, 1_000);

    return () => clearInterval(interval);
  }, [backup.id]);

  const isRestoreInProgress = restores.some(
    (restore) => restore.status === RestoreStatus.IN_PROGRESS,
  );

  if (isShowRestore) {
    if (database.type === DatabaseType.POSTGRES) {
      return (
        <>
          <div className="my-5 text-sm">
            {t('restore:message.restore_instructions_line1')}{' '}
            <u>{t('restore:message.restore_instructions_line2')}</u>{t('restore:message.restore_instructions_line3')}
            <br />
            <br />
            {t('restore:message.restore_instructions_line4')}
          </div>

          <EditDatabaseSpecificDataComponent
            database={editingDatabase}
            onCancel={() => setIsShowRestore(false)}
            isShowBackButton={false}
            onBack={() => setIsShowRestore(false)}
            saveButtonText={t('restore:action.restore_to_db')}
            isSaveToApi={false}
            onSaved={(database) => {
              setEditingDatabase({ ...database });
              restore(database);
            }}
            isShowDbVersionHint={false}
          />
        </>
      );
    }
  }

  return (
    <div className="mt-5">
      {isLoading ? (
        <div className="flex w-full justify-center">
          <Spin />
        </div>
      ) : (
        <>
          <Button
            className="w-full"
            type="primary"
            disabled={isRestoreInProgress}
            loading={isRestoreInProgress}
            onClick={() => setIsShowRestore(true)}
          >
            {t('restore:action.restore_from_backup')}
          </Button>

          {restores.length === 0 && (
            <div className="my-5 text-center text-gray-400">{t('restore:message.no_restores')}</div>
          )}

          <div className="mt-5">
            {restores.map((restore) => {
              let restoreDurationMs = 0;
              if (restore.status === RestoreStatus.IN_PROGRESS) {
                restoreDurationMs = Date.now() - new Date(restore.createdAt).getTime();
              } else {
                restoreDurationMs = restore.restoreDurationMs;
              }

              const minutes = Math.floor(restoreDurationMs / 60000);
              const seconds = Math.floor((restoreDurationMs % 60000) / 1000);
              const milliseconds = restoreDurationMs % 1000;
              const duration = `${minutes}m ${seconds}s ${milliseconds}ms`;

              const backupDurationMs = backup.backupDurationMs;
              const expectedRestoreDurationMs = backupDurationMs * 5;
              const expectedRestoreDuration = `${Math.floor(expectedRestoreDurationMs / 60000)}m ${Math.floor((expectedRestoreDurationMs % 60000) / 1000)}s`;

              return (
                <div key={restore.id} className="mb-1 rounded border border-gray-200 p-3 text-sm">
                  <div className="mb-1 flex">
                    <div className="w-[75px] min-w-[75px]">{t('restore:details.status')}</div>

                    {restore.status === RestoreStatus.FAILED && (
                      <Tooltip title={t('restore:details.click_error_tooltip')}>
                        <div
                          className="flex cursor-pointer items-center text-red-600 underline"
                          onClick={() => setShowingRestoreError(restore)}
                        >
                          <ExclamationCircleOutlined
                            className="mr-2"
                            style={{ fontSize: 16, color: '#ff0000' }}
                          />

                          <div>{t('restore:status.failed')}</div>
                        </div>
                      </Tooltip>
                    )}

                    {restore.status === RestoreStatus.COMPLETED && (
                      <div className="flex items-center">
                        <CheckCircleOutlined
                          className="mr-2"
                          style={{ fontSize: 16, color: '#008000' }}
                        />

                        <div>{t('restore:status.successful')}</div>
                      </div>
                    )}

                    {restore.status === RestoreStatus.IN_PROGRESS && (
                      <div className="flex items-center font-bold text-blue-600">
                        <SyncOutlined spin />
                        <span className="ml-2">{t('restore:status.in_progress')}</span>
                      </div>
                    )}
                  </div>

                  <div className="mb-1 flex">
                    <div className="w-[75px] min-w-[75px]">{t('restore:details.started_at')}</div>
                    <div>
                      {dayjs.utc(restore.createdAt).local().format(getUserTimeFormat().format)} (
                      {dayjs.utc(restore.createdAt).local().fromNow()})
                    </div>
                  </div>

                  {restore.status === RestoreStatus.IN_PROGRESS && (
                    <div className="flex">
                      <div className="w-[75px] min-w-[75px]">{t('restore:details.duration')}</div>
                      <div>
                        <div>{duration}</div>
                        <div className="mt-2 text-xs text-gray-500">
                          {t('restore:message.expected_duration_text')}
                          <br />
                          <br />
                          {t('restore:message.expected_duration_value', { duration: expectedRestoreDuration })}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </>
      )}

      {showingRestoreError && (
        <Modal
          title={t('restore:details.error_modal_title')}
          open={!!showingRestoreError}
          onCancel={() => setShowingRestoreError(undefined)}
          footer={null}
        >
          <div className="text-sm">{showingRestoreError.failMessage}</div>
        </Modal>
      )}
    </div>
  );
};
