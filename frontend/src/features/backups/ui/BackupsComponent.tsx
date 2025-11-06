import {
  CheckCircleOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  ExclamationCircleOutlined,
  InfoCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { Button, Modal, Spin, Table, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Backup, BackupStatus, backupConfigApi, backupsApi } from '../../../entity/backups';
import type { Database } from '../../../entity/databases';
import { getUserTimeFormat } from '../../../shared/time';
import { ConfirmationComponent } from '../../../shared/ui';
import { RestoresComponent } from '../../restores';

interface Props {
  database: Database;
}

export const BackupsComponent = ({ database }: Props) => {
  const { t } = useTranslation(['backup', 'common']);
  const [isBackupsLoading, setIsBackupsLoading] = useState(false);
  const [backups, setBackups] = useState<Backup[]>([]);

  const [isBackupConfigLoading, setIsBackupConfigLoading] = useState(false);
  const [isShowBackupConfig, setIsShowBackupConfig] = useState(false);

  const [isMakeBackupRequestLoading, setIsMakeBackupRequestLoading] = useState(false);

  const [showingBackupError, setShowingBackupError] = useState<Backup | undefined>();

  const [deleteConfimationId, setDeleteConfimationId] = useState<string | undefined>();
  const [deletingBackupId, setDeletingBackupId] = useState<string | undefined>();

  const [showingRestoresBackupId, setShowingRestoresBackupId] = useState<string | undefined>();

  const isReloadInProgress = useRef(false);

  const [downloadingBackupId, setDownloadingBackupId] = useState<string | undefined>();

  const downloadBackup = async (backupId: string) => {
    try {
      const blob = await backupsApi.downloadBackup(backupId);

      // Create a download link
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;

      // Find the backup to get a meaningful filename
      const backup = backups.find((b) => b.id === backupId);
      const createdAt = backup ? dayjs(backup.createdAt).format('YYYY-MM-DD_HH-mm-ss') : 'backup';
      link.download = `${database.name}_backup_${createdAt}.dump`;

      // Trigger download
      document.body.appendChild(link);
      link.click();

      // Cleanup
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (e) {
      alert((e as Error).message);
    } finally {
      setDownloadingBackupId(undefined);
    }
  };

  const loadBackups = async () => {
    if (isReloadInProgress.current) {
      return;
    }

    isReloadInProgress.current = true;

    try {
      const backups = await backupsApi.getBackups(database.id);
      setBackups(backups);
    } catch (e) {
      alert((e as Error).message);
    }

    isReloadInProgress.current = false;
  };

  const makeBackup = async () => {
    setIsMakeBackupRequestLoading(true);

    try {
      await backupsApi.makeBackup(database.id);
      await loadBackups();
    } catch (e) {
      alert((e as Error).message);
    }

    setIsMakeBackupRequestLoading(false);
  };

  const deleteBackup = async () => {
    if (!deleteConfimationId) {
      return;
    }

    setDeleteConfimationId(undefined);
    setDeletingBackupId(deleteConfimationId);

    try {
      await backupsApi.deleteBackup(deleteConfimationId);
      await loadBackups();
    } catch (e) {
      alert((e as Error).message);
    }

    setDeletingBackupId(undefined);
    setDeleteConfimationId(undefined);
  };

  useEffect(() => {
    let isBackupsEnabled = false;

    setIsBackupConfigLoading(true);
    backupConfigApi.getBackupConfigByDbID(database.id).then((backupConfig) => {
      setIsBackupConfigLoading(false);

      if (backupConfig.isBackupsEnabled) {
        // load backups
        isBackupsEnabled = true;
        setIsShowBackupConfig(true);

        setIsBackupsLoading(true);
        loadBackups().then(() => setIsBackupsLoading(false));
      }
    });

    const interval = setInterval(() => {
      if (isBackupsEnabled) {
        loadBackups();
      }
    }, 1_000);

    return () => clearInterval(interval);
  }, [database]);

  useEffect(() => {
    if (downloadingBackupId) {
      downloadBackup(downloadingBackupId);
    }
  }, [downloadingBackupId]);

  const columns: ColumnsType<Backup> = [
    {
      title: t('backup:list.created_at'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (createdAt: string) => (
        <div>
          {dayjs.utc(createdAt).local().format(getUserTimeFormat().format)} <br />
          <span className="text-gray-500">({dayjs.utc(createdAt).local().fromNow()})</span>
        </div>
      ),
      sorter: (a, b) => dayjs(a.createdAt).unix() - dayjs(b.createdAt).unix(),
      defaultSortOrder: 'descend',
    },
    {
      title: t('backup:list.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status: BackupStatus, record: Backup) => {
        if (status === BackupStatus.FAILED) {
          return (
            <Tooltip title={t('backup:action.click_error_details')}>
              <div
                className="flex cursor-pointer items-center text-red-600 underline"
                onClick={() => setShowingBackupError(record)}
              >
                <ExclamationCircleOutlined className="mr-2" style={{ fontSize: 16 }} />

                <div>{t('backup:status.failed')}</div>
              </div>
            </Tooltip>
          );
        }

        if (status === BackupStatus.COMPLETED) {
          return (
            <div className="flex items-center text-green-600">
              <CheckCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
              <div>{t('backup:status.successful')}</div>
            </div>
          );
        }

        if (status === BackupStatus.DELETED) {
          return (
            <div className="flex items-center text-gray-600">
              <DeleteOutlined className="mr-2" style={{ fontSize: 16 }} />
              <div>{t('backup:status.deleted')}</div>
            </div>
          );
        }

        if (status === BackupStatus.IN_PROGRESS) {
          return (
            <div className="flex items-center font-bold text-blue-600">
              <SyncOutlined spin />
              <span className="ml-2">{t('backup:status.in_progress')}</span>
            </div>
          );
        }

        return <span className="font-bold">{status}</span>;
      },
      filters: [
        {
          value: BackupStatus.IN_PROGRESS,
          text: t('backup:status.in_progress'),
        },
        {
          value: BackupStatus.FAILED,
          text: t('backup:status.failed'),
        },
        {
          value: BackupStatus.COMPLETED,
          text: t('backup:status.successful'),
        },
        {
          value: BackupStatus.DELETED,
          text: t('backup:status.deleted'),
        },
      ],
      onFilter: (value, record) => record.status === value,
    },
    {
      title: (
        <div className="flex items-center">
          {t('backup:list.size')}
          <Tooltip
            className="ml-1"
            title={t('common:tooltip.backup_size')}
          >
            <InfoCircleOutlined />
          </Tooltip>
        </div>
      ),
      dataIndex: 'backupSizeMb',
      key: 'backupSizeMb',
      width: 150,
      render: (sizeMb: number) => {
        if (sizeMb >= 1024) {
          const sizeGb = sizeMb / 1024;
          return `${Number(sizeGb.toFixed(2)).toLocaleString()} GB`;
        }
        return `${Number(sizeMb?.toFixed(2)).toLocaleString()} MB`;
      },
    },
    {
      title: t('backup:list.duration'),
      dataIndex: 'backupDurationMs',
      key: 'backupDurationMs',
      width: 150,
      render: (durationMs: number) => {
        const hours = Math.floor(durationMs / 3600000);
        const minutes = Math.floor((durationMs % 3600000) / 60000);
        const seconds = Math.floor((durationMs % 60000) / 1000);

        if (hours > 0) {
          return `${hours}h ${minutes}m ${seconds}s`;
        }

        return `${minutes}m ${seconds}s`;
      },
    },
    {
      title: t('backup:list.actions'),
      dataIndex: '',
      key: '',
      render: (_, record: Backup) => {
        return (
          <div className="flex gap-2 text-lg">
            {record.status === BackupStatus.COMPLETED && (
              <div className="flex gap-2">
                {deletingBackupId === record.id ? (
                  <SyncOutlined spin />
                ) : (
                  <>
                    <Tooltip title={t('backup:action.delete')}>
                      <DeleteOutlined
                        className="cursor-pointer"
                        onClick={() => {
                          if (deletingBackupId) return;
                          setDeleteConfimationId(record.id);
                        }}
                        style={{ color: '#ff0000', opacity: deletingBackupId ? 0.2 : 1 }}
                      />
                    </Tooltip>

                    <Tooltip title={t('backup:action.restore')}>
                      <CloudUploadOutlined
                        className="cursor-pointer"
                        onClick={() => {
                          setShowingRestoresBackupId(record.id);
                        }}
                        style={{
                          color: '#0d6efd',
                        }}
                      />
                    </Tooltip>

                    <Tooltip title={t('backup:action.download')}>
                      {downloadingBackupId === record.id ? (
                        <SyncOutlined spin style={{ color: '#0d6efd' }} />
                      ) : (
                        <DownloadOutlined
                          className="cursor-pointer"
                          onClick={() => {
                            if (downloadingBackupId) return;
                            setDownloadingBackupId(record.id);
                          }}
                          style={{
                            opacity: downloadingBackupId ? 0.2 : 1,
                            color: '#0d6efd',
                          }}
                        />
                      )}
                    </Tooltip>
                  </>
                )}
              </div>
            )}
          </div>
        );
      },
    },
  ];

  if (isBackupConfigLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  if (!isShowBackupConfig) {
    return <div />;
  }

  return (
    <div className="mt-5 w-full rounded-md bg-white p-5 shadow">
      <h2 className="text-xl font-bold">{t('backup:list.title')}</h2>

      <div className="mt-5" />

      <div className="flex">
        <Button
          onClick={makeBackup}
          className="mr-1"
          type="primary"
          disabled={isMakeBackupRequestLoading}
          loading={isMakeBackupRequestLoading}
        >
          {t('backup:list.make_backup')}
        </Button>
      </div>

      <div className="mt-5 max-w-[850px]">
        <Table
          bordered
          columns={columns}
          dataSource={backups}
          rowKey="id"
          loading={isBackupsLoading}
          size="small"
          pagination={false}
        />
      </div>

      {deleteConfimationId && (
        <ConfirmationComponent
          onConfirm={deleteBackup}
          onDecline={() => setDeleteConfimationId(undefined)}
          description={t('backup:modal.delete_confirm')}
          actionButtonColor="red"
          actionText={t('common:button.delete')}
        />
      )}

      {showingRestoresBackupId && (
        <Modal
          width={400}
          open={!!showingRestoresBackupId}
          onCancel={() => setShowingRestoresBackupId(undefined)}
          title={t('backup:modal.restore_title')}
          footer={null}
        >
          <RestoresComponent
            database={database}
            backup={backups.find((b) => b.id === showingRestoresBackupId) as Backup}
          />
        </Modal>
      )}

      {showingBackupError && (
        <Modal
          title={t('backup:modal.error_title')}
          open={!!showingBackupError}
          onCancel={() => setShowingBackupError(undefined)}
          footer={null}
        >
          <div className="text-sm">{showingBackupError.failMessage}</div>
        </Modal>
      )}
    </div>
  );
};
