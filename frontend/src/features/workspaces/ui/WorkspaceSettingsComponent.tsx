import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Input, Spin } from 'antd';
import { useEffect, useRef, useState } from 'react';

import { databaseApi } from '../../../entity/databases/api/databaseApi';
import type { UserProfile } from '../../../entity/users/model/UserProfile';
import { UserRole } from '../../../entity/users/model/UserRole';
import { WorkspaceRole } from '../../../entity/users/model/WorkspaceRole';
import { workspaceApi } from '../../../entity/workspaces/api/workspaceApi';
import type { Workspace } from '../../../entity/workspaces/model/Workspace';
import type { WorkspaceResponse } from '../../../entity/workspaces/model/WorkspaceResponse';
import { useIsMobile } from '../../../shared/hooks';
import { WorkspaceAuditLogsComponent } from './WorkspaceAuditLogsComponent';
import { WorkspaceMembershipComponent } from './WorkspaceMembershipComponent';

interface Props {
  workspaceResponse: WorkspaceResponse;
  user: UserProfile;
  contentHeight: number;
}

export function WorkspaceSettingsComponent({ workspaceResponse, user, contentHeight }: Props) {
  const { message, modal } = App.useApp();
  const isMobile = useIsMobile();
  const [workspace, setWorkspace] = useState<Workspace | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const [formWorkspace, setFormWorkspace] = useState<Partial<Workspace>>({});
  const [nameError, setNameError] = useState(false);

  const [basicInfoChanges, setBasicInfoChanges] = useState(false);

  // Only OWNER and ADMIN can edit workspace settings
  // MEMBER and VIEWER can only view
  const canEdit =
    user.role === UserRole.ADMIN ||
    workspaceResponse.userRole === WorkspaceRole.OWNER ||
    workspaceResponse.userRole === WorkspaceRole.ADMIN;

  useEffect(() => {
    loadWorkspace();
  }, [workspaceResponse.id]);

  const checkBasicInfoChanges = (newFormWorkspace: Partial<Workspace>): boolean => {
    if (!workspace) return false;
    return newFormWorkspace.name !== workspace.name;
  };

  const loadWorkspace = async () => {
    setIsLoading(true);

    try {
      const workspaceData = await workspaceApi.getWorkspace(workspaceResponse.id);
      setWorkspace(workspaceData);
      setFormWorkspace(workspaceData);
      setNameError(false);

      setBasicInfoChanges(false);
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to load workspace';
      message.error(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const handleFieldChange = <K extends keyof Workspace>(field: K, value: Workspace[K]) => {
    const newFormWorkspace = { ...formWorkspace, [field]: value };
    setFormWorkspace(newFormWorkspace);

    if (workspace) {
      setBasicInfoChanges(checkBasicInfoChanges(newFormWorkspace));
    }
  };

  const saveBasicInfo = async () => {
    if (!basicInfoChanges || !workspace || !canEdit) return;

    if (!formWorkspace.name?.trim()) {
      setNameError(true);
      message.error('Workspace name is required');
      return;
    }
    setNameError(false);

    setIsSaving(true);
    try {
      const updateData = {
        ...workspace,
        name: formWorkspace.name,
      };
      const updatedWorkspace = await workspaceApi.updateWorkspace(workspace.id, updateData);
      setWorkspace(updatedWorkspace);
      setFormWorkspace(updatedWorkspace);

      setBasicInfoChanges(false);

      setNameError(false);
      message.success('Basic information updated successfully');
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to update basic information';
      message.error(errorMessage);
    } finally {
      setIsSaving(false);
    }
  };

  const handleDeleteWorkspace = async () => {
    if (!workspace) {
      message.error('Workspace not found');
      return;
    }

    if (!canEdit) {
      message.error('You do not have permission to delete this workspace');
      return;
    }

    modal.confirm({
      title: 'Delete Workspace',
      content: (
        <div>
          <p>
            Are you sure you want to delete the workspace <strong>{workspace.name}</strong>?
          </p>
          <p className="mt-2 text-red-600">
            <strong>This action cannot be undone.</strong> All data and associated resources will be
            permanently removed.
          </p>
        </div>
      ),
      okText: 'Delete Workspace',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: async () => {
        setIsDeleting(true);
        try {
          // Check if there are any databases in the workspace
          const databases = await databaseApi.getDatabases(workspace.id);

          if (databases && databases.length > 0) {
            message.error(
              `Cannot delete workspace. Please remove all databases first. Found ${databases.length} database(s).`,
            );
            return;
          }

          await workspaceApi.deleteWorkspace(workspace.id);
          message.success('Workspace deleted successfully');
          window.location.href = '/';
        } catch (error: unknown) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to delete workspace';
          message.error(errorMessage);
        } finally {
          setIsDeleting(false);
        }
      },
    });
  };

  return (
    <div className="flex grow sm:pl-2">
      <div className="w-full">
        <div
          ref={scrollContainerRef}
          className={`grow overflow-y-auto rounded bg-white shadow ${isMobile ? 'p-3' : 'p-5'}`}
          style={{ height: contentHeight }}
        >
          <h1 className="mb-6 text-2xl font-bold">Workspace settings</h1>

          {isLoading || !workspace ? (
            <Spin indicator={<LoadingOutlined spin />} size="large" />
          ) : (
            <>
              {!canEdit && (
                <div className="my-4 max-w-[500px] rounded-md bg-yellow-50 p-3">
                  <div className="text-sm text-yellow-800">
                    You don&apos;t have permission to modify these settings
                  </div>
                </div>
              )}

              <div className="space-y-6 text-sm">
                <div className="max-w-2xl border-b border-gray-200 pb-6">
                  <div className="max-w-md">
                    <div className="mb-1 font-medium text-gray-900">Workspace name</div>
                    <Input
                      value={formWorkspace.name || ''}
                      onChange={(e) => {
                        setNameError(false);
                        handleFieldChange('name', e.target.value);
                      }}
                      disabled={!canEdit}
                      placeholder="Enter workspace name"
                      maxLength={100}
                      status={nameError ? 'error' : undefined}
                    />
                  </div>

                  {basicInfoChanges && canEdit && (
                    <div className="mt-4 flex space-x-2">
                      <Button
                        type="primary"
                        onClick={saveBasicInfo}
                        loading={isSaving}
                        disabled={isSaving}
                        className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
                      >
                        {isSaving ? 'Saving...' : 'Save Changes'}
                      </Button>

                      <Button
                        type="default"
                        onClick={() => {
                          if (workspace) {
                            const updatedForm = { ...formWorkspace, name: workspace.name };
                            setFormWorkspace(updatedForm);
                            setBasicInfoChanges(false);
                            setNameError(false);
                          }
                        }}
                        disabled={isSaving}
                      >
                        Reset
                      </Button>
                    </div>
                  )}
                </div>

                <div className="max-w-2xl border-b border-gray-200 pb-6">
                  <WorkspaceMembershipComponent workspaceResponse={workspaceResponse} user={user} />
                </div>

                {canEdit && (
                  <div className="max-w-2xl border-b border-gray-200 pb-6">
                    <h2 className="mb-4 text-xl font-bold text-gray-900">Danger Zone</h2>

                    <div className="rounded-lg border border-red-200 bg-red-50 p-4">
                      <div
                        className={`flex ${isMobile ? 'flex-col gap-3' : 'items-start justify-between'}`}
                      >
                        <div className="flex-1">
                          <div className="font-medium text-red-900">Delete this workspace</div>
                          <div className="mt-1 text-sm text-red-700">
                            Once you delete a workspace, there is no going back. All data and
                            resources associated with this workspace will be permanently removed.
                          </div>
                        </div>

                        <div className={isMobile ? '' : 'ml-4'}>
                          <Button
                            type="primary"
                            danger
                            onClick={handleDeleteWorkspace}
                            disabled={!canEdit || isDeleting || isSaving}
                            loading={isDeleting}
                            className={`bg-red-600 hover:bg-red-700 ${isMobile ? 'w-full' : ''}`}
                          >
                            {isDeleting ? 'Deleting...' : 'Delete workspace'}
                          </Button>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                <WorkspaceAuditLogsComponent
                  workspaceId={workspace.id}
                  scrollContainerRef={scrollContainerRef}
                />
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
