import { LoadingOutlined } from '@ant-design/icons';
import {
  App,
  Button,
  DatePicker,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Spin,
  Table,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';

import {
  type ApiKey,
  ApiKeyRole,
  type CreateApiKeyRequest,
  apiKeysApi,
} from '../../../entity/api-keys';

interface Props {
  contentHeight: number | string;
  workspaces: { id: string; name: string }[];
}

export function ApiKeysComponent({ contentHeight, workspaces }: Props) {
  const { message } = App.useApp();
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [createdToken, setCreatedToken] = useState<string | undefined>(undefined);

  const loadApiKeys = async () => {
    setIsLoading(true);
    try {
      const response = await apiKeysApi.listApiKeys();
      setApiKeys(response.apiKeys);
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : 'Failed to load API keys');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreate = async () => {
    const values = await form.validateFields();
    setIsCreating(true);
    try {
      const request: CreateApiKeyRequest = {
        name: values.name,
        role: values.role,
        expiresAt: values.expiresAt ? dayjs(values.expiresAt).toISOString() : undefined,
        workspaceIds: values.role === ApiKeyRole.MEMBER ? values.workspaceIds : undefined,
      };
      const response = await apiKeysApi.createApiKey(request);
      setCreatedToken(response.token);
      setIsCreateOpen(false);
      form.resetFields();
      await loadApiKeys();
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : 'Failed to create API key');
    } finally {
      setIsCreating(false);
    }
  };

  const handleRevoke = async (id: string) => {
    try {
      await apiKeysApi.revokeApiKey(id);
      message.success('API key revoked');
      await loadApiKeys();
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : 'Failed to revoke API key');
    }
  };

  const [form] = Form.useForm<CreateApiKeyRequest & { role: ApiKeyRole }>();
  const selectedRole = Form.useWatch('role', form);

  useEffect(() => {
    loadApiKeys();
  }, []);

  const columns: ColumnsType<ApiKey> = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Role', dataIndex: 'role', key: 'role' },
    {
      title: 'Token',
      dataIndex: 'tokenPrefix',
      key: 'tokenPrefix',
      render: (p: string) => `${p}…`,
    },
    {
      title: 'Workspaces',
      dataIndex: 'workspaceIds',
      key: 'workspaceIds',
      render: (ids: string[] | undefined, record: ApiKey) =>
        record.role === ApiKeyRole.ADMIN ? 'All' : `${(ids ?? []).length} workspace(s)`,
    },
    {
      title: 'Last used',
      dataIndex: 'lastUsedAt',
      key: 'lastUsedAt',
      render: (v?: string) => (v ? dayjs(v).fromNow() : '—'),
    },
    {
      title: 'Expires',
      dataIndex: 'expiresAt',
      key: 'expiresAt',
      render: (v?: string) => (v ? dayjs(v).format('YYYY-MM-DD') : 'Never'),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (v: string) => dayjs(v).format('YYYY-MM-DD'),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record: ApiKey) => (
        <Popconfirm title="Revoke this API key?" onConfirm={() => handleRevoke(record.id)}>
          <Button danger size="small">
            Revoke
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div className="flex grow">
      <div className="w-full">
        <div
          className="grow overflow-y-auto rounded bg-white p-5 shadow dark:bg-gray-800"
          style={{ height: contentHeight }}
        >
          <div className="mb-4 flex items-center justify-between">
            <h1 className="text-2xl font-bold dark:text-white">API keys</h1>
            <Button type="primary" onClick={() => setIsCreateOpen(true)}>
              Create API key
            </Button>
          </div>

          {isLoading ? (
            <div className="flex h-64 items-center justify-center">
              <Spin indicator={<LoadingOutlined spin />} size="large" />
            </div>
          ) : (
            <Table
              columns={columns}
              dataSource={apiKeys}
              rowKey="id"
              pagination={false}
              size="small"
            />
          )}
        </div>
      </div>

      <Modal
        title="Create API key"
        open={isCreateOpen}
        onCancel={() => setIsCreateOpen(false)}
        onOk={handleCreate}
        confirmLoading={isCreating}
        okText="Create"
        maskClosable={false}
      >
        <Form form={form} layout="vertical" initialValues={{ role: ApiKeyRole.ADMIN }}>
          <Form.Item
            name="name"
            label="Name"
            rules={[{ required: true, message: 'Name is required' }]}
          >
            <Input placeholder="e.g. CI/CD pipeline" />
          </Form.Item>
          <Form.Item name="role" label="Role" rules={[{ required: true }]}>
            <Select
              options={[
                { label: 'Admin (all workspaces)', value: ApiKeyRole.ADMIN },
                { label: 'Member (granted workspaces)', value: ApiKeyRole.MEMBER },
              ]}
            />
          </Form.Item>
          {selectedRole === ApiKeyRole.MEMBER && (
            <Form.Item
              name="workspaceIds"
              label="Workspaces"
              rules={[{ required: true, message: 'Select at least one workspace' }]}
            >
              <Select
                mode="multiple"
                options={workspaces.map((w) => ({ label: w.name, value: w.id }))}
              />
            </Form.Item>
          )}
          <Form.Item name="expiresAt" label="Expires at (optional)">
            <DatePicker className="w-full" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="API key created"
        open={createdToken !== undefined}
        onCancel={() => setCreatedToken(undefined)}
        onOk={() => setCreatedToken(undefined)}
        okText="Done"
      >
        <p className="mb-2 text-gray-600 dark:text-gray-300">
          Copy this token now - you will not be able to see it again.
        </p>
        <Typography.Paragraph
          copyable={{ text: createdToken }}
          className="rounded bg-gray-100 p-2 break-all dark:bg-gray-700"
        >
          {createdToken}
        </Typography.Paragraph>
      </Modal>
    </div>
  );
}
