import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
  EyeOutlined,
  PauseCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { Spin, Table, Tag, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useEffect, useRef, useState } from 'react';

import type { Database } from '../../../../entity/databases';
import { verificationAgentApi } from '../../../../entity/verification/agents';
import {
  type RestoreVerification,
  VerificationStatus,
  VerificationTrigger,
  verificationRunsApi,
} from '../../../../entity/verification/runs';
import { getUserTimeFormat } from '../../../../shared/time';
import { VerificationDetailDrawer } from './VerificationDetailDrawer';

interface Props {
  database: Database;
  isCanManageDBs: boolean;
  isDirectlyUnderTab?: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
}

const AGENT_AVAILABILITY_REFRESH_MS = 5_000;
const VERIFICATIONS_REFRESH_MS = 1_000;
const VERIFICATIONS_PAGE_SIZE = 50;

const formatDurationMs = (durationMs?: number) => {
  if (durationMs === undefined || durationMs === null) {
    return '-';
  }

  const totalSeconds = Math.floor(durationMs / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}h ${minutes}m ${seconds}s`;
  }

  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }

  return `${seconds}s`;
};

const renderStatus = (status: VerificationStatus) => {
  if (status === VerificationStatus.COMPLETED) {
    return (
      <div className="flex items-center text-green-600">
        <CheckCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
        <span>Successful</span>
      </div>
    );
  }

  if (status === VerificationStatus.FAILED) {
    return (
      <div className="flex items-center text-red-600">
        <ExclamationCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
        <span>Failed</span>
      </div>
    );
  }

  if (status === VerificationStatus.RUNNING) {
    return (
      <div className="flex items-center font-bold text-blue-600">
        <SyncOutlined spin />
        <span className="ml-2">Running</span>
      </div>
    );
  }

  if (status === VerificationStatus.PENDING) {
    return (
      <div className="flex items-center text-gray-600">
        <ClockCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
        <span>Pending</span>
      </div>
    );
  }

  if (status === VerificationStatus.CANCELED) {
    return (
      <div className="flex items-center text-gray-500">
        <CloseCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
        <span>Canceled</span>
      </div>
    );
  }

  return (
    <div className="flex items-center">
      <PauseCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
      <span>{status}</span>
    </div>
  );
};

const renderTrigger = (trigger: VerificationTrigger) => {
  if (trigger === VerificationTrigger.MANUAL) {
    return <Tag color="blue">Manual</Tag>;
  }

  return <Tag color="purple">Scheduled</Tag>;
};

export const VerificationsComponent = ({
  database,
  isCanManageDBs,
  isDirectlyUnderTab,
  scrollContainerRef,
}: Props) => {
  const [verifications, setVerifications] = useState<RestoreVerification[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [selectedVerificationId, setSelectedVerificationId] = useState<string | undefined>();
  const [hasAgents, setHasAgents] = useState<boolean | undefined>();
  const [cancellingVerificationId, setCancellingVerificationId] = useState<string | undefined>();

  const [totalVerifications, setTotalVerifications] = useState(0);
  const [currentLimit, setCurrentLimit] = useState(VERIFICATIONS_PAGE_SIZE);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);

  const isVerificationsRequestInFlightRef = useRef(false);

  const loadVerifications = async (limit?: number) => {
    if (isVerificationsRequestInFlightRef.current) {
      return;
    }

    isVerificationsRequestInFlightRef.current = true;
    const loadLimit = limit ?? currentLimit;

    try {
      const response = await verificationRunsApi.listByDatabase(database.id, loadLimit, 0);
      setVerifications(response.verifications);
      setTotalVerifications(response.total);
      setHasMore(response.verifications.length < response.total);
    } catch (e) {
      alert((e as Error).message);
    } finally {
      isVerificationsRequestInFlightRef.current = false;
    }
  };

  const loadMoreVerifications = async () => {
    if (isLoadingMore || !hasMore) {
      return;
    }

    setIsLoadingMore(true);

    const newLimit = currentLimit + VERIFICATIONS_PAGE_SIZE;
    setCurrentLimit(newLimit);

    await loadVerifications(newLimit);

    setIsLoadingMore(false);
  };

  const cancelVerification = async (id: string) => {
    setCancellingVerificationId(id);

    try {
      await verificationRunsApi.cancel(id);
      await loadVerifications();
    } catch (e) {
      alert((e as Error).message);
    } finally {
      setCancellingVerificationId(undefined);
    }
  };

  const renderActions = (record: RestoreVerification) => {
    if (
      record.status === VerificationStatus.PENDING ||
      record.status === VerificationStatus.RUNNING
    ) {
      if (!isCanManageDBs) {
        return <span className="text-xs text-gray-400">-</span>;
      }

      const isCancelling = cancellingVerificationId === record.id;

      return (
        <div className="flex gap-2 text-lg">
          {isCancelling ? (
            <SyncOutlined spin />
          ) : (
            <Tooltip title="Cancel verification">
              <CloseCircleOutlined
                className="cursor-pointer"
                onClick={() => {
                  if (cancellingVerificationId) return;
                  cancelVerification(record.id);
                }}
                style={{ color: '#ff0000', opacity: cancellingVerificationId ? 0.2 : 1 }}
              />
            </Tooltip>
          )}
        </div>
      );
    }

    if (
      record.status === VerificationStatus.COMPLETED ||
      record.status === VerificationStatus.FAILED ||
      record.status === VerificationStatus.CANCELED
    ) {
      return (
        <div className="flex gap-2 text-lg">
          <Tooltip title="View details">
            <EyeOutlined
              className="cursor-pointer"
              onClick={() => setSelectedVerificationId(record.id)}
              style={{ color: '#155dfc' }}
            />
          </Tooltip>
        </div>
      );
    }

    return <span className="text-xs text-gray-400">-</span>;
  };

  useEffect(() => {
    setCurrentLimit(VERIFICATIONS_PAGE_SIZE);
    setHasMore(true);
    setIsLoading(true);
    loadVerifications(VERIFICATIONS_PAGE_SIZE).then(() => setIsLoading(false));
  }, [database.id]);

  useEffect(() => {
    const intervalId = setInterval(() => {
      loadVerifications();
    }, VERIFICATIONS_REFRESH_MS);

    return () => clearInterval(intervalId);
  }, [currentLimit]);

  useEffect(() => {
    let isCancelled = false;

    const loadAvailability = () => {
      verificationAgentApi
        .getAvailability()
        .then((availability) => {
          if (isCancelled) return;
          setHasAgents(availability.hasAgents);
        })
        .catch(() => {
          if (isCancelled) return;
          setHasAgents(undefined);
        });
    };

    loadAvailability();
    const intervalId = setInterval(loadAvailability, AGENT_AVAILABILITY_REFRESH_MS);

    return () => {
      isCancelled = true;
      clearInterval(intervalId);
    };
  }, []);

  useEffect(() => {
    if (!scrollContainerRef?.current) {
      return;
    }

    const handleScroll = () => {
      if (!scrollContainerRef.current) return;

      const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;

      if (scrollHeight - scrollTop <= clientHeight + 100 && hasMore && !isLoadingMore) {
        loadMoreVerifications();
      }
    };

    const container = scrollContainerRef.current;
    container.addEventListener('scroll', handleScroll);
    return () => container.removeEventListener('scroll', handleScroll);
  }, [hasMore, isLoadingMore, currentLimit, scrollContainerRef]);

  const columns: ColumnsType<RestoreVerification> = [
    {
      title: 'Created at',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 220,
      render: (createdAt: string) => (
        <div>
          {dayjs.utc(createdAt).local().format(getUserTimeFormat().format)} <br />
          <span className="text-gray-500 dark:text-gray-400">
            ({dayjs.utc(createdAt).local().fromNow()})
          </span>
        </div>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 160,
      render: (status: VerificationStatus) => renderStatus(status),
    },
    {
      title: 'Trigger',
      dataIndex: 'trigger',
      key: 'trigger',
      width: 110,
      render: (trigger: VerificationTrigger) => renderTrigger(trigger),
    },
    {
      title: (
        <div className="flex items-center">
          Duration
          <Tooltip className="ml-1" title="Restore + verify time reported by the agent.">
            <ExclamationCircleOutlined />
          </Tooltip>
        </div>
      ),
      key: 'duration',
      width: 130,
      render: (_, record: RestoreVerification) => {
        const restoreMs = record.restoreDurationMs ?? 0;
        const verifyMs = record.verifyDurationMs ?? 0;
        const total = restoreMs + verifyMs;

        if (total === 0) {
          return '-';
        }

        return (
          <div className="text-sm">
            <div>{formatDurationMs(total)}</div>
            <div className="text-xs text-gray-500 dark:text-gray-400">
              restore {formatDurationMs(restoreMs)}
            </div>
          </div>
        );
      },
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 120,
      render: (_, record: RestoreVerification) => renderActions(record),
    },
  ];

  return (
    <div
      className={`w-full bg-white p-3 shadow md:p-5 dark:bg-gray-800 ${isDirectlyUnderTab ? 'rounded-tr-md rounded-br-md rounded-bl-md' : 'rounded-md'}`}
    >
      <h2 className="text-lg font-bold md:text-xl dark:text-white">Restore verifications</h2>

      <p className="mt-2 max-w-2xl text-sm text-gray-500 dark:text-gray-400">
        Each row is one attempt to restore a backup of this database into a temporary copy
      </p>

      {hasAgents === false && (
        <div className="mt-3 flex max-w-2xl max-w-[400px] items-center gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-200">
          <ExclamationCircleOutlined className="shrink-0" />
          <span>
            No verification agents registered - please add it in Databasus settings tab in
            &quot;Verification agents&quot; section.{' '}
            <a
              href="https://databasus.com/restore-verification"
              target="_blank"
              rel="noopener noreferrer"
              className="underline hover:no-underline"
            >
              Learn more
            </a>
          </span>
        </div>
      )}

      <div className="mt-5 w-full md:max-w-[900px]">
        {/* Desktop table */}
        <div className="hidden md:block">
          <Table
            bordered
            columns={columns}
            dataSource={verifications}
            rowKey="id"
            loading={isLoading}
            size="small"
            pagination={false}
          />
          {!isLoading && verifications.length === 0 && (
            <div className="mt-3 text-center text-sm text-gray-500 dark:text-gray-400">
              No restore checks yet
            </div>
          )}
          {isLoadingMore && (
            <div className="mt-2 flex justify-center">
              <Spin />
            </div>
          )}
          {!hasMore && verifications.length > 0 && (
            <div className="mt-2 text-center text-gray-500 dark:text-gray-400">
              All restore checks loaded ({totalVerifications} total)
            </div>
          )}
        </div>

        {/* Mobile card list */}
        <div className="md:hidden">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <Spin />
            </div>
          ) : verifications.length === 0 ? (
            <div className="py-8 text-center text-gray-500 dark:text-gray-400">
              No restore checks yet
            </div>
          ) : (
            verifications.map((verification) => (
              <div
                key={verification.id}
                className="mb-2 rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800"
              >
                <div className="flex items-start justify-between">
                  <div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">Created at</div>
                    <div className="text-sm font-medium">
                      {dayjs.utc(verification.createdAt).local().format(getUserTimeFormat().format)}
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">
                      ({dayjs.utc(verification.createdAt).local().fromNow()})
                    </div>
                  </div>
                  <div>{renderStatus(verification.status)}</div>
                </div>

                <div className="mt-3 flex items-center gap-2">
                  {renderTrigger(verification.trigger)}
                  {verification.attemptCount > 1 && (
                    <span className="text-xs text-gray-500">
                      attempt {verification.attemptCount}
                    </span>
                  )}
                </div>

                {verification.failMessage && (
                  <div
                    className={`mt-2 text-xs ${
                      verification.status === VerificationStatus.CANCELED
                        ? 'text-gray-500'
                        : 'text-red-600'
                    }`}
                  >
                    {verification.failMessage}
                  </div>
                )}

                <div className="mt-3 flex items-center justify-end border-t border-gray-200 pt-3 dark:border-gray-700">
                  {renderActions(verification)}
                </div>
              </div>
            ))
          )}

          {isLoadingMore && (
            <div className="mt-3 flex justify-center">
              <Spin />
            </div>
          )}
          {!hasMore && verifications.length > 0 && (
            <div className="mt-3 text-center text-sm text-gray-500 dark:text-gray-400">
              All restore checks loaded ({totalVerifications} total)
            </div>
          )}
        </div>
      </div>

      {selectedVerificationId && (
        <VerificationDetailDrawer
          verificationId={selectedVerificationId}
          onClose={() => setSelectedVerificationId(undefined)}
        />
      )}
    </div>
  );
};
