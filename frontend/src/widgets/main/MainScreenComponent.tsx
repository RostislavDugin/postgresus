import { LoadingOutlined, MenuOutlined } from '@ant-design/icons';
import { App, Button, Spin, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import GitHubButton from 'react-github-btn';

import { APP_VERSION } from '../../constants';
import { type DiskUsage, diskApi } from '../../entity/disk';
import {
  type UserProfile,
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
import { useIsMobile, useScreenHeight } from '../../shared/hooks';
import { SidebarComponent } from './SidebarComponent';
import { ThemeToggleComponent } from './ThemeToggleComponent';
import { WorkspaceSelectionComponent } from './WorkspaceSelectionComponent';

export const MainScreenComponent = () => {
  const { message } = App.useApp();
  const screenHeight = useScreenHeight();
  const isMobile = useIsMobile();
  const contentHeight = screenHeight - (isMobile ? 70 : 95);

  const [selectedTab, setSelectedTab] = useState<
    'notifiers' | 'storages' | 'databases' | 'profile' | 'databasus-settings' | 'users' | 'settings'
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
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

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

  const tabs = [
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
      text: 'Databasus settings',
      name: 'databasus-settings',
      icon: '/icons/menu/global-settings-gray.svg',
      selectedIcon: '/icons/menu/global-settings-white.svg',
      onClick: () => setSelectedTab('databasus-settings'),
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
  ];

  return (
    <div style={{ height: screenHeight }} className="bg-[#f5f5f5] p-2 md:p-3 dark:bg-gray-900">
      <div className="mb-2 flex h-[50px] items-center rounded bg-white px-2 py-2 shadow md:mb-3 md:h-[60px] md:p-3 dark:bg-gray-800">
        <div className="flex items-center gap-2 hover:opacity-80 md:gap-3">
          <a href="https://databasus.com" target="_blank" rel="noreferrer">
            <img
              className="padding-[2.5px] h-[30px] w-[30px] md:h-[40px] md:w-[40px]"
              src="/logo.svg"
            />
          </a>
        </div>

        <div className="ml-2 flex-1 pr-2 md:ml-5 md:flex-initial md:pr-0">
          {!isLoading && (
            <WorkspaceSelectionComponent
              workspaces={workspaces}
              selectedWorkspace={selectedWorkspace}
              onCreateWorkspace={handleCreateWorkspace}
              onWorkspaceSelect={setSelectedWorkspace}
            />
          )}
        </div>

        <div className="ml-auto hidden items-center gap-5 md:flex">
          <a
            className="!text-black hover:opacity-80 dark:!text-gray-200"
            href="https://databasus.com/installation"
            target="_blank"
            rel="noreferrer"
          >
            Docs
          </a>

          <a
            className="!text-black hover:opacity-80 dark:!text-gray-200"
            href="https://databasus.com/contribute"
            target="_blank"
            rel="noreferrer"
          >
            Contribute
          </a>
          <a
            className="!text-black hover:opacity-80 dark:!text-gray-200"
            href="https://t.me/databasus_community"
            target="_blank"
            rel="noreferrer"
          >
            Community
          </a>

          <div className="mt-1">
            <GitHubButton
              href="https://github.com/databasus/databasus"
              data-icon="octicon-star"
              data-size="large"
              data-show-count="true"
              aria-label="Star databasus/databasus on GitHub"
            >
              &nbsp;Star Databasus on GitHub
            </GitHubButton>
          </div>

          {diskUsage && (
            <Tooltip title="To make backups locally and restore them, you need to have enough space on your disk. For restore, you need to have same amount of space that the backup size.">
              <div
                className={`cursor-pointer text-center text-xs ${isUsedMoreThan95Percent ? 'text-red-500' : 'text-gray-500 dark:text-gray-400'}`}
              >
                {(diskUsage.usedSpaceBytes / 1024 ** 3).toFixed(1)} of{' '}
                {(diskUsage.totalSpaceBytes / 1024 ** 3).toFixed(1)} GB
                <br />
                ROM used (
                {((diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes) * 100).toFixed(1)}%)
              </div>
            </Tooltip>
          )}

          <ThemeToggleComponent />
        </div>

        <div className="ml-auto flex items-center gap-2 md:hidden">
          <Button
            type="text"
            icon={<MenuOutlined style={{ fontSize: '20px' }} />}
            onClick={() => setIsSidebarOpen(!isSidebarOpen)}
            className="mt-1"
          />
        </div>
      </div>
      {isLoading ? (
        <div className="flex items-center justify-center py-2" style={{ height: contentHeight }}>
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        </div>
      ) : (
        <div className="relative flex">
          <SidebarComponent
            isOpen={isSidebarOpen}
            onClose={() => setIsSidebarOpen(false)}
            selectedTab={selectedTab}
            tabs={tabs}
            user={user}
            diskUsage={diskUsage}
            contentHeight={contentHeight}
          />

          {selectedTab === 'profile' && <ProfileComponent contentHeight={contentHeight} />}

          {selectedTab === 'databasus-settings' && (
            <SettingsComponent contentHeight={contentHeight} />
          )}

          {selectedTab === 'users' && <UsersComponent contentHeight={contentHeight} />}

          {(selectedTab === 'databases' ||
            selectedTab === 'storages' ||
            selectedTab === 'notifiers' ||
            selectedTab === 'settings') && (
            <>
              {workspaces.length === 0 ? (
                <div className="flex-1 md:pl-3">
                  <div
                    className="flex grow items-center justify-center rounded"
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
                </div>
              ) : (
                <>
                  <div className="flex-1 md:pl-3">
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
                  </div>
                </>
              )}
            </>
          )}

          <div className="absolute bottom-1 left-2 mb-[0px] hidden text-sm text-gray-400 md:block">
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
