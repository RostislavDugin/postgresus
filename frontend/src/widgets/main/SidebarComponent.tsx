import { CloseOutlined } from '@ant-design/icons';
import { Drawer, Tooltip } from 'antd';
import { useEffect } from 'react';
import GitHubButton from 'react-github-btn';

import { type DiskUsage } from '../../entity/disk';
import { type UserProfile, UserRole } from '../../entity/users';
import { useIsMobile } from '../../shared/hooks';

interface TabItem {
  text: string;
  name: string;
  icon: string;
  selectedIcon: string;
  onClick: () => void;
  isAdminOnly: boolean;
  marginTop: string;
  isVisible: boolean;
}

interface Props {
  isOpen: boolean;
  onClose: () => void;
  selectedTab: string;
  tabs: TabItem[];
  user?: UserProfile;
  diskUsage?: DiskUsage;
  contentHeight: number;
}

export const SidebarComponent = ({
  isOpen,
  onClose,
  selectedTab,
  tabs,
  user,
  diskUsage,
  contentHeight,
}: Props) => {
  const isMobile = useIsMobile();

  // Close sidebar on desktop when it becomes desktop size
  useEffect(() => {
    if (!isMobile && isOpen) {
      onClose();
    }
  }, [isMobile, isOpen, onClose]);

  // Prevent background scrolling when mobile sidebar is open
  useEffect(() => {
    if (isMobile && isOpen) {
      document.body.style.overflowY = 'hidden';
      return () => {
        document.body.style.overflowY = '';
      };
    }
  }, [isMobile, isOpen]);

  const isUsedMoreThan95Percent =
    diskUsage && diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes > 0.95;

  const filteredTabs = tabs
    .filter((tab) => !tab.isAdminOnly || user?.role === UserRole.ADMIN)
    .filter((tab) => tab.isVisible);

  const handleTabClick = (tab: TabItem) => {
    tab.onClick();
    if (isMobile) {
      onClose();
    }
  };

  if (!isMobile) {
    return (
      <div
        className="max-w-[60px] min-w-[60px] rounded bg-white py-2 shadow"
        style={{ height: contentHeight }}
      >
        <div className="flex h-full flex-col">
          <div className="flex-1">
            {filteredTabs.map((tab) => (
              <div key={tab.text} className="flex justify-center">
                <div
                  className={`flex h-[50px] w-[50px] cursor-pointer items-center justify-center rounded select-none ${selectedTab === tab.name ? 'bg-blue-600' : 'hover:bg-blue-50'}`}
                  onClick={() => handleTabClick(tab)}
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
        </div>
      </div>
    );
  }

  return (
    <Drawer
      open={isOpen}
      onClose={onClose}
      placement="right"
      width={280}
      styles={{
        body: { padding: 0 },
      }}
      closable={false}
      mask={false}
    >
      <div className="flex h-full flex-col">
        {/* Custom Close Button */}
        <div className="flex justify-end border-b border-gray-200 px-3 py-3">
          <button
            onClick={onClose}
            className="flex h-8 w-8 items-center justify-center rounded hover:bg-gray-100"
          >
            <CloseOutlined />
          </button>
        </div>

        {/* Navigation Tabs */}
        <div className="flex-1 overflow-y-auto px-3 py-4">
          {filteredTabs.map((tab, index) => {
            const showDivider =
              index < filteredTabs.length - 1 && filteredTabs[index + 1]?.marginTop !== '0px';

            return (
              <div key={tab.text}>
                <div
                  className={`flex cursor-pointer items-center gap-3 rounded px-3 py-3 select-none ${selectedTab === tab.name ? 'bg-blue-600 text-white' : 'text-gray-700 hover:bg-gray-100'}`}
                  onClick={() => handleTabClick(tab)}
                >
                  <img
                    src={selectedTab === tab.name ? tab.selectedIcon : tab.icon}
                    width={24}
                    alt={tab.text}
                    loading="lazy"
                  />
                  <span className="text-sm font-medium">{tab.text}</span>
                </div>
                {showDivider && <div className="my-2 border-t border-gray-200" />}
              </div>
            );
          })}
        </div>

        {/* Footer Section */}
        <div className="border-t border-gray-200 bg-gray-50 px-3 py-4">
          {diskUsage && (
            <div className="mb-4">
              <Tooltip title="To make backups locally and restore them, you need to have enough space on your disk. For restore, you need to have same amount of space that the backup size.">
                <div
                  className={`cursor-pointer text-xs ${isUsedMoreThan95Percent ? 'text-red-500' : 'text-gray-600'}`}
                >
                  <div className="font-medium">Disk Usage</div>
                  <div className="mt-1">
                    {(diskUsage.usedSpaceBytes / 1024 ** 3).toFixed(1)} of{' '}
                    {(diskUsage.totalSpaceBytes / 1024 ** 3).toFixed(1)} GB used (
                    {((diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes) * 100).toFixed(1)}%)
                  </div>
                </div>
              </Tooltip>
            </div>
          )}

          <div className="space-y-2">
            <a
              className="!hover:text-blue-600 block rounded text-sm font-medium !text-gray-700 hover:bg-gray-100"
              href="https://postgresus.com/installation"
              target="_blank"
              rel="noreferrer"
            >
              Documentation
            </a>

            <a
              className="!hover:text-blue-600 block rounded text-sm font-medium !text-gray-700 hover:bg-gray-100"
              href="https://postgresus.com/contribute"
              target="_blank"
              rel="noreferrer"
            >
              Contribute
            </a>

            <a
              className="!hover:text-blue-600 block rounded text-sm font-medium !text-gray-700 hover:bg-gray-100"
              href="https://t.me/postgresus_community"
              target="_blank"
              rel="noreferrer"
            >
              Community
            </a>

            <div className="pt-2">
              <GitHubButton
                href="https://github.com/RostislavDugin/postgresus"
                data-icon="octicon-star"
                data-size="large"
                data-show-count="true"
                aria-label="Star RostislavDugin/postgresus on GitHub"
              >
                Star on GitHub
              </GitHubButton>
            </div>
          </div>
        </div>
      </div>
    </Drawer>
  );
};
