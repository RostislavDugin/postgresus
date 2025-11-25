import { LoadingOutlined } from '@ant-design/icons';
import { App, Spin, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useCallback, useEffect, useRef, useState } from 'react';

import { auditLogApi } from '../../../entity/audit-logs/api/auditLogApi';
import type { AuditLog } from '../../../entity/audit-logs/model/AuditLog';
import type { GetAuditLogsRequest } from '../../../entity/audit-logs/model/GetAuditLogsRequest';
import { useIsMobile } from '../../../shared/hooks';
import { getUserTimeFormat } from '../../../shared/time';

interface Props {
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
}

export function AuditLogsComponent({ scrollContainerRef: externalScrollRef }: Props) {
  const { message } = App.useApp();
  const isMobile = useIsMobile();
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [total, setTotal] = useState(0);

  const pageSize = 50;

  const internalScrollRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = externalScrollRef || internalScrollRef;
  const loadingRef = useRef(false);

  useEffect(() => {
    loadAuditLogs(true);
  }, []);

  const handleScroll = useCallback(() => {
    if (!scrollContainerRef.current || isLoadingMore || !hasMore || loadingRef.current) return;

    const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
    const threshold = 100;

    if (scrollHeight - scrollTop - clientHeight < threshold) {
      loadAuditLogs(false);
    }
  }, [isLoadingMore, hasMore]);

  useEffect(() => {
    const scrollContainer = scrollContainerRef.current;
    if (scrollContainer) {
      scrollContainer.addEventListener('scroll', handleScroll);
      return () => scrollContainer.removeEventListener('scroll', handleScroll);
    }
  }, [handleScroll]);

  const loadAuditLogs = async (isInitialLoad = false) => {
    if (!isInitialLoad && loadingRef.current) {
      return;
    }

    loadingRef.current = true;

    if (isInitialLoad) {
      setIsLoading(true);
      setAuditLogs([]);
    } else {
      setIsLoadingMore(true);
    }

    try {
      const offset = isInitialLoad ? 0 : auditLogs.length;
      const request: GetAuditLogsRequest = {
        limit: pageSize,
        offset: offset,
      };

      const response = await auditLogApi.getGlobalAuditLogs(request);

      if (isInitialLoad) {
        setAuditLogs(response.auditLogs);
      } else {
        setAuditLogs((prev) => {
          const existingIds = new Set(prev.map((log) => log.id));
          const newLogs = response.auditLogs.filter((log) => !existingIds.has(log.id));
          return [...prev, ...newLogs];
        });
      }

      setTotal(response.total);
      setHasMore(response.auditLogs.length === pageSize);
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to load audit logs';
      message.error(errorMessage);
    } finally {
      loadingRef.current = false;
      setIsLoading(false);
      setIsLoadingMore(false);
    }
  };

  const columns: ColumnsType<AuditLog> = [
    {
      title: 'User',
      key: 'user',
      width: 300,
      render: (_, record: AuditLog) => {
        if (!record.userEmail && !record.userName) {
          return (
            <span className="inline-block rounded-full bg-gray-100 px-1.5 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-300">
              System
            </span>
          );
        }

        const displayText = record.userName
          ? `${record.userName} (${record.userEmail})`
          : record.userEmail;

        return (
          <span className="inline-block rounded-full bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900 dark:text-blue-200">
            {displayText}
          </span>
        );
      },
    },
    {
      title: 'Message',
      dataIndex: 'message',
      key: 'message',
      render: (message: string) => (
        <span className="text-xs text-gray-900 dark:text-gray-100">{message}</span>
      ),
    },
    {
      title: 'Workspace',
      dataIndex: 'workspaceName',
      key: 'workspaceName',
      width: 200,
      render: (workspaceId: string | undefined) => (
        <span
          className={`inline-block rounded-full px-1.5 py-0.5 text-xs font-medium ${
            workspaceId
              ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
              : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
          }`}
        >
          {workspaceId || '-'}
        </span>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 250,
      render: (createdAt: string) => {
        const date = dayjs(createdAt);
        const timeFormat = getUserTimeFormat();
        return (
          <span className="text-xs text-gray-700 dark:text-gray-300">
            {`${date.format(timeFormat.format)} (${date.fromNow()})`}
          </span>
        );
      },
    },
  ];

  const renderAuditLogCard = (log: AuditLog) => {
    const date = dayjs(log.createdAt);
    const timeFormat = getUserTimeFormat();

    const getUserDisplay = () => {
      if (!log.userEmail && !log.userName) {
        return (
          <span className="inline-block rounded-full bg-gray-100 px-1.5 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-300">
            System
          </span>
        );
      }

      const displayText = log.userName ? `${log.userName} (${log.userEmail})` : log.userEmail;

      return (
        <span className="inline-block rounded-full bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900 dark:text-blue-200">
          {displayText}
        </span>
      );
    };

    return (
      <div
        key={log.id}
        className="mb-3 rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-gray-700 dark:bg-gray-800"
      >
        <div className="flex items-start justify-between">
          <div className="flex-1">{getUserDisplay()}</div>
          <div className="text-right text-xs text-gray-500 dark:text-gray-400">
            <div>{date.format(timeFormat.format)}</div>
            <div className="text-gray-400 dark:text-gray-500">{date.fromNow()}</div>
          </div>
        </div>
        <div className="mt-2 text-sm text-gray-900 dark:text-gray-100">{log.message}</div>
        {log.workspaceName && (
          <div className="mt-2">
            <span className="inline-block rounded-full bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900 dark:text-blue-200">
              {log.workspaceName}
            </span>
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="max-w-[1200px]">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-bold dark:text-white">Audit Logs</h2>
        <div className="text-sm text-gray-500 dark:text-gray-400">
          {isLoading ? (
            <Spin indicator={<LoadingOutlined spin />} />
          ) : (
            `${auditLogs.length} of ${total} logs`
          )}
        </div>
      </div>

      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        </div>
      ) : auditLogs.length === 0 ? (
        <div className="flex h-32 items-center justify-center text-gray-500 dark:text-gray-400">
          No audit logs found.
        </div>
      ) : (
        <>
          {isMobile ? (
            <div>{auditLogs.map(renderAuditLogCard)}</div>
          ) : (
            <Table
              columns={columns}
              dataSource={auditLogs}
              pagination={false}
              rowKey="id"
              size="small"
              className="mb-4"
            />
          )}

          {isLoadingMore && (
            <div className="flex justify-center py-4">
              <Spin indicator={<LoadingOutlined spin />} />
              <span className="ml-2 text-sm text-gray-500 dark:text-gray-400">
                Loading more logs...
              </span>
            </div>
          )}

          {!hasMore && auditLogs.length > 0 && (
            <div className="py-4 text-center text-sm text-gray-500 dark:text-gray-400">
              All logs loaded ({auditLogs.length} total)
            </div>
          )}
        </>
      )}
    </div>
  );
}
