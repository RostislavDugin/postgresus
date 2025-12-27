import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  ExclamationCircleOutlined,
  InfoCircleOutlined,
  LockOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { Button, Modal, Spin, Table, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useEffect, useRef, useState } from 'react';

import {
  type Backup,
  type BackupConfig,
  BackupEncryption,
  BackupStatus,
  backupConfigApi,
  backupsApi,
} from '../../../entity/backups';
import { type Database, DatabaseType } from '../../../entity/databases';
import { getUserTimeFormat } from '../../../shared/time';
import { ConfirmationComponent } from '../../../shared/ui';
import { RestoresComponent } from '../../restores';

const BACKUPS_PAGE_SIZE = 50;

interface Props {
  database: Database;
  isCanManageDBs: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
}

export const BackupsComponent = ({ database, isCanManageDBs, scrollContainerRef }: Props) => {
  const [isBackupsLoading, setIsBackupsLoading] = useState(false);
  const [backups, setBackups] = useState<Backup[]>([]);

  const [totalBackups, setTotalBackups] = useState(0);
  const [currentLimit, setCurrentLimit] = useState(BACKUPS_PAGE_SIZE);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);

  const [backupConfig, setBackupConfig] = useState<BackupConfig | undefined>();
  const [isBackupConfigLoading, setIsBackupConfigLoading] = useState(false);

  const [isMakeBackupRequestLoading, setIsMakeBackupRequestLoading] = useState(false);

  const [showingBackupError, setShowingBackupError] = useState<Backup | undefined>();

  const [deleteConfimationId, setDeleteConfimationId] = useState<string | undefined>();
  const [deletingBackupId, setDeletingBackupId] = useState<string | undefined>();

  const [showingRestoresBackupId, setShowingRestoresBackupId] = useState<string | undefined>();

  const isReloadInProgress = useRef(false);
  const isLazyLoadInProgress = useRef(false);

  const [downloadingBackupId, setDownloadingBackupId] = useState<string | undefined>();
  const [cancellingBackupId, setCancellingBackupId] = useState<string | undefined>();

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
      const extension = database.type === DatabaseType.MYSQL ? '.sql.zst' : '.dump.zst';
      link.download = `${database.name}_backup_${createdAt}${extension}`;

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

  const loadBackups = async (limit?: number) => {
    if (isReloadInProgress.current || isLazyLoadInProgress.current) {
      return;
    }

    isReloadInProgress.current = true;

    try {
      const loadLimit = limit || currentLimit;
      const response = await backupsApi.getBackups(database.id, loadLimit, 0);

      setBackups(response.backups);
      setTotalBackups(response.total);
      setHasMore(response.backups.length < response.total);
    } catch (e) {
      alert((e as Error).message);
    }

    isReloadInProgress.current = false;
  };

  const reloadInProgressBackups = async () => {
    if (isReloadInProgress.current || isLazyLoadInProgress.current) {
      return;
    }

    isReloadInProgress.current = true;

    try {
      // Fetch only the recent backups that could be in progress
      // We fetch a small number (20) to capture recent backups that might be in progress
      const response = await backupsApi.getBackups(database.id, 20, 0);

      // Update only the backups that exist in both lists
      setBackups((prevBackups) => {
        const updatedBackups = [...prevBackups];

        response.backups.forEach((newBackup) => {
          const index = updatedBackups.findIndex((b) => b.id === newBackup.id);
          if (index !== -1) {
            updatedBackups[index] = newBackup;
          } else if (index === -1 && updatedBackups.length < currentLimit) {
            // New backup that doesn't exist yet (e.g., just created)
            updatedBackups.unshift(newBackup);
          }
        });

        return updatedBackups;
      });

      setTotalBackups(response.total);
    } catch (e) {
      alert((e as Error).message);
    }

    isReloadInProgress.current = false;
  };

  const loadMoreBackups = async () => {
    if (isLoadingMore || !hasMore || isLazyLoadInProgress.current) {
      return;
    }

    isLazyLoadInProgress.current = true;
    setIsLoadingMore(true);

    try {
      const newLimit = currentLimit + BACKUPS_PAGE_SIZE;
      const response = await backupsApi.getBackups(database.id, newLimit, 0);

      setBackups(response.backups);
      setCurrentLimit(newLimit);
      setTotalBackups(response.total);
      setHasMore(response.backups.length < response.total);
    } catch (e) {
      alert((e as Error).message);
    }

    setIsLoadingMore(false);
    isLazyLoadInProgress.current = false;
  };

  const makeBackup = async () => {
    setIsMakeBackupRequestLoading(true);

    try {
      await backupsApi.makeBackup(database.id);
      await new Promise((resolve) => setTimeout(resolve, 1000));
      setCurrentLimit(BACKUPS_PAGE_SIZE);
      setHasMore(true);
      await loadBackups(BACKUPS_PAGE_SIZE);
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
      setCurrentLimit(BACKUPS_PAGE_SIZE);
      setHasMore(true);
      await loadBackups(BACKUPS_PAGE_SIZE);
    } catch (e) {
      alert((e as Error).message);
    }

    setDeletingBackupId(undefined);
    setDeleteConfimationId(undefined);
  };

  const cancelBackup = async (backupId: string) => {
    setCancellingBackupId(backupId);

    try {
      await backupsApi.cancelBackup(backupId);
      await reloadInProgressBackups();
    } catch (e) {
      alert((e as Error).message);
    }

    setCancellingBackupId(undefined);
  };

  useEffect(() => {
    setIsBackupConfigLoading(true);
    setCurrentLimit(BACKUPS_PAGE_SIZE);
    setHasMore(true);

    backupConfigApi.getBackupConfigByDbID(database.id).then((config) => {
      setBackupConfig(config);
      setIsBackupConfigLoading(false);

      setIsBackupsLoading(true);
      loadBackups(BACKUPS_PAGE_SIZE).then(() => setIsBackupsLoading(false));
    });

    return () => {};
  }, [database]);

  // Reload backups that are in progress to update their state
  useEffect(() => {
    const hasInProgressBackups = backups.some(
      (backup) => backup.status === BackupStatus.IN_PROGRESS,
    );

    if (!hasInProgressBackups) {
      return;
    }

    const timeoutId = setTimeout(async () => {
      await reloadInProgressBackups();
    }, 1_000);

    return () => clearTimeout(timeoutId);
  }, [backups]);

  useEffect(() => {
    if (downloadingBackupId) {
      downloadBackup(downloadingBackupId);
    }
  }, [downloadingBackupId]);

  useEffect(() => {
    if (!scrollContainerRef?.current) {
      return;
    }

    const handleScroll = () => {
      if (!scrollContainerRef.current) return;

      const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;

      if (scrollHeight - scrollTop <= clientHeight + 100 && hasMore && !isLoadingMore) {
        loadMoreBackups();
      }
    };

    const container = scrollContainerRef.current;
    container.addEventListener('scroll', handleScroll);
    return () => container.removeEventListener('scroll', handleScroll);
  }, [hasMore, isLoadingMore, currentLimit, scrollContainerRef]);

  const renderStatus = (status: BackupStatus, record: Backup) => {
    if (status === BackupStatus.FAILED) {
      return (
        <Tooltip title="Click to see error details">
          <div
            className="flex cursor-pointer items-center text-red-600 underline"
            onClick={() => setShowingBackupError(record)}
          >
            <ExclamationCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
            <div>Failed</div>
          </div>
        </Tooltip>
      );
    }

    if (status === BackupStatus.COMPLETED) {
      return (
        <div className="flex items-center text-green-600">
          <CheckCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
          <div>Successful</div>
          {record.encryption === BackupEncryption.ENCRYPTED && (
            <Tooltip title="Encrypted">
              <LockOutlined className="ml-1" style={{ fontSize: 14 }} />
            </Tooltip>
          )}
        </div>
      );
    }

    if (status === BackupStatus.DELETED) {
      return (
        <div className="flex items-center text-gray-600">
          <DeleteOutlined className="mr-2" style={{ fontSize: 16 }} />
          <div>Deleted</div>
        </div>
      );
    }

    if (status === BackupStatus.IN_PROGRESS) {
      return (
        <div className="flex items-center font-bold text-blue-600">
          <SyncOutlined spin />
          <span className="ml-2">In progress</span>
        </div>
      );
    }

    if (status === BackupStatus.CANCELED) {
      return (
        <div className="flex items-center text-gray-600">
          <CloseCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
          <div>Canceled</div>
        </div>
      );
    }

    return <span className="font-bold">{status}</span>;
  };

  const renderActions = (record: Backup) => {
    return (
      <div className="flex gap-2 text-lg">
        {record.status === BackupStatus.IN_PROGRESS && isCanManageDBs && (
          <div className="flex gap-2">
            {cancellingBackupId === record.id ? (
              <SyncOutlined spin />
            ) : (
              <Tooltip title="Cancel backup">
                <CloseCircleOutlined
                  className="cursor-pointer"
                  onClick={() => {
                    if (cancellingBackupId) return;
                    cancelBackup(record.id);
                  }}
                  style={{ color: '#ff0000', opacity: cancellingBackupId ? 0.2 : 1 }}
                />
              </Tooltip>
            )}
          </div>
        )}

        {record.status === BackupStatus.COMPLETED && (
          <div className="flex gap-2">
            {deletingBackupId === record.id ? (
              <SyncOutlined spin />
            ) : (
              <>
                {isCanManageDBs && (
                  <Tooltip title="Delete backup">
                    <DeleteOutlined
                      className="cursor-pointer"
                      onClick={() => {
                        if (deletingBackupId) return;
                        setDeleteConfimationId(record.id);
                      }}
                      style={{ color: '#ff0000', opacity: deletingBackupId ? 0.2 : 1 }}
                    />
                  </Tooltip>
                )}

                <Tooltip title="Restore from backup">
                  <CloudUploadOutlined
                    className="cursor-pointer"
                    onClick={() => {
                      setShowingRestoresBackupId(record.id);
                    }}
                    style={{
                      color: '#155dfc',
                    }}
                  />
                </Tooltip>

                <Tooltip
                  title={
                    database.type === DatabaseType.POSTGRES
                      ? 'Download backup file. It can be restored manually via pg_restore (from custom format)'
                      : database.type === DatabaseType.MYSQL
                        ? 'Download backup file. It can be restored manually via mysql client (from SQL dump)'
                        : database.type === DatabaseType.MARIADB
                          ? 'Download backup file. It can be restored manually via mariadb client (from SQL dump)'
                          : database.type === DatabaseType.MONGODB
                            ? 'Download backup file. It can be restored manually via mongorestore (from archive)'
                            : 'Download backup file'
                  }
                >
                  {downloadingBackupId === record.id ? (
                    <SyncOutlined spin style={{ color: '#155dfc' }} />
                  ) : (
                    <DownloadOutlined
                      className="cursor-pointer"
                      onClick={() => {
                        if (downloadingBackupId) return;
                        setDownloadingBackupId(record.id);
                      }}
                      style={{
                        opacity: downloadingBackupId ? 0.2 : 1,
                        color: '#155dfc',
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
  };

  const formatSize = (sizeMb: number) => {
    if (sizeMb >= 1024) {
      const sizeGb = sizeMb / 1024;
      return `${Number(sizeGb.toFixed(2)).toLocaleString()} GB`;
    }
    return `${Number(sizeMb?.toFixed(2)).toLocaleString()} MB`;
  };

  const formatDuration = (durationMs: number) => {
    const hours = Math.floor(durationMs / 3600000);
    const minutes = Math.floor((durationMs % 3600000) / 60000);
    const seconds = Math.floor((durationMs % 60000) / 1000);

    if (hours > 0) {
      return `${hours}h ${minutes}m ${seconds}s`;
    }

    return `${minutes}m ${seconds}s`;
  };

  const columns: ColumnsType<Backup> = [
    {
      title: 'Created at',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (createdAt: string) => (
        <div>
          {dayjs.utc(createdAt).local().format(getUserTimeFormat().format)} <br />
          <span className="text-gray-500 dark:text-gray-400">
            ({dayjs.utc(createdAt).local().fromNow()})
          </span>
        </div>
      ),
      sorter: (a, b) => dayjs(a.createdAt).unix() - dayjs(b.createdAt).unix(),
      defaultSortOrder: 'descend',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: BackupStatus, record: Backup) => renderStatus(status, record),
      filters: [
        {
          value: BackupStatus.IN_PROGRESS,
          text: 'In progress',
        },
        {
          value: BackupStatus.FAILED,
          text: 'Failed',
        },
        {
          value: BackupStatus.COMPLETED,
          text: 'Successful',
        },
        {
          value: BackupStatus.DELETED,
          text: 'Deleted',
        },
        {
          value: BackupStatus.CANCELED,
          text: 'Canceled',
        },
      ],
      onFilter: (value, record) => record.status === value,
    },
    {
      title: (
        <div className="flex items-center">
          Size
          <Tooltip
            className="ml-1"
            title="The file size we actually store in the storage (local, S3, Google Drive, etc.), usually compressed in ~5x times"
          >
            <InfoCircleOutlined />
          </Tooltip>
        </div>
      ),
      dataIndex: 'backupSizeMb',
      key: 'backupSizeMb',
      width: 150,
      render: (sizeMb: number) => formatSize(sizeMb),
    },
    {
      title: 'Duration',
      dataIndex: 'backupDurationMs',
      key: 'backupDurationMs',
      width: 150,
      render: (durationMs: number) => formatDuration(durationMs),
    },
    {
      title: 'Actions',
      dataIndex: '',
      key: '',
      render: (_, record: Backup) => renderActions(record),
    },
  ];

  if (isBackupConfigLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  return (
    <div className="mt-5 w-full rounded-md bg-white p-3 shadow md:p-5 dark:bg-gray-800">
      <h2 className="text-lg font-bold md:text-xl dark:text-white">Backups</h2>

      {!isBackupConfigLoading && !backupConfig?.isBackupsEnabled && (
        <div className="text-sm text-red-600">
          Scheduled backups are disabled (you can enable it back in the backup configuration)
        </div>
      )}

      <div className="mt-5" />

      <div className="flex">
        <Button
          onClick={makeBackup}
          className="mr-1"
          type="primary"
          disabled={isMakeBackupRequestLoading}
          loading={isMakeBackupRequestLoading}
        >
          <span className="md:hidden">Backup now</span>
          <span className="hidden md:inline">Make backup right now</span>
        </Button>
      </div>

      <div className="mt-5 w-full md:max-w-[850px]">
        {/* Mobile card view */}
        <div className="md:hidden">
          {isBackupsLoading ? (
            <div className="flex justify-center py-8">
              <Spin />
            </div>
          ) : (
            <div>
              {backups.map((backup) => (
                <div
                  key={backup.id}
                  className="mb-2 rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800"
                >
                  <div className="space-y-3">
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">Created at</div>
                        <div className="text-sm font-medium">
                          {dayjs.utc(backup.createdAt).local().format(getUserTimeFormat().format)}
                        </div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          ({dayjs.utc(backup.createdAt).local().fromNow()})
                        </div>
                      </div>
                      <div>{renderStatus(backup.status, backup)}</div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">Size</div>
                        <div className="text-sm font-medium">{formatSize(backup.backupSizeMb)}</div>
                      </div>
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">Duration</div>
                        <div className="text-sm font-medium">
                          {formatDuration(backup.backupDurationMs)}
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center justify-end border-t border-gray-200 pt-3">
                      {renderActions(backup)}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {isLoadingMore && (
            <div className="mt-3 flex justify-center">
              <Spin />
            </div>
          )}
          {!hasMore && backups.length > 0 && (
            <div className="mt-3 text-center text-sm text-gray-500 dark:text-gray-400">
              All backups loaded ({totalBackups} total)
            </div>
          )}
          {!isBackupsLoading && backups.length === 0 && (
            <div className="py-8 text-center text-gray-500 dark:text-gray-400">No backups yet</div>
          )}
        </div>

        {/* Desktop table view */}
        <div className="hidden md:block">
          <Table
            bordered
            columns={columns}
            dataSource={backups}
            rowKey="id"
            loading={isBackupsLoading}
            size="small"
            pagination={false}
          />
          {isLoadingMore && (
            <div className="mt-2 flex justify-center">
              <Spin />
            </div>
          )}
          {!hasMore && backups.length > 0 && (
            <div className="mt-2 text-center text-gray-500 dark:text-gray-400">
              All backups loaded ({totalBackups} total)
            </div>
          )}
        </div>
      </div>

      {deleteConfimationId && (
        <ConfirmationComponent
          onConfirm={deleteBackup}
          onDecline={() => setDeleteConfimationId(undefined)}
          description="Are you sure you want to delete this backup?"
          actionButtonColor="red"
          actionText="Delete"
        />
      )}

      {showingRestoresBackupId && (
        <Modal
          width={400}
          open={!!showingRestoresBackupId}
          onCancel={() => setShowingRestoresBackupId(undefined)}
          title="Restore from backup"
          footer={null}
          maskClosable={false}
        >
          <RestoresComponent
            database={database}
            backup={backups.find((b) => b.id === showingRestoresBackupId) as Backup}
          />
        </Modal>
      )}

      {showingBackupError && (
        <Modal
          title="Backup error details"
          open={!!showingBackupError}
          onCancel={() => setShowingBackupError(undefined)}
          maskClosable={false}
          footer={null}
        >
          <div className="text-sm">{showingBackupError.failMessage}</div>
        </Modal>
      )}
    </div>
  );
};
