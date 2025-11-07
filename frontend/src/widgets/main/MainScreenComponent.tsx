import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Spin, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import GitHubButton from 'react-github-btn';

import { APP_VERSION } from '../../constants';
import { type DiskUsage, diskApi } from '../../entity/disk';
import {
  type UserProfile,
  UserRole,
  type UsersSettings,
  WorkspaceRole,
  settingsApi,
  userApi,
} from '../../entity/users';
import { type WorkspaceResponse, workspaceApi } from '../../entity/workspaces';
import { DatabasesComponent } from '../../features/databases/ui/DatabasesComponent';
import { NotifiersComponent } from '../../features/notifiers/ui/NotifiersComponent';
import { SettingsComponent } from '../../features/settings';
import { StoragesComponent } from '../../features/storages/ui/StoragesComponent';
import { ProfileComponent } from '../../features/users';
import { UsersComponent } from '../../features/users/ui/UsersComponent';
import {
  CreateWorkspaceDialogComponent,
  WorkspaceSettingsComponent,
} from '../../features/workspaces';
import { useScreenHeight } from '../../shared/hooks';
import { WorkspaceSelectionComponent } from './WorkspaceSelectionComponent';

export const MainScreenComponent = () => {
  const { message } = App.useApp();
  const screenHeight = useScreenHeight();
  const contentHeight = screenHeight - 95;

  const [selectedTab, setSelectedTab] = useState<
    | 'notifiers'
    | 'storages'
    | 'databases'
    | 'profile'
    | 'postgresus-settings'
    | 'users'
    | 'settings'
  >('databases');
  const [diskUsage, setDiskUsage] = useState<DiskUsage | undefined>(undefined);
  const [user, setUser] = useState<UserProfile | undefined>(undefined);
  const [globalSettings, setGlobalSettings] = useState<UsersSettings | undefined>(undefined);

  const [workspaces, setWorkspaces] = useState<WorkspaceResponse[]>([]);
  const [selectedWorkspace, setSelectedWorkspace] = useState<WorkspaceResponse | undefined>(
    undefined,
  );

  const [isLoading, setIsLoading] = useState(false);
  const [showCreateWorkspaceDialog, setShowCreateWorkspaceDialog] = useState(false);

  const loadData = async () => {
    setIsLoading(true);

    try {
      const [diskUsage, user, workspaces, settings] = await Promise.all([
        diskApi.getDiskUsage(),
        userApi.getCurrentUser(),
        workspaceApi.getWorkspaces(),
        settingsApi.getSettings(),
      ]);

      setDiskUsage(diskUsage);
      setUser(user);
      setWorkspaces(workspaces.workspaces);
      setGlobalSettings(settings);
    } catch (e) {
      message.error((e as Error).message);
    }

    setIsLoading(false);
  };

  useEffect(() => {
    loadData();
  }, []);

  // Set selected workspace if none selected and workspaces available
  useEffect(() => {
    if (!selectedWorkspace && workspaces.length > 0) {
      const previouslySelectedWorkspaceId = localStorage.getItem('selected_workspace_id');
      const previouslySelectedWorkspace = workspaces.find(
        (workspace) => workspace.id === previouslySelectedWorkspaceId,
      );
      const workspaceToSelect = previouslySelectedWorkspace || workspaces[0];
      setSelectedWorkspace(workspaceToSelect);
    }
  }, [workspaces, selectedWorkspace]);

  // Save selected workspace to localStorage
  useEffect(() => {
    if (selectedWorkspace) {
      localStorage.setItem('selected_workspace_id', selectedWorkspace.id);
    }
  }, [selectedWorkspace]);

  const handleCreateWorkspace = () => {
    setShowCreateWorkspaceDialog(true);
  };

  const handleWorkspaceCreated = async (newWorkspace: WorkspaceResponse) => {
    try {
      const workspacesResponse = await workspaceApi.getWorkspaces();
      setWorkspaces(workspacesResponse.workspaces);
      setSelectedWorkspace(newWorkspace);
      setSelectedTab('databases');
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  const isUsedMoreThan95Percent =
    diskUsage && diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes > 0.95;

  const isCanManageDBs = selectedWorkspace?.userRole !== WorkspaceRole.VIEWER;

  return (
    <div style={{ height: screenHeight }} className="bg-[#f5f5f5] p-3">
      {/* ===================== NAVBAR ===================== */}
      <div className="mb-3 flex h-[60px] items-center rounded bg-white p-3 shadow">
        <div className="flex items-center gap-3 hover:opacity-80">
          <a href="https://postgresus.com" target="_blank" rel="noreferrer">
            <img className="h-[35px] w-[35px]" src="/logo.svg" />
          </a>
        </div>

        <div className="ml-6">
          {!isLoading && (
            <WorkspaceSelectionComponent
              workspaces={workspaces}
              selectedWorkspace={selectedWorkspace}
              onCreateWorkspace={handleCreateWorkspace}
              onWorkspaceSelect={setSelectedWorkspace}
            />
          )}
        </div>

        <div className="ml-auto flex items-center gap-5">
          <a
            className="!text-black hover:opacity-80"
            href="https://t.me/postgresus_community"
            target="_blank"
            rel="noreferrer"
          >
            Community
          </a>

          <div className="mt-1">
            <GitHubButton
              href="https://github.com/RostislavDugin/postgresus"
              data-icon="octicon-star"
              data-size="large"
              data-show-count="true"
              aria-label="Star RostislavDugin/postgresus on GitHub"
            >
              &nbsp;Star Postgresus on GitHub
            </GitHubButton>
          </div>

          {diskUsage && (
            <Tooltip title="To make backups locally and restore them, you need to have enough space on your disk. For restore, you need to have same amount of space that the backup size.">
              <div
                className={`cursor-pointer text-center text-xs ${isUsedMoreThan95Percent ? 'text-red-500' : 'text-gray-500'}`}
              >
                {(diskUsage.usedSpaceBytes / 1024 ** 3).toFixed(1)} of{' '}
                {(diskUsage.totalSpaceBytes / 1024 ** 3).toFixed(1)} GB
                <br />
                ROM used (
                {((diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes) * 100).toFixed(1)}%)
              </div>
            </Tooltip>
          )}
        </div>
      </div>
      {/* ===================== END NAVBAR ===================== */}

      {isLoading ? (
        <div className="flex items-center justify-center py-2" style={{ height: contentHeight }}>
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        </div>
      ) : (
        <div className="relative flex">
          <div
            className="max-w-[60px] min-w-[60px] rounded bg-white py-2 shadow"
            style={{ height: contentHeight }}
          >
            {[
              {
                text: 'Databases',
                name: 'databases',
                icon: '/icons/menu/database-gray.svg',
                selectedIcon: '/icons/menu/database-white.svg',
                onClick: () => setSelectedTab('databases'),
                isAdminOnly: false,
                marginTop: '0px',
                isVisible: true,
              },
              {
                text: 'Storages',
                name: 'storages',
                icon: '/icons/menu/storage-gray.svg',
                selectedIcon: '/icons/menu/storage-white.svg',
                onClick: () => setSelectedTab('storages'),
                isAdminOnly: false,
                marginTop: '0px',
                isVisible: !!selectedWorkspace,
              },
              {
                text: 'Notifiers',
                name: 'notifiers',
                icon: '/icons/menu/notifier-gray.svg',
                selectedIcon: '/icons/menu/notifier-white.svg',
                onClick: () => setSelectedTab('notifiers'),
                isAdminOnly: false,
                marginTop: '0px',
                isVisible: !!selectedWorkspace,
              },
              {
                text: 'Settings',
                name: 'settings',
                icon: '/icons/menu/workspace-settings-gray.svg',
                selectedIcon: '/icons/menu/workspace-settings-white.svg',
                onClick: () => setSelectedTab('settings'),
                isAdminOnly: false,
                marginTop: '0px',
                isVisible: !!selectedWorkspace,
              },
              {
                text: 'Profile',
                name: 'profile',
                icon: '/icons/menu/profile-gray.svg',
                selectedIcon: '/icons/menu/profile-white.svg',
                onClick: () => setSelectedTab('profile'),
                isAdminOnly: false,
                marginTop: '25px',
                isVisible: true,
              },
              {
                text: 'Postgresus settings',
                name: 'postgresus-settings',
                icon: '/icons/menu/global-settings-gray.svg',
                selectedIcon: '/icons/menu/global-settings-white.svg',
                onClick: () => setSelectedTab('postgresus-settings'),
                isAdminOnly: true,
                marginTop: '0px',
                isVisible: true,
              },
              {
                text: 'Users',
                name: 'users',
                icon: '/icons/menu/user-card-gray.svg',
                selectedIcon: '/icons/menu/user-card-white.svg',
                onClick: () => setSelectedTab('users'),
                isAdminOnly: true,
                marginTop: '0px',
                isVisible: true,
              },
            ]
              .filter((tab) => !tab.isAdminOnly || user?.role === UserRole.ADMIN)
              .filter((tab) => tab.isVisible)
              .map((tab) => (
                <div key={tab.text} className="flex justify-center">
                  <div
                    className={`flex h-[50px] w-[50px] cursor-pointer items-center justify-center rounded select-none ${selectedTab === tab.name ? 'bg-blue-600' : 'hover:bg-blue-50'}`}
                    onClick={tab.onClick}
                    style={{ marginTop: tab.marginTop }}
                  >
                    <div className="mb-1">
                      <div className="flex justify-center">
                        <img
                          src={selectedTab === tab.name ? tab.selectedIcon : tab.icon}
                          width={20}
                          alt={tab.text}
                          loading="lazy"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              ))}
          </div>

          {selectedTab === 'profile' && <ProfileComponent contentHeight={contentHeight} />}

          {selectedTab === 'postgresus-settings' && (
            <SettingsComponent contentHeight={contentHeight} />
          )}

          {selectedTab === 'users' && <UsersComponent contentHeight={contentHeight} />}

          {workspaces.length === 0 &&
          (selectedTab === 'databases' ||
            selectedTab === 'storages' ||
            selectedTab === 'notifiers' ||
            selectedTab === 'settings') ? (
            <div
              className="flex grow items-center justify-center rounded pl-5"
              style={{ height: contentHeight }}
            >
              <Button
                type="primary"
                size="large"
                onClick={handleCreateWorkspace}
                className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
              >
                Create workspace
              </Button>
            </div>
          ) : (
            <>
              {selectedTab === 'notifiers' && selectedWorkspace && (
                <NotifiersComponent
                  contentHeight={contentHeight}
                  workspace={selectedWorkspace}
                  isCanManageNotifiers={isCanManageDBs}
                  key={`notifiers-${selectedWorkspace.id}`}
                />
              )}
              {selectedTab === 'storages' && selectedWorkspace && (
                <StoragesComponent
                  contentHeight={contentHeight}
                  workspace={selectedWorkspace}
                  isCanManageStorages={isCanManageDBs}
                  key={`storages-${selectedWorkspace.id}`}
                />
              )}
              {selectedTab === 'databases' && selectedWorkspace && (
                <DatabasesComponent
                  contentHeight={contentHeight}
                  workspace={selectedWorkspace}
                  isCanManageDBs={isCanManageDBs}
                  key={`databases-${selectedWorkspace.id}`}
                />
              )}
              {selectedTab === 'settings' && selectedWorkspace && user && (
                <WorkspaceSettingsComponent
                  workspaceResponse={selectedWorkspace}
                  contentHeight={contentHeight}
                  user={user}
                  key={`settings-${selectedWorkspace.id}`}
                />
              )}
            </>
          )}

          <div className="absolute bottom-1 left-2 mb-[0px] text-sm text-gray-400">
            v{APP_VERSION}
          </div>
        </div>
      )}

      {/* Create Workspace Dialog */}
      {showCreateWorkspaceDialog && user && globalSettings && (
        <CreateWorkspaceDialogComponent
          user={user}
          globalSettings={globalSettings}
          onClose={() => setShowCreateWorkspaceDialog(false)}
          onWorkspaceCreated={handleWorkspaceCreated}
          workspacesCount={workspaces.length}
        />
      )}
    </div>
  );
};
