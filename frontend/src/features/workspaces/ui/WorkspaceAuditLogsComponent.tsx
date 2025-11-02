import { LoadingOutlined } from '@ant-design/icons';
import { App, Spin, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useCallback, useEffect, useRef, useState } from 'react';

import type { AuditLog } from '../../../entity/audit-logs/model/AuditLog';
import { workspaceApi } from '../../../entity/workspaces/api/workspaceApi';
import { getUserShortTimeFormat } from '../../../shared/time';

interface Props {
  workspaceId: string;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
}

export function WorkspaceAuditLogsComponent({
  workspaceId,
  scrollContainerRef: externalScrollRef,
}: Props) {
  const { message } = App.useApp();
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [total, setTotal] = useState(0);

  const pageSize = 50;

  const internalScrollRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = externalScrollRef || internalScrollRef;
  const loadingRef = useRef(false);

  const handleScroll = useCallback(() => {
    if (!scrollContainerRef.current || isLoadingMore || !hasMore || loadingRef.current) return;

    const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
    const threshold = 100;

    if (scrollHeight - scrollTop - clientHeight < threshold) {
      loadAuditLogs(false);
    }
  }, [isLoadingMore, hasMore]);

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
      const params = {
        limit: pageSize,
        offset: offset,
      };

      const response = await workspaceApi.getWorkspaceAuditLogs(workspaceId, params);

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
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to load workspace audit logs';
      message.error(errorMessage);
    } finally {
      loadingRef.current = false;
      setIsLoading(false);
      setIsLoadingMore(false);
    }
  };

  useEffect(() => {
    if (workspaceId) {
      loadAuditLogs(true);
    }
  }, [workspaceId]);

  useEffect(() => {
    const scrollContainer = scrollContainerRef.current;
    if (scrollContainer) {
      scrollContainer.addEventListener('scroll', handleScroll);
      return () => scrollContainer.removeEventListener('scroll', handleScroll);
    }
  }, [handleScroll]);

  const columns: ColumnsType<AuditLog> = [
    {
      title: 'User',
      key: 'user',
      width: 300,
      render: (_, record: AuditLog) => {
        if (!record.userEmail && !record.userName) {
          return (
            <span className="inline-block rounded-full bg-gray-100 px-1.5 py-0.5 text-xs font-medium text-gray-600">
              System
            </span>
          );
        }

        const displayText = record.userName
          ? `${record.userName} (${record.userEmail})`
          : record.userEmail;

        return (
          <span className="inline-block rounded-full bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-800">
            {displayText}
          </span>
        );
      },
    },
    {
      title: 'Message',
      dataIndex: 'message',
      key: 'message',
      render: (message: string) => <span className="text-xs text-gray-900">{message}</span>,
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 250,
      render: (createdAt: string) => {
        const date = dayjs(createdAt);
        const timeFormat = getUserShortTimeFormat();
        return (
          <span className="text-xs text-gray-700">
            {`${date.format(timeFormat.format)} (${date.fromNow()})`}
          </span>
        );
      },
    },
  ];

  if (!workspaceId) {
    return null;
  }

  return (
    <div className="max-w-[1200px]">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-bold text-gray-900">Audit logs</h2>
        <div className="text-sm text-gray-500">
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
        <div className="flex h-32 items-center justify-center text-gray-500">
          No audit logs found for this workspace.
        </div>
      ) : (
        <>
          <Table
            columns={columns}
            dataSource={auditLogs}
            pagination={false}
            rowKey="id"
            size="small"
            className="mb-4"
          />

          {isLoadingMore && (
            <div className="flex justify-center py-4">
              <Spin indicator={<LoadingOutlined spin />} />
              <span className="ml-2 text-sm text-gray-500">Loading more logs...</span>
            </div>
          )}

          {!hasMore && auditLogs.length > 0 && (
            <div className="py-4 text-center text-sm text-gray-500">
              All logs loaded ({total} total)
            </div>
          )}
        </>
      )}
    </div>
  );
}
