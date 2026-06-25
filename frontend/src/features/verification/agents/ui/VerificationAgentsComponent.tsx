import {
  CopyOutlined,
  DeleteOutlined,
  EyeOutlined,
  LoadingOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { App, Button, Input, Modal, Popconfirm, Spin, Table, Tag, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useCallback, useEffect, useRef, useState } from 'react';

import {
  type VerificationAgent,
  verificationAgentApi,
} from '../../../../entity/verification/agents';
import { ClipboardHelper } from '../../../../shared/lib/ClipboardHelper';
import { AGENT_STATUS_COLORS, AGENT_STATUS_LABELS, getAgentStatus } from '../model/agentStatus';

const LIST_REFRESH_MS = 15_000;
const MUTED_TEXT_CLASS = 'text-gray-400 dark:text-gray-500';

type AgentArchitecture = 'amd64' | 'arm64';

const formatCapacity = (agent: VerificationAgent): string => {
  if (
    agent.maxCpu <= 0 &&
    agent.maxRamGb <= 0 &&
    agent.maxDiskGb <= 0 &&
    agent.maxConcurrentJobs <= 0
  ) {
    return '';
  }

  const parts: string[] = [];
  if (agent.maxCpu > 0) parts.push(`${agent.maxCpu} CPU`);
  if (agent.maxRamGb > 0) parts.push(`${agent.maxRamGb} GB RAM`);
  if (agent.maxDiskGb > 0) parts.push(`${agent.maxDiskGb} GB disk`);
  if (agent.maxConcurrentJobs > 0) parts.push(`${agent.maxConcurrentJobs} jobs`);
  return parts.join(' · ');
};

const TOKEN_PLACEHOLDER = '<YOUR_AGENT_TOKEN>';

const buildInstallCommand = (arch: AgentArchitecture): string => {
  const host = window.location.origin;
  return `curl -L -o verification-agent "${host}/api/v1/system/verification-agent?arch=${arch}" && chmod +x verification-agent`;
};

const buildLaunchCommand = (agentId: string, token: string): string => {
  const host = window.location.origin;

  return [
    './verification-agent start \\',
    `  --databasus-host=${host} \\`,
    `  --agent-id=${agentId} \\`,
    `  --token=${token} \\`,
    '  --max-cpu=2 \\',
    '  --max-ram-mb=2048 \\',
    '  --max-disk-gb=20 \\',
    '  --max-concurrent-jobs=1',
  ].join('\n');
};

export const VerificationAgentsComponent = () => {
  const { message } = App.useApp();

  const [agents, setAgents] = useState<VerificationAgent[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newAgentName, setNewAgentName] = useState('');
  const [isCreating, setIsCreating] = useState(false);

  const [revealedToken, setRevealedToken] = useState<string | null>(null);
  const [revealedTokenAgentName, setRevealedTokenAgentName] = useState<string>('');
  const [revealedTokenAgentId, setRevealedTokenAgentId] = useState<string>('');
  const [selectedArch, setSelectedArch] = useState<AgentArchitecture>('amd64');

  const [rotatingAgent, setRotatingAgent] = useState<VerificationAgent | null>(null);
  const [isRotating, setIsRotating] = useState(false);

  const [viewingInstallAgent, setViewingInstallAgent] = useState<VerificationAgent | null>(null);

  const [deletingAgentId, setDeletingAgentId] = useState<string | null>(null);
  const [currentTimeMs, setCurrentTimeMs] = useState<number>(Date.now());

  const copyToClipboard = async (text: string) => {
    try {
      await ClipboardHelper.copyToClipboard(text);
      message.success('Copied to clipboard');
    } catch {
      message.error('Failed to copy');
    }
  };

  const closeRevealedTokenModal = () => {
    setRevealedToken(null);
    setRevealedTokenAgentName('');
    setRevealedTokenAgentId('');
  };

  const renderCodeBlock = (code: string) => (
    <div className="relative mt-2">
      <pre className="rounded-md bg-gray-900 p-4 pr-10 font-mono text-sm break-all whitespace-pre-wrap text-gray-100">
        {code}
      </pre>
      <Tooltip title="Copy">
        <button
          className="absolute top-2 right-2 cursor-pointer rounded p-1 text-gray-400 hover:text-white"
          onClick={() => copyToClipboard(code)}
        >
          <CopyOutlined />
        </button>
      </Tooltip>
    </div>
  );

  const renderArchButton = (arch: AgentArchitecture) => (
    <Button
      type="primary"
      ghost={selectedArch !== arch}
      onClick={() => setSelectedArch(arch)}
      className="mr-2"
    >
      {arch}
    </Button>
  );

  const renderAgentIdRow = (agentId: string) => (
    <div className="mt-3 flex items-center text-sm text-gray-500 dark:text-gray-400">
      <span className="mr-1">Agent ID:</span>
      <code className="rounded bg-gray-100 px-2 py-0.5 text-xs dark:bg-gray-700">{agentId}</code>
      <Tooltip title="Copy">
        <button
          className="ml-1 cursor-pointer rounded p-1 text-gray-400 hover:text-gray-700 dark:hover:text-white"
          onClick={() => copyToClipboard(agentId)}
        >
          <CopyOutlined style={{ fontSize: 12 }} />
        </button>
      </Tooltip>
    </div>
  );

  const renderArchitecturePicker = () => (
    <div className="mt-4">
      <div className="mb-1 text-sm font-medium text-gray-700 dark:text-gray-300">Architecture</div>
      <div className="flex">
        {renderArchButton('amd64')}
        {renderArchButton('arm64')}
      </div>
    </div>
  );

  const renderInstallAndLaunchSteps = (agentId: string, token: string) => (
    <>
      <div className="mt-4 font-semibold dark:text-white">Step 1 - Install</div>
      {renderCodeBlock(buildInstallCommand(selectedArch))}

      <div className="mt-4 font-semibold dark:text-white">Step 2 - Launch</div>
      <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
        The capacity values below are starting defaults - tune <code>--max-cpu</code>,{' '}
        <code>--max-ram-mb</code>, <code>--max-disk-gb</code> and <code>--max-concurrent-jobs</code>{' '}
        to the machine running the agent.
      </p>
      {renderCodeBlock(buildLaunchCommand(agentId, token))}

      <div className="mt-4 font-semibold dark:text-white">After installation</div>
      <ul className="mt-1 list-disc space-y-1 pl-5 text-sm text-gray-600 dark:text-gray-400">
        <li>
          The agent runs in the background after <code>start</code>
        </li>
        <li>
          Check status: <code>./verification-agent status</code>
        </li>
        <li>
          View logs: <code>databasus-verification.log</code> in the working directory
        </li>
        <li>
          Stop the agent: <code>./verification-agent stop</code>
        </li>
      </ul>
    </>
  );

  const closeViewingInstallModal = () => {
    setViewingInstallAgent(null);
  };

  const handleCreate = async () => {
    const name = newAgentName.trim();
    if (!name) {
      message.error('Name is required');
      return;
    }

    setIsCreating(true);

    try {
      const response = await verificationAgentApi.createAgent(name);
      setAgents((prev) => [response.agent, ...prev]);
      setRevealedToken(response.token);
      setRevealedTokenAgentName(response.agent.name);
      setRevealedTokenAgentId(response.agent.id);
      setIsCreateModalOpen(false);
      setNewAgentName('');
    } catch (e) {
      message.error((e as Error).message);
    } finally {
      setIsCreating(false);
    }
  };

  const handleConfirmRotate = async () => {
    if (!rotatingAgent) return;

    setIsRotating(true);

    try {
      const response = await verificationAgentApi.rotateToken(rotatingAgent.id);
      setRevealedToken(response.token);
      setRevealedTokenAgentName(rotatingAgent.name);
      setRevealedTokenAgentId(rotatingAgent.id);
      setRotatingAgent(null);
    } catch (e) {
      message.error((e as Error).message);
    } finally {
      setIsRotating(false);
    }
  };

  const handleDelete = async (agent: VerificationAgent) => {
    setDeletingAgentId(agent.id);

    try {
      await verificationAgentApi.deleteAgent(agent.id);
      setAgents((prev) => prev.filter((a) => a.id !== agent.id));
      message.success(`Agent "${agent.name}" deleted`);
    } catch (e) {
      message.error((e as Error).message);
    } finally {
      setDeletingAgentId(null);
    }
  };

  // Pause background polling while any modal is open so it doesn't trample state.
  const hasOpenModal = useRef(false);
  hasOpenModal.current =
    isCreateModalOpen ||
    revealedToken !== null ||
    rotatingAgent !== null ||
    viewingInstallAgent !== null;

  const loadAgents = useCallback(async () => {
    try {
      const list = await verificationAgentApi.listAgents();
      setAgents(list);
    } catch (e) {
      message.error((e as Error).message);
    }
  }, [message]);

  useEffect(() => {
    setIsLoading(true);
    loadAgents().finally(() => setIsLoading(false));
  }, [loadAgents]);

  useEffect(() => {
    const interval = window.setInterval(() => {
      setCurrentTimeMs(Date.now());

      if (!hasOpenModal.current) {
        loadAgents();
      }
    }, LIST_REFRESH_MS);

    return () => window.clearInterval(interval);
  }, [loadAgents]);

  const columns: ColumnsType<VerificationAgent> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <span className="font-medium text-gray-900 dark:text-white">{name}</span>
      ),
    },
    {
      title: 'Status',
      key: 'status',
      width: 180,
      render: (_, record) => {
        const status = getAgentStatus(record.lastSeenAt, currentTimeMs);
        return (
          <div className="flex items-center gap-2">
            <Tag color={AGENT_STATUS_COLORS[status]} className="!m-0">
              {AGENT_STATUS_LABELS[status]}
            </Tag>
            {record.lastSeenAt && (
              <span className={`text-xs ${MUTED_TEXT_CLASS}`}>
                {dayjs(record.lastSeenAt).fromNow()}
              </span>
            )}
          </div>
        );
      },
    },
    {
      title: 'Capacity',
      key: 'capacity',
      render: (_, record) => {
        const capacity = formatCapacity(record);
        return capacity ? (
          <span className="text-xs text-gray-700 dark:text-gray-300">{capacity}</span>
        ) : (
          <span className={`text-xs ${MUTED_TEXT_CLASS}`}>not yet reported</span>
        );
      },
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 110,
      render: (createdAt: string) => (
        <span className={`text-xs ${MUTED_TEXT_CLASS}`}>{dayjs(createdAt).fromNow()}</span>
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 100,
      align: 'right',
      render: (_, record) => (
        <div className="flex items-center justify-end gap-1">
          <Tooltip title="View install commands">
            <Button
              type="text"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => setViewingInstallAgent(record)}
            />
          </Tooltip>

          <Tooltip title="Rotate token">
            <Button
              type="text"
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => setRotatingAgent(record)}
            />
          </Tooltip>

          <Popconfirm
            title="Delete this agent?"
            okText="Delete"
            okButtonProps={{ danger: true, loading: deletingAgentId === record.id }}
            cancelText="Cancel"
            onConfirm={() => handleDelete(record)}
          >
            <Tooltip title="Delete">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </div>
      ),
    },
  ];

  return (
    <section className="my-8 max-w-[800px]">
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-baseline gap-3">
          <h2 className="text-xl font-bold dark:text-white">Verification agents</h2>
        </div>

        <div className="flex items-center gap-2">
          <Button
            type="primary"
            size="small"
            icon={<PlusOutlined />}
            onClick={() => setIsCreateModalOpen(true)}
          >
            Create
          </Button>
        </div>
      </div>

      <p className="mb-4 max-w-2xl text-sm text-gray-500 dark:text-gray-400">
        Agents that run restore verifications to confirm a backup is restorable (
        <a
          href="https://databasus.com/restore-verification"
          target="_blank"
          rel="noopener noreferrer"
        >
          read more
        </a>
        )
      </p>

      {isLoading ? (
        <div className="py-4">
          <Spin indicator={<LoadingOutlined spin />} />
        </div>
      ) : agents.length === 0 ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">No agents registered yet.</p>
      ) : (
        <Table
          columns={columns}
          dataSource={agents}
          pagination={false}
          rowKey="id"
          size="small"
          className="[&_.ant-table-tbody_.ant-table-cell]:align-top"
        />
      )}

      <Modal
        title="Create verification agent"
        open={isCreateModalOpen}
        onCancel={() => {
          setIsCreateModalOpen(false);
          setNewAgentName('');
        }}
        footer={[
          <Button
            key="cancel"
            onClick={() => {
              setIsCreateModalOpen(false);
              setNewAgentName('');
            }}
          >
            Cancel
          </Button>,
          <Button key="create" type="primary" loading={isCreating} onClick={handleCreate}>
            Create
          </Button>,
        ]}
      >
        <p className="mb-2 text-sm text-gray-500 dark:text-gray-400">
          A token will be generated and shown exactly once on the next screen.
        </p>
        <Input
          placeholder="Agent name"
          value={newAgentName}
          onChange={(e) => setNewAgentName(e.target.value)}
          onPressEnter={handleCreate}
          maxLength={200}
          autoFocus
        />
      </Modal>

      <Modal
        title="Rotate token"
        open={rotatingAgent !== null}
        onCancel={() => setRotatingAgent(null)}
        footer={[
          <Button key="cancel" onClick={() => setRotatingAgent(null)}>
            Cancel
          </Button>,
          <Button
            key="rotate"
            type="primary"
            danger
            loading={isRotating}
            onClick={handleConfirmRotate}
          >
            Rotate token
          </Button>,
        ]}
      >
        <p className="text-sm text-gray-700 dark:text-gray-300">
          Rotating the token for <strong>{rotatingAgent?.name}</strong> invalidates the existing
          token immediately. Any worker still using the old token will be rejected on its next
          heartbeat.
        </p>
      </Modal>

      <Modal
        title="Agent token"
        open={revealedToken !== null}
        onCancel={closeRevealedTokenModal}
        width={640}
        footer={
          <Button type="primary" onClick={closeRevealedTokenModal}>
            I&apos;ve saved the token
          </Button>
        }
        maskClosable={false}
      >
        <p className="text-sm text-gray-700 dark:text-gray-300">
          Token for <strong>{revealedTokenAgentName}</strong>:
        </p>
        {renderCodeBlock(revealedToken ?? '')}

        {renderAgentIdRow(revealedTokenAgentId)}
        {renderArchitecturePicker()}
        {renderInstallAndLaunchSteps(revealedTokenAgentId, revealedToken ?? '')}

        <p className="mt-3 text-sm text-amber-600 dark:text-amber-400">
          Shown once. Store it securely - you won&apos;t be able to retrieve it again.
        </p>
      </Modal>

      <Modal
        title="Install commands"
        open={viewingInstallAgent !== null}
        onCancel={closeViewingInstallModal}
        width={640}
        footer={
          <Button type="primary" onClick={closeViewingInstallModal}>
            Close
          </Button>
        }
      >
        <p className="text-sm text-gray-700 dark:text-gray-300">
          Install commands for <strong>{viewingInstallAgent?.name}</strong>. Replace{' '}
          <code className="rounded bg-gray-100 px-1 text-xs dark:bg-gray-700">
            {TOKEN_PLACEHOLDER}
          </code>{' '}
          with the token you saved when you created or last rotated this agent.
        </p>

        {viewingInstallAgent && (
          <>
            {renderAgentIdRow(viewingInstallAgent.id)}
            {renderArchitecturePicker()}
            {renderInstallAndLaunchSteps(viewingInstallAgent.id, TOKEN_PLACEHOLDER)}
          </>
        )}
      </Modal>
    </section>
  );
};
