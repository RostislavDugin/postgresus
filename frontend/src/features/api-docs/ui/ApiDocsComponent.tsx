import { CopyOutlined } from '@ant-design/icons';
import { App, Collapse, Table, Tag, Tooltip } from 'antd';

import { getApplicationServer } from '../../../constants';
import { ClipboardHelper } from '../../../shared/lib/ClipboardHelper';

interface Props {
  contentHeight: number | string;
}

interface RequestField {
  field: string;
  type: string;
  required: boolean;
  description: string;
}

interface ResponseRow {
  code: string;
  meaning: string;
}

interface PublicEndpoint {
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  path: string;
  summary: string;
  description: string;
  requestBody?: RequestField[];
  responses: ResponseRow[];
  curl: string;
}

const methodColor = (method: string): string =>
  ({ GET: 'green', POST: 'blue', PUT: 'orange', DELETE: 'red', PATCH: 'purple' })[method] ??
  'default';

const responseColor = (code: string): string => {
  if (code.startsWith('2')) return 'green';
  if (code.startsWith('4')) return 'red';

  return 'default';
};

const buildPublicEndpoints = (baseUrl: string): PublicEndpoint[] => [
  {
    method: 'POST',
    path: '/api/v1/public/backups',
    summary: 'Trigger a database backup',
    description:
      'Starts a backup for the given database and blocks until it finishes or the server-side timeout (DATABASUS_API_BACKUP_SYNC_TIMEOUT, default 30m) elapses. Intended for CI/CD - take a verified backup before publishing an update.',
    requestBody: [
      {
        field: 'databaseId',
        type: 'string (uuid)',
        required: true,
        description: "ID of the database to back up. Copy it from the database's Config tab.",
      },
    ],
    responses: [
      { code: '200', meaning: 'Backup completed. Body: { backupId, status: "COMPLETED" }.' },
      {
        code: '202',
        meaning: 'Still running after the timeout. Body: { backupId, status: "IN_PROGRESS" }.',
      },
      { code: '401', meaning: 'Missing, invalid, revoked, or expired API key.' },
      { code: '403', meaning: "MEMBER key is not granted to this database's workspace." },
      { code: '404', meaning: 'Database not found.' },
      {
        code: '422',
        meaning:
          'Database has no workspace, backups are not configured, or the backup FAILED / CANCELED.',
      },
    ],
    curl: `curl -X POST ${baseUrl}/api/v1/public/backups \\
  -H "Authorization: Bearer dbs_YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"databaseId":"<DATABASE_ID>"}'`,
  },
];

export const ApiDocsComponent = ({ contentHeight }: Props) => {
  const { message } = App.useApp();

  const baseUrl = getApplicationServer();
  const endpoints = buildPublicEndpoints(baseUrl);

  const copyCurl = async (curl: string) => {
    try {
      await ClipboardHelper.copyToClipboard(curl);
      message.success('Copied to clipboard');
    } catch {
      message.error('Failed to copy');
    }
  };

  const items = endpoints.map((endpoint, index) => ({
    key: String(index),
    label: (
      <div className="flex flex-wrap items-center gap-2">
        <Tag color={methodColor(endpoint.method)} className="m-0 font-mono">
          {endpoint.method}
        </Tag>
        <code className="text-sm font-semibold dark:text-gray-100">{endpoint.path}</code>
        <span className="text-sm text-gray-500 dark:text-gray-400">- {endpoint.summary}</span>
      </div>
    ),
    children: (
      <div className="space-y-5">
        <p className="text-gray-600 dark:text-gray-300">{endpoint.description}</p>

        <div>
          <div className="mb-1 font-semibold dark:text-white">Authentication</div>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Send your API key in the <code>Authorization</code> header as{' '}
            <code>Bearer dbs_...</code>. Admins create keys in the <strong>API keys</strong> page.
          </p>
        </div>

        {endpoint.requestBody && (
          <div>
            <div className="mb-2 font-semibold dark:text-white">Request body (JSON)</div>
            <Table<RequestField>
              size="small"
              pagination={false}
              rowKey="field"
              dataSource={endpoint.requestBody}
              columns={[
                {
                  title: 'Field',
                  dataIndex: 'field',
                  render: (field: string) => <code>{field}</code>,
                },
                { title: 'Type', dataIndex: 'type' },
                {
                  title: 'Required',
                  dataIndex: 'required',
                  render: (required: boolean) =>
                    required ? <Tag color="red">required</Tag> : <Tag>optional</Tag>,
                },
                { title: 'Description', dataIndex: 'description' },
              ]}
            />
          </div>
        )}

        <div>
          <div className="mb-2 font-semibold dark:text-white">Responses</div>
          <Table<ResponseRow>
            size="small"
            pagination={false}
            rowKey="code"
            dataSource={endpoint.responses}
            columns={[
              {
                title: 'Status',
                dataIndex: 'code',
                width: 90,
                render: (code: string) => <Tag color={responseColor(code)}>{code}</Tag>,
              },
              { title: 'Meaning', dataIndex: 'meaning' },
            ]}
          />
        </div>

        <div>
          <div className="mb-2 flex items-center justify-between">
            <span className="font-semibold dark:text-white">Example</span>
            <Tooltip title="Copy">
              <button
                className="cursor-pointer rounded p-1 text-gray-400 hover:text-gray-700 dark:hover:text-white"
                onClick={() => copyCurl(endpoint.curl)}
              >
                <CopyOutlined />
              </button>
            </Tooltip>
          </div>
          <pre className="overflow-x-auto rounded bg-gray-100 p-3 text-xs dark:bg-gray-700 dark:text-gray-100">
            {endpoint.curl}
          </pre>
        </div>
      </div>
    ),
  }));

  return (
    <div className="flex grow">
      <div className="w-full">
        <div
          className="grow overflow-y-auto rounded bg-white p-5 shadow dark:bg-gray-800"
          style={{ height: contentHeight }}
        >
          <h1 className="mb-1 text-2xl font-bold dark:text-white">API documentation</h1>
          <p className="mb-5 text-gray-500 dark:text-gray-400">
            Public endpoints you can call programmatically with an API key (for example, from a
            CI/CD pipeline). Base URL: <code>{baseUrl}</code>
          </p>

          <Collapse items={items} defaultActiveKey={['0']} />
        </div>
      </div>
    </div>
  );
};
