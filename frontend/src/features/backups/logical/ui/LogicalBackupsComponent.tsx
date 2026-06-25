import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  ExclamationCircleOutlined,
  FilterFilled,
  FilterOutlined,
  InfoCircleOutlined,
  LockOutlined,
  SafetyOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { App, Button, Modal, Spin, Table, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { type JSX, useEffect, useRef, useState } from 'react';

import {
  type BackupsFilters,
  type LogicalBackup,
  type LogicalBackupConfig,
  LogicalBackupStatus,
  RestoreVerificationStatus,
  logicalBackupConfigApi,
  logicalBackupsApi,
} from '../../../../entity/backups/logical';
import { BackupEncryption } from '../../../../entity/backups/shared';
import { type Database, DatabaseType } from '../../../../entity/databases';
import { verificationRunsApi } from '../../../../entity/verification/runs';
import { usePersistentState } from '../../../../shared/hooks';
import { formatDuration } from '../../../../shared/lib';
import { getUserTimeFormat } from '../../../../shared/time';
import { ConfirmationComponent } from '../../../../shared/ui';
import { RestoresComponent } from '../../../restores';
import {
  RESTORE_VERIFICATION_STATUS_BADGE_STYLES,
  RESTORE_VERIFICATION_STATUS_LABELS,
} from '../model/restoreVerificationStatus';
import { LogicalBackupsFiltersPanelComponent } from './LogicalBackupsFiltersPanelComponent';

const BACKUPS_PAGE_SIZE = 50;

interface Props {
  database: Database;
  isCanManageDBs: boolean;
  isDirectlyUnderTab?: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
  onNavigateToVerifications?: () => void;
}

const renderRestoreVerificationTag = (
  status: RestoreVerificationStatus | undefined,
): JSX.Element | null => {
  if (!status || status === RestoreVerificationStatus.NOT_VERIFIED) {
    return null;
  }

  const badgeStyle = RESTORE_VERIFICATION_STATUS_BADGE_STYLES[status];
  const label = RESTORE_VERIFICATION_STATUS_LABELS[status];
  if (!badgeStyle || !label) {
    return null;
  }

  return (
    <span
      className={`ml-3 inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ring-1 ring-inset ${badgeStyle.pillClasses}`}
    >
      <span className={`h-1.5 w-1.5 rounded-full ${badgeStyle.dotClasses}`} />
      {label}
    </span>
  );
};

export const LogicalBackupsComponent = ({
  database,
  isCanManageDBs,
  isDirectlyUnderTab,
  scrollContainerRef,
  onNavigateToVerifications,
}: Props) => {
  const { message, modal } = App.useApp();
  const [isBackupsLoading, setIsBackupsLoading] = useState(false);
  const [backups, setBackups] = useState<LogicalBackup[]>([]);

  const [totalBackups, setTotalBackups] = useState(0);
  const [currentLimit, setCurrentLimit] = useState(BACKUPS_PAGE_SIZE);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);

  const [backupConfig, setBackupConfig] = useState<LogicalBackupConfig | undefined>();
  const [isBackupConfigLoading, setIsBackupConfigLoading] = useState(false);

  const [isMakeBackupRequestLoading, setIsMakeBackupRequestLoading] = useState(false);

  const [showingBackupError, setShowingBackupError] = useState<LogicalBackup | undefined>();

  const [deleteConfimationId, setDeleteConfimationId] = useState<string | undefined>();
  const [deletingBackupId, setDeletingBackupId] = useState<string | undefined>();

  const [showingRestoresBackupId, setShowingRestoresBackupId] = useState<string | undefined>();

  const lastRequestTimeRef = useRef<number>(0);
  const isBackupsRequestInFlightRef = useRef(false);

  const [downloadingBackupId, setDownloadingBackupId] = useState<string | undefined>();
  const [cancellingBackupId, setCancellingBackupId] = useState<string | undefined>();
  const [verifyingBackupId, setVerifyingBackupId] = useState<string | undefined>();

  const [isFilterPanelVisible, setIsFilterPanelVisible] = useState(false);
  const [filters, setFilters] = usePersistentState<BackupsFilters>('logicalBackupsFilters', {});

  const downloadBackup = async (backupId: string) => {
    try {
      await logicalBackupsApi.downloadBackup(backupId);
    } catch (e) {
      alert((e as Error).message);
    } finally {
      setDownloadingBackupId(undefined);
    }
  };

  const loadBackups = async (limit?: number, filtersOverride?: BackupsFilters) => {
    if (isBackupsRequestInFlightRef.current) return;
    isBackupsRequestInFlightRef.current = true;

    const requestTime = Date.now();
    lastRequestTimeRef.current = requestTime;

    const loadLimit = limit ?? currentLimit;
    const activeFilters = filtersOverride ?? filters;

    try {
      const response = await logicalBackupsApi.getBackups(database.id, loadLimit, 0, activeFilters);

      if (lastRequestTimeRef.current !== requestTime) return;

      setBackups(response.backups);
      setTotalBackups(response.total);
      setHasMore(response.backups.length < response.total);
    } catch (e) {
      if (lastRequestTimeRef.current === requestTime) {
        alert((e as Error).message);
      }
    } finally {
      isBackupsRequestInFlightRef.current = false;
    }
  };

  const loadMoreBackups = async () => {
    if (isLoadingMore || !hasMore) {
      return;
    }

    setIsLoadingMore(true);

    const newLimit = currentLimit + BACKUPS_PAGE_SIZE;
    setCurrentLimit(newLimit);

    const requestTime = Date.now();
    lastRequestTimeRef.current = requestTime;

    try {
      const response = await logicalBackupsApi.getBackups(database.id, newLimit, 0, filters);

      if (lastRequestTimeRef.current !== requestTime) return;

      setBackups(response.backups);
      setTotalBackups(response.total);
      setHasMore(response.backups.length < response.total);
    } catch (e) {
      if (lastRequestTimeRef.current === requestTime) {
        alert((e as Error).message);
      }
    }

    setIsLoadingMore(false);
  };

  const makeBackup = async () => {
    setIsMakeBackupRequestLoading(true);

    try {
      await logicalBackupsApi.makeBackup(database.id);
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
      await logicalBackupsApi.deleteBackup(deleteConfimationId);
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
      await logicalBackupsApi.cancelBackup(backupId);
      await loadBackups();
    } catch (e) {
      alert((e as Error).message);
    }

    setCancellingBackupId(undefined);
  };

  const enqueueVerifyRestore = async (backupId: string) => {
    setVerifyingBackupId(backupId);

    try {
      await verificationRunsApi.enqueue(backupId);
      message.success('Restore check queued');
      onNavigateToVerifications?.();
    } catch (e) {
      message.error((e as Error).message);
    } finally {
      setVerifyingBackupId(undefined);
    }
  };

  const confirmVerifyRestore = (backupId: string) => {
    modal.confirm({
      title: 'Verify restore?',
      icon: <InfoCircleOutlined className="!text-blue-600" />,
      content: (
        <span>
          An agent will restore this backup to a temporary database and report row counts. This may
          take a while depending on backup size.{' '}
          <a
            href="https://databasus.com/restore-verification"
            target="_blank"
            rel="noopener noreferrer"
          >
            How it works?
          </a>
        </span>
      ),
      okText: 'Queue check',
      okType: 'primary',
      okButtonProps: { type: 'primary', danger: false },
      cancelText: 'Cancel',
      onOk: () => enqueueVerifyRestore(backupId),
    });
  };

  useEffect(() => {
    setIsBackupConfigLoading(true);
    setCurrentLimit(BACKUPS_PAGE_SIZE);
    setHasMore(true);

    logicalBackupConfigApi.getBackupConfigByDbID(database.id).then((config) => {
      setBackupConfig(config);
      setIsBackupConfigLoading(false);

      setIsBackupsLoading(true);
      loadBackups(BACKUPS_PAGE_SIZE).then(() => setIsBackupsLoading(false));
    });

    return () => {};
  }, [database]);

  useEffect(() => {
    setCurrentLimit(BACKUPS_PAGE_SIZE);
    setHasMore(true);
    setIsBackupsLoading(true);
    loadBackups(BACKUPS_PAGE_SIZE, filters).then(() => setIsBackupsLoading(false));
  }, [filters]);

  useEffect(() => {
    const intervalId = setInterval(() => {
      loadBackups();
    }, 1_000);

    return () => clearInterval(intervalId);
  }, [currentLimit, filters]);

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

  const renderBackupStatusLabel = (status: LogicalBackupStatus, record: LogicalBackup) => {
    if (status === LogicalBackupStatus.FAILED) {
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

    if (status === LogicalBackupStatus.COMPLETED) {
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

    if (status === LogicalBackupStatus.DELETED) {
      return (
        <div className="flex items-center text-gray-600">
          <DeleteOutlined className="mr-2" style={{ fontSize: 16 }} />
          <div>Deleted</div>
        </div>
      );
    }

    if (status === LogicalBackupStatus.IN_PROGRESS) {
      return (
        <div className="flex items-center font-bold text-blue-600">
          <SyncOutlined spin />
          <span className="ml-2">In progress</span>
        </div>
      );
    }

    if (status === LogicalBackupStatus.CANCELED) {
      return (
        <div className="flex items-center text-gray-600">
          <CloseCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
          <div>Canceled</div>
        </div>
      );
    }

    return <span className="font-bold">{status}</span>;
  };

  const renderStatus = (status: LogicalBackupStatus, record: LogicalBackup) => {
    const verificationTag = renderRestoreVerificationTag(record.restoreVerificationStatus);
    if (!verificationTag) {
      return renderBackupStatusLabel(status, record);
    }

    return (
      <div className="flex items-center">
        {renderBackupStatusLabel(status, record)}
        {verificationTag}
      </div>
    );
  };

  const renderActions = (record: LogicalBackup) => {
    return (
      <div className="flex gap-2 text-lg">
        {record.status === LogicalBackupStatus.IN_PROGRESS && isCanManageDBs && (
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

        {record.status === LogicalBackupStatus.COMPLETED && (
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

                {isCanManageDBs &&
                  database.type === DatabaseType.POSTGRES_LOGICAL &&
                  (verifyingBackupId === record.id ? (
                    <SyncOutlined spin style={{ color: '#155dfc' }} />
                  ) : (
                    <Tooltip title="Verify restore - queue an agent to restore this backup to a temporary database and report row counts">
                      <SafetyOutlined
                        className="cursor-pointer"
                        onClick={() => {
                          if (verifyingBackupId) return;
                          confirmVerifyRestore(record.id);
                        }}
                        style={{
                          color: '#155dfc',
                          opacity: verifyingBackupId ? 0.2 : 1,
                        }}
                      />
                    </Tooltip>
                  ))}

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
                    database.type === DatabaseType.POSTGRES_LOGICAL
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

  const columns: ColumnsType<LogicalBackup> = [
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
      render: (status: LogicalBackupStatus, record: LogicalBackup) => renderStatus(status, record),
    },
    {
      title: (
        <div className="flex items-center">
          Size
          <Tooltip
            className="ml-1"
            title="Top: the compressed backup file size we actually store in the storage (local, S3, Google Drive, etc.). Bottom (grayed): the original uncompressed database size at backup time. Backups are usually compressed ~5x."
          >
            <InfoCircleOutlined />
          </Tooltip>
        </div>
      ),
      dataIndex: 'backupSizeMb',
      key: 'backupSizeMb',
      width: 150,
      render: (sizeMb: number, record: LogicalBackup) => (
        <div>
          <div>{formatSize(sizeMb)}</div>
          {record.backupRawDbSizeMb > 0 && (
            <div className="text-xs text-gray-500 dark:text-gray-500">
              {formatSize(record.backupRawDbSizeMb)} (DB size)
            </div>
          )}
        </div>
      ),
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
      render: (_, record: LogicalBackup) => renderActions(record),
    },
  ];

  const isAnyFilterApplied =
    (filters.statuses && filters.statuses.length > 0) || filters.beforeDate !== undefined;

  if (isBackupConfigLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  return (
    <div
      className={`w-full bg-white p-3 shadow md:p-5 dark:bg-gray-800 ${isDirectlyUnderTab ? 'rounded-tr-md rounded-br-md rounded-bl-md' : 'rounded-md'}`}
    >
      <div className="flex items-center gap-2">
        <h2 className="text-lg font-bold md:text-xl dark:text-white">Backups</h2>
        <div className="relative">
          {isFilterPanelVisible ? (
            <FilterFilled
              className="cursor-pointer text-blue-600"
              onClick={() => setIsFilterPanelVisible(false)}
            />
          ) : (
            <FilterOutlined
              className="cursor-pointer"
              onClick={() => setIsFilterPanelVisible(true)}
            />
          )}
          {!isFilterPanelVisible && isAnyFilterApplied && (
            <span className="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-blue-600" />
          )}
        </div>
      </div>

      {isFilterPanelVisible && (
        <div className="mt-3">
          <LogicalBackupsFiltersPanelComponent filters={filters} onFiltersChange={setFilters} />
        </div>
      )}

      {!isBackupConfigLoading && !backupConfig?.isBackupsEnabled && (
        <div className="text-sm text-red-600">
          Scheduled backups are disabled (you can enable it back in the backup configuration)
        </div>
      )}

      <div className="mt-5" />

      <div className="flex items-center">
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
                        {backup.backupRawDbSizeMb > 0 && (
                          <div className="text-xs text-gray-500 dark:text-gray-500">
                            {formatSize(backup.backupRawDbSizeMb)} (DB size)
                          </div>
                        )}
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
            backup={backups.find((b) => b.id === showingRestoresBackupId) as LogicalBackup}
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
