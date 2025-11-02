import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Drawer, Input, Select, Spin, Switch, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useCallback, useEffect, useRef, useState } from 'react';

import { userManagementApi } from '../../../entity/users/api/userManagementApi';
import type { ChangeUserRoleRequest } from '../../../entity/users/model/ChangeUserRoleRequest';
import type { ListUsersRequest } from '../../../entity/users/model/ListUsersRequest';
import type { UserProfile } from '../../../entity/users/model/UserProfile';
import { UserRole } from '../../../entity/users/model/UserRole';
import { getUserTimeFormat } from '../../../shared/time';
import { UserAuditLogsSidebarComponent } from './UserAuditLogsSidebarComponent';

interface Props {
  contentHeight: number;
}

const getRoleColor = (role: UserRole): string => {
  switch (role) {
    case UserRole.ADMIN:
      return '#3b82f6';
    case UserRole.MEMBER:
      return '#10b981';
    default:
      return '#6b7280';
  }
};

export function UsersComponent({ contentHeight }: Props) {
  const { message } = App.useApp();
  const [users, setUsers] = useState<UserProfile[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [total, setTotal] = useState(0);
  const [searchQuery, setSearchQuery] = useState('');
  const [inputValue, setInputValue] = useState('');

  const pageSize = 20;

  const [processingUsers, setProcessingUsers] = useState<Set<string>>(new Set());
  const [changingRoleUsers, setChangingRoleUsers] = useState<Set<string>>(new Set());

  const [selectedUser, setSelectedUser] = useState<UserProfile | null>(null);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const loadingRef = useRef(false);

  useEffect(() => {
    loadUsers(true);
  }, []);

  useEffect(() => {
    const timer = setTimeout(() => {
      if (inputValue !== searchQuery) {
        setSearchQuery(inputValue);
        setHasMore(true);
        loadUsers(true, inputValue);
      }
    }, 500);

    return () => clearTimeout(timer);
  }, [inputValue]);

  const handleScroll = useCallback(() => {
    if (!scrollContainerRef.current || isLoadingMore || !hasMore || loadingRef.current) return;

    const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
    const threshold = 100;

    if (scrollHeight - scrollTop - clientHeight < threshold) {
      loadUsers(false);
    }
  }, [isLoadingMore, hasMore]);

  useEffect(() => {
    const scrollContainer = scrollContainerRef.current;
    if (scrollContainer) {
      scrollContainer.addEventListener('scroll', handleScroll);
      return () => scrollContainer.removeEventListener('scroll', handleScroll);
    }
  }, [handleScroll]);

  const loadUsers = async (isInitialLoad = false, query?: string) => {
    if (!isInitialLoad && loadingRef.current) {
      return;
    }

    loadingRef.current = true;

    if (isInitialLoad) {
      setIsLoading(true);
      setUsers([]);
    } else {
      setIsLoadingMore(true);
    }

    try {
      const offset = isInitialLoad ? 0 : users.length;
      const currentQuery = query !== undefined ? query : searchQuery;
      const request: ListUsersRequest = {
        limit: pageSize,
        offset: offset,
        query: currentQuery || undefined,
      };

      const response = await userManagementApi.getUsers(request);

      if (isInitialLoad) {
        setUsers(response.users);
      } else {
        setUsers((prev) => {
          const existingIds = new Set(prev.map((user) => user.id));
          const newUsers = response.users.filter((user) => !existingIds.has(user.id));
          return [...prev, ...newUsers];
        });
      }

      setTotal(response.total);
      setHasMore(response.users.length === pageSize);
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to load users';
      message.error(errorMessage);
    } finally {
      loadingRef.current = false;
      setIsLoading(false);
      setIsLoadingMore(false);
    }
  };

  const handleActivationToggle = async (userId: string, isActive: boolean) => {
    setUsers((prev) =>
      prev.map((user) => (user.id === userId ? { ...user, isActive: !isActive } : user)),
    );

    setProcessingUsers((prev) => new Set(prev).add(userId));

    try {
      if (isActive) {
        await userManagementApi.deactivateUser(userId);
        message.success('User deactivated successfully');
      } else {
        await userManagementApi.activateUser(userId);
        message.success('User activated successfully');
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Operation failed';
      message.error(errorMessage);

      setUsers((prev) =>
        prev.map((user) => (user.id === userId ? { ...user, isActive: isActive } : user)),
      );
    } finally {
      setProcessingUsers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(userId);
        return newSet;
      });
    }
  };

  const handleRoleChange = async (userId: string, newRole: UserRole) => {
    const currentUser = users.find((user) => user.id === userId);
    const originalRole = currentUser?.role;

    setUsers((prev) =>
      prev.map((user) => (user.id === userId ? { ...user, role: newRole } : user)),
    );

    setChangingRoleUsers((prev) => new Set(prev).add(userId));

    try {
      const request: ChangeUserRoleRequest = { role: newRole };
      await userManagementApi.changeUserRole(userId, request);
      message.success('User role changed successfully');
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to change user role';
      message.error(errorMessage);

      if (originalRole) {
        setUsers((prev) =>
          prev.map((user) => (user.id === userId ? { ...user, role: originalRole } : user)),
        );
      }
    } finally {
      setChangingRoleUsers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(userId);
        return newSet;
      });
    }
  };

  const handleRowClick = (user: UserProfile) => {
    setSelectedUser(user);
    setIsDrawerOpen(true);
  };

  const handleDrawerClose = () => {
    setIsDrawerOpen(false);
    setSelectedUser(null);
  };

  const columns: ColumnsType<UserProfile> = [
    {
      title: 'User',
      key: 'user',
      width: 350,
      render: (_, record: UserProfile) => (
        <div>
          {record.name} ({record.email})
        </div>
      ),
    },
    {
      title: 'System role',
      dataIndex: 'role',
      key: 'role',
      width: 200,
      render: (role: UserRole, record: UserProfile) => (
        <Select
          value={role}
          onChange={(value) => handleRoleChange(record.id, value)}
          loading={changingRoleUsers.has(record.id)}
          disabled={changingRoleUsers.has(record.id)}
          size="small"
          className="w-24"
          style={{
            color: getRoleColor(role),
          }}
          options={[
            {
              label: <span style={{ color: getRoleColor(UserRole.ADMIN) }}>Admin</span>,
              value: UserRole.ADMIN,
            },
            {
              label: <span style={{ color: getRoleColor(UserRole.MEMBER) }}>Member</span>,
              value: UserRole.MEMBER,
            },
          ]}
        />
      ),
    },
    {
      title: 'Is active?',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 200,
      render: (isActive: boolean, record: UserProfile) => (
        <Switch
          checked={isActive}
          onChange={() => handleActivationToggle(record.id, isActive)}
          loading={processingUsers.has(record.id)}
          disabled={processingUsers.has(record.id)}
          size="small"
          style={{
            backgroundColor: isActive ? '#155dfc' : undefined,
          }}
        />
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 300,
      render: (createdAt: string) => {
        const date = dayjs(createdAt);
        const timeFormat = getUserTimeFormat();
        return (
          <div className="text-sm text-gray-600">
            <div>{date.format(timeFormat.format)}</div>
            <div className="text-xs text-gray-400">{date.fromNow()}</div>
          </div>
        );
      },
    },
    {
      title: '',
      key: 'empty',
      render: (_, record: UserProfile) => (
        <div>
          <Button type="primary" ghost size="small" onClick={() => handleRowClick(record)}>
            View audit logs
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div className="flex grow pl-3">
      <div className="w-full">
        <div
          ref={scrollContainerRef}
          className="grow overflow-y-auto rounded bg-white p-5 shadow"
          style={{ height: contentHeight }}
        >
          <div className="mb-4 flex items-center justify-between">
            <h1 className="text-2xl font-bold">Postgresus Users</h1>
            <div className="text-sm text-gray-500">
              {isLoading ? 'Loading...' : `${users.length} of ${total} users`}
            </div>
          </div>

          <div className="mb-4">
            <Input
              placeholder="Search by email or name..."
              allowClear
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              style={{ width: 400 }}
            />
          </div>

          {isLoading ? (
            <div className="flex h-64 items-center justify-center">
              <Spin indicator={<LoadingOutlined spin />} size="large" />
            </div>
          ) : (
            <>
              <Table
                columns={columns}
                dataSource={users}
                pagination={false}
                rowKey="id"
                size="small"
                className="mb-4"
              />

              {isLoadingMore && (
                <div className="flex justify-center py-4">
                  <Spin indicator={<LoadingOutlined spin />} />
                </div>
              )}

              {!hasMore && users.length > 0 && (
                <div className="py-4 text-center text-sm text-gray-500">
                  All users loaded ({total} total)
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Audit logs drawer */}
      <Drawer
        title={
          <div>
            <div className="text-lg font-semibold text-gray-900">User Audit Logs</div>
            <div className="text-sm text-gray-600">{selectedUser?.email}</div>
          </div>
        }
        placement="right"
        width={900}
        onClose={handleDrawerClose}
        open={isDrawerOpen}
      >
        {selectedUser && <UserAuditLogsSidebarComponent user={selectedUser} />}
      </Drawer>
    </div>
  );
}
