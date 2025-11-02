import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Input, Modal } from 'antd';
import { Spin } from 'antd';
import { useState } from 'react';

import { type UserProfile, UserRole, type UsersSettings } from '../../../entity/users';
import type { WorkspaceResponse } from '../../../entity/workspaces';
import { workspaceApi } from '../../../entity/workspaces';

interface Props {
  user: UserProfile;
  globalSettings: UsersSettings;

  onClose: () => void;
  onWorkspaceCreated: (workspace: WorkspaceResponse) => void;

  workspacesCount: number;
}

export const CreateWorkspaceDialogComponent = ({
  user,
  globalSettings,
  onClose,
  onWorkspaceCreated,
  workspacesCount,
}: Props) => {
  const { message } = App.useApp();
  const [isCreating, setIsCreating] = useState(false);
  const [workspaceName, setWorkspaceName] = useState(workspacesCount === 0 ? 'My workspace' : '');

  const isAllowedToCreateWorkspaces =
    globalSettings.isMemberAllowedToCreateWorkspaces || user.role === UserRole.ADMIN;

  const handleCreateWorkspace = async () => {
    if (!workspaceName.trim()) {
      message.error('Please enter a workspace name');
      return;
    }

    setIsCreating(true);

    try {
      const newWorkspace = await workspaceApi.createWorkspace({
        name: workspaceName.trim(),
      });

      message.success('Workspace created successfully');
      onWorkspaceCreated(newWorkspace);
      onClose();
    } catch (error) {
      message.error((error as Error).message || 'Failed to create workspace');
    } finally {
      setIsCreating(false);
    }
  };

  if (!isAllowedToCreateWorkspaces) {
    return (
      <Modal
        title="Permission denied"
        open
        onCancel={onClose}
        footer={[
          <Button key="ok" type="primary" onClick={onClose}>
            OK
          </Button>,
        ]}
      >
        <p>
          You don&apos;t have permission to create workspaces. Please ask the administrator to
          create the workspace for you.
        </p>
      </Modal>
    );
  }

  return (
    <Modal
      title="Create workspace"
      open
      onCancel={onClose}
      footer={[
        <Button key="cancel" onClick={onClose} disabled={isCreating}>
          Cancel
        </Button>,

        <Button
          key="create"
          type="primary"
          onClick={handleCreateWorkspace}
          disabled={isCreating || !workspaceName.trim()}
          className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
        >
          {isCreating ? (
            <Spin indicator={<LoadingOutlined spin />} size="small" />
          ) : (
            'Create workspace'
          )}
        </Button>,
      ]}
    >
      <div className="mb-4">
        <div>
          Workspace is a place where you group:
          <br />
          - your databases;
          <br />
          - storages (like local drive, S3, Google Drive, etc.)
          <br />
          - notifiers (like email, Slack, Telegram, etc.);
          <br />- access control (if you have team);
        </div>

        <label className="mt-5 mb-2 block text-sm font-medium text-gray-700">Workspace name</label>
        <Input
          value={workspaceName}
          onChange={(e) => setWorkspaceName(e.target.value)}
          placeholder="Enter workspace name"
          disabled={isCreating}
          onPressEnter={handleCreateWorkspace}
          autoFocus
        />
      </div>
    </Modal>
  );
};
