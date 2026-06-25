import {
  CloseCircleOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  DownOutlined,
  FilterFilled,
  FilterOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { App, Button, Dropdown, type MenuProps, Modal, Spin, Table, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { type JSX, useEffect, useRef, useState } from 'react';

import {
  type PhysicalBackupConfig,
  type PhysicalBackupListItem,
  PhysicalBackupStatus,
  PhysicalBackupType,
  type PhysicalBackupsFilters,
  type TriggerPhysicalBackupType,
  physicalBackupConfigApi,
  physicalBackupsApi,
} from '../../../../entity/backups/physical';
import { type Database, PhysicalDatabaseBackupType } from '../../../../entity/databases';
import { usePersistentState } from '../../../../shared/hooks';
import { formatDuration } from '../../../../shared/lib';
import { getUserTimeFormat } from '../../../../shared/time';
import { ConfirmationComponent } from '../../../../shared/ui';
import {
  PHYSICAL_BACKUP_STATUS_BADGE_STYLES,
  PHYSICAL_BACKUP_STATUS_LABELS,
} from '../model/physicalBackupStatus';
import { PhysicalBackupsFiltersPanelComponent } from './PhysicalBackupsFiltersPanelComponent';
import { PhysicalRestoreComponent } from './PhysicalRestoreComponent';

const BACKUPS_PAGE_SIZE = 50;

interface Props {
  database: Database;
  isCanManageDBs: boolean;
  isDirectlyUnderTab?: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
}

const formatSize = (sizeMb: number): string => {
  if (sizeMb >= 1024) {
    const sizeGb = sizeMb / 1024;
    return `${Number(sizeGb.toFixed(2)).toLocaleString()} GB`;
  }

  return `${Number((sizeMb ?? 0).toFixed(2)).toLocaleString()} MB`;
};

const renderStatusBadge = (
  backup: PhysicalBackupListItem,
  onShowError: (backup: PhysicalBackupListItem) => void,
): JSX.Element => {
  const badgeStyle = PHYSICAL_BACKUP_STATUS_BADGE_STYLES[backup.status];
  const label = PHYSICAL_BACKUP_STATUS_LABELS[backup.status];

  const badge = (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ring-1 ring-inset ${badgeStyle.pillClasses}`}
    >
      {backup.status === PhysicalBackupStatus.IN_PROGRESS ? (
        <SyncOutlined spin style={{ fontSize: 10 }} />
      ) : (
        <span className={`h-1.5 w-1.5 rounded-full ${badgeStyle.dotClasses}`} />
      )}
      {label}
    </span>
  );

  if (!backup.failMessage) {
    return badge;
  }

  return (
    <Tooltip title="Click to see error details">
      <span className="cursor-pointer" onClick={() => onShowError(backup)}>
        {badge}
      </span>
    </Tooltip>
  );
};

const renderTypeBadge = (type: PhysicalBackupType): JSX.Element => {
  if (type === PhysicalBackupType.FULL) {
    return (
      <span className="inline-flex items-center rounded bg-indigo-500/15 px-2 py-0.5 text-xs font-semibold text-indigo-700 ring-1 ring-indigo-500/40 ring-inset dark:text-indigo-300">
        Full
      </span>
    );
  }

  if (type === PhysicalBackupType.WAL) {
    return (
      <span className="inline-flex items-center rounded bg-slate-400/10 px-2 py-0.5 text-xs font-medium text-slate-600 ring-1 ring-slate-400/30 ring-inset dark:text-slate-300">
        WAL
      </span>
    );
  }

  return (
    <span className="inline-flex items-center rounded bg-indigo-400/10 px-2 py-0.5 text-xs font-medium text-indigo-600 ring-1 ring-indigo-400/30 ring-inset dark:text-indigo-300">
      Incremental
    </span>
  );
};

export const PhysicalBackupsComponent = ({
  database,
  isCanManageDBs,
  isDirectlyUnderTab,
  scrollContainerRef,
}: Props): JSX.Element => {
  const { message } = App.useApp();

  const [isBackupsLoading, setIsBackupsLoading] = useState(false);
  const [backups, setBackups] = useState<PhysicalBackupListItem[]>([]);
  const [totalUsageMb, setTotalUsageMb] = useState(0);

  const [totalBackups, setTotalBackups] = useState(0);
  const [currentLimit, setCurrentLimit] = useState(BACKUPS_PAGE_SIZE);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);

  const [backupConfig, setBackupConfig] = useState<PhysicalBackupConfig | undefined>();
  const [isBackupConfigLoading, setIsBackupConfigLoading] = useState(false);

  const [isTriggeringBackup, setIsTriggeringBackup] = useState(false);
  const [cancellingBackupId, setCancellingBackupId] = useState<string | undefined>();
  const [deletingBackupId, setDeletingBackupId] = useState<string | undefined>();
  const [deleteConfirmationBackup, setDeleteConfirmationBackup] = useState<
    PhysicalBackupListItem | undefined
  >();

  const [restoringBackup, setRestoringBackup] = useState<PhysicalBackupListItem | undefined>();
  const [isPitrModalOpen, setIsPitrModalOpen] = useState(false);

  const [showingBackupError, setShowingBackupError] = useState<
    PhysicalBackupListItem | undefined
  >();

  const [isFilterPanelVisible, setIsFilterPanelVisible] = useState(false);
  const [filters, setFilters] = usePersistentState<PhysicalBackupsFilters>(
    'physicalBackupsFilters',
    {},
  );

  const lastRequestTimeRef = useRef<number>(0);
  const isBackupsRequestInFlightRef = useRef(false);

  const isIncrementalAllowed =
    database.postgresqlPhysical?.backupType !== PhysicalDatabaseBackupType.FULL;

  const loadBackups = async (limit?: number, filtersOverride?: PhysicalBackupsFilters) => {
    if (isBackupsRequestInFlightRef.current) return;
    isBackupsRequestInFlightRef.current = true;

    const requestTime = Date.now();
    lastRequestTimeRef.current = requestTime;

    const loadLimit = limit ?? currentLimit;
    const activeFilters = filtersOverride ?? filters;

    try {
      const response = await physicalBackupsApi.getPhysicalBackups(
        database.id,
        loadLimit,
        0,
        activeFilters,
      );

      if (lastRequestTimeRef.current !== requestTime) return;

      setBackups(response.backups);
      setTotalUsageMb(response.totalUsageMb);
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
    if (isLoadingMore || !hasMore) return;

    setIsLoadingMore(true);

    const newLimit = currentLimit + BACKUPS_PAGE_SIZE;
    setCurrentLimit(newLimit);

    const requestTime = Date.now();
    lastRequestTimeRef.current = requestTime;

    try {
      const response = await physicalBackupsApi.getPhysicalBackups(
        database.id,
        newLimit,
        0,
        filters,
      );

      if (lastRequestTimeRef.current !== requestTime) return;

      setBackups(response.backups);
      setTotalUsageMb(response.totalUsageMb);
      setTotalBackups(response.total);
      setHasMore(response.backups.length < response.total);
    } catch (e) {
      if (lastRequestTimeRef.current === requestTime) {
        alert((e as Error).message);
      }
    }

    setIsLoadingMore(false);
  };

  const triggerBackup = async (type: TriggerPhysicalBackupType) => {
    setIsTriggeringBackup(true);

    try {
      await physicalBackupsApi.triggerPhysicalBackup(database.id, type);
      await new Promise((resolve) => setTimeout(resolve, 1000));
      setCurrentLimit(BACKUPS_PAGE_SIZE);
      setHasMore(true);
      await loadBackups(BACKUPS_PAGE_SIZE);
    } catch (e) {
      message.error((e as Error).message);
    }

    setIsTriggeringBackup(false);
  };

  const cancelBackup = async (backupId: string) => {
    setCancellingBackupId(backupId);

    try {
      await physicalBackupsApi.cancelPhysicalBackup(backupId);
      await loadBackups();
    } catch (e) {
      alert((e as Error).message);
    }

    setCancellingBackupId(undefined);
  };

  const deleteBackup = async () => {
    if (!deleteConfirmationBackup) return;

    const backupId = deleteConfirmationBackup.id;
    setDeleteConfirmationBackup(undefined);
    setDeletingBackupId(backupId);

    try {
      await physicalBackupsApi.deletePhysicalBackup(backupId);
      setCurrentLimit(BACKUPS_PAGE_SIZE);
      setHasMore(true);
      await loadBackups(BACKUPS_PAGE_SIZE);
    } catch (e) {
      alert((e as Error).message);
    }

    setDeletingBackupId(undefined);
  };

  useEffect(() => {
    setIsBackupConfigLoading(true);
    setCurrentLimit(BACKUPS_PAGE_SIZE);
    setHasMore(true);

    physicalBackupConfigApi.getPhysicalBackupConfigByDbId(database.id).then((config) => {
      setBackupConfig(config);
      setIsBackupConfigLoading(false);

      setIsBackupsLoading(true);
      loadBackups(BACKUPS_PAGE_SIZE).then(() => setIsBackupsLoading(false));
    });
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
    if (!scrollContainerRef?.current) return;

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

  const renderActions = (record: PhysicalBackupListItem): JSX.Element => (
    <div className="flex gap-2 text-lg">
      {/* WAL segments have no standalone backup identity - they are not restored or deleted on their own (use point-in-time restore instead). */}
      {record.type === PhysicalBackupType.WAL ? null : (
        <>
          {record.status === PhysicalBackupStatus.IN_PROGRESS && isCanManageDBs && (
            <>
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
            </>
          )}

          {record.status === PhysicalBackupStatus.COMPLETED && (
            <>
              <Tooltip title="Restore from this backup">
                <CloudUploadOutlined
                  className="cursor-pointer"
                  onClick={() => setRestoringBackup(record)}
                  style={{ color: '#155dfc' }}
                />
              </Tooltip>

              {isCanManageDBs &&
                (deletingBackupId === record.id ? (
                  <SyncOutlined spin />
                ) : (
                  <Tooltip title="Delete backup">
                    <DeleteOutlined
                      className="cursor-pointer"
                      onClick={() => {
                        if (deletingBackupId) return;
                        setDeleteConfirmationBackup(record);
                      }}
                      style={{ color: '#ff0000', opacity: deletingBackupId ? 0.2 : 1 }}
                    />
                  </Tooltip>
                ))}
            </>
          )}
        </>
      )}
    </div>
  );

  const renderDuration = (record: PhysicalBackupListItem): string => {
    if (record.completedAt) {
      return formatDuration(dayjs(record.completedAt).diff(dayjs(record.createdAt)));
    }

    if (record.status === PhysicalBackupStatus.IN_PROGRESS) {
      // Live elapsed time; the 1s list refresh re-renders this and makes it tick.
      return formatDuration(dayjs().diff(dayjs(record.createdAt)));
    }

    return '-';
  };

  const columns: ColumnsType<PhysicalBackupListItem> = [
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      render: (type: PhysicalBackupType) => renderTypeBadge(type),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (_status: PhysicalBackupStatus, record: PhysicalBackupListItem) =>
        renderStatusBadge(record, setShowingBackupError),
    },
    {
      title: 'Size',
      dataIndex: 'sizeMb',
      key: 'sizeMb',
      width: 110,
      render: (sizeMb: number) => formatSize(sizeMb),
    },
    {
      title: 'Duration',
      key: 'duration',
      width: 150,
      render: (_, record: PhysicalBackupListItem) => renderDuration(record),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (createdAt: Date) => (
        <div>
          {dayjs.utc(createdAt).local().format(getUserTimeFormat().format)}
          <br />
          <span className="text-gray-500 dark:text-gray-400">
            ({dayjs.utc(createdAt).local().fromNow()})
          </span>
        </div>
      ),
      sorter: (a, b) => dayjs(a.createdAt).unix() - dayjs(b.createdAt).unix(),
      defaultSortOrder: 'descend',
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record: PhysicalBackupListItem) => renderActions(record),
    },
  ];

  const backupNowMenu: MenuProps = {
    items: [
      { key: 'full', label: 'Full backup' },
      {
        key: 'incremental',
        label: 'Incremental backup',
        disabled: !isIncrementalAllowed,
      },
    ],
    onClick: ({ key }) => triggerBackup(key as TriggerPhysicalBackupType),
  };

  const isAnyFilterApplied =
    (filters.types !== undefined && filters.types.length > 0) ||
    (filters.statuses !== undefined && filters.statuses.length > 0) ||
    filters.beforeDate !== undefined;

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
          <PhysicalBackupsFiltersPanelComponent filters={filters} onFiltersChange={setFilters} />
        </div>
      )}

      {!isBackupConfigLoading && !backupConfig?.isBackupsEnabled && (
        <div className="text-sm text-red-600">
          Scheduled backups are disabled (you can enable it back in the backup configuration)
        </div>
      )}

      <div className="mt-3 text-sm text-gray-600 dark:text-gray-400">
        Total usage: {formatSize(totalUsageMb)}
      </div>

      <div className="mt-4 flex items-center">
        <Dropdown.Button
          className="!w-auto"
          type="primary"
          loading={isTriggeringBackup}
          disabled={isTriggeringBackup}
          icon={<DownOutlined />}
          menu={backupNowMenu}
          onClick={() => triggerBackup('auto')}
        >
          Back up now
        </Dropdown.Button>

        <Button className="ml-2" onClick={() => setIsPitrModalOpen(true)}>
          Restore
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
                        <div className="text-xs text-gray-500 dark:text-gray-400">Created</div>
                        <div className="text-sm font-medium">
                          {dayjs.utc(backup.createdAt).local().format(getUserTimeFormat().format)}
                        </div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          ({dayjs.utc(backup.createdAt).local().fromNow()})
                        </div>
                      </div>
                      <div>{renderStatusBadge(backup, setShowingBackupError)}</div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">Type</div>
                        <div className="text-sm font-medium">{renderTypeBadge(backup.type)}</div>
                      </div>
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">Size</div>
                        <div className="text-sm font-medium">{formatSize(backup.sizeMb)}</div>
                      </div>
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">Duration</div>
                        <div className="text-sm font-medium">{renderDuration(backup)}</div>
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

      {deleteConfirmationBackup && (
        <ConfirmationComponent
          onConfirm={deleteBackup}
          onDecline={() => setDeleteConfirmationBackup(undefined)}
          description={
            deleteConfirmationBackup.type === PhysicalBackupType.FULL
              ? 'Deleting this full backup removes its entire chain (all incrementals that depend on it). This cannot be undone. Continue?'
              : 'Deleting this incremental backup removes all later incrementals that depend on it. This cannot be undone. Continue?'
          }
          actionButtonColor="red"
          actionText="Delete"
        />
      )}

      {restoringBackup && (
        <Modal
          width={640}
          open={!!restoringBackup}
          onCancel={() => setRestoringBackup(undefined)}
          title="Restore from backup"
          footer={null}
          maskClosable={false}
        >
          <PhysicalRestoreComponent
            database={database}
            backup={restoringBackup}
            onClose={() => setRestoringBackup(undefined)}
          />
        </Modal>
      )}

      {isPitrModalOpen && (
        <Modal
          width={640}
          open={isPitrModalOpen}
          onCancel={() => setIsPitrModalOpen(false)}
          title="Point-in-time restore"
          footer={null}
          maskClosable={false}
        >
          <PhysicalRestoreComponent database={database} onClose={() => setIsPitrModalOpen(false)} />
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
          <div className="text-sm break-words whitespace-pre-wrap">
            {showingBackupError.failMessage}
          </div>
        </Modal>
      )}
    </div>
  );
};
