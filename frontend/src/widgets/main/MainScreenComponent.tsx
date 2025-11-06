import { Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import GitHubButton from 'react-github-btn';
import { useTranslation } from 'react-i18next';

import { APP_VERSION, getApplicationServer } from '../../constants';
import { type DiskUsage, diskApi } from '../../entity/disk';
import { DatabasesComponent } from '../../features/databases/ui/DatabasesComponent';
import { NotifiersComponent } from '../../features/notifiers/ui/NotifiersComponent';
import { StoragesComponent } from '../../features/storages/StoragesComponent';
import { useScreenHeight } from '../../shared/hooks';
import { LanguageSwitcher } from '../../shared/ui/LanguageSwitcher';

export const MainScreenComponent = () => {
  const { t } = useTranslation('common');
  const screenHeight = useScreenHeight();
  const contentHeight = screenHeight - 95;

  const [selectedTab, setSelectedTab] = useState<'notifiers' | 'storages' | 'databases'>(
    'databases',
  );
  const [diskUsage, setDiskUsage] = useState<DiskUsage | undefined>(undefined);

  useEffect(() => {
    diskApi
      .getDiskUsage()
      .then((diskUsage) => {
        setDiskUsage(diskUsage);
      })
      .catch((error) => {
        alert(error.message);
      });
  }, []);

  const isUsedMoreThan95Percent =
    diskUsage && diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes > 0.95;

  return (
    <div style={{ height: screenHeight }} className="bg-[#f5f5f5] p-3">
      {/* ===================== NAVBAR ===================== */}
      <div className="mb-3 flex h-[60px] items-center rounded bg-white p-3 shadow">
        <div className="flex items-center gap-3 hover:opacity-80">
          <a href="https://postgresus.com" target="_blank" rel="noreferrer">
            <img className="h-[35px] w-[35px]" src="/logo.svg" />
          </a>

          <div className="text-xl font-bold">
            <a
              href="https://postgresus.com"
              className="text-black"
              target="_blank"
              rel="noreferrer"
            >
              Postgresus
            </a>
          </div>
        </div>

        <div className="mr-3 ml-auto flex items-center gap-5">
          <LanguageSwitcher />

          <a
            className="hover:opacity-80"
            href={`${getApplicationServer()}/api/v1/system/health`}
            target="_blank"
            rel="noreferrer"
          >
            {t('navigation.health_check')}
          </a>

          <a
            className="hover:opacity-80"
            href="https://t.me/postgresus_community"
            target="_blank"
            rel="noreferrer"
          >
            {t('navigation.community')}
          </a>

          <div className="mt-1">
            <GitHubButton
              href="https://github.com/RostislavDugin/postgresus"
              data-icon="octicon-star"
              data-size="large"
              data-show-count="true"
              aria-label="Star RostislavDugin/postgresus on GitHub"
            >
              &nbsp;{t('navigation.star_on_github')}
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

      <div className="relative flex">
        <div
          className="max-w-[60px] min-w-[60px] rounded bg-white py-2 shadow"
          style={{ height: contentHeight }}
        >
          {[
            {
              text: t('navigation.databases'),
              name: 'databases',
              icon: '/icons/menu/database-gray.svg',
              selectedIcon: '/icons/menu/database-white.svg',
              onClick: () => setSelectedTab('databases'),
            },
            {
              text: t('navigation.storages'),
              name: 'storages',
              icon: '/icons/menu/storage-gray.svg',
              selectedIcon: '/icons/menu/storage-white.svg',
              onClick: () => setSelectedTab('storages'),
            },
            {
              text: t('navigation.notifiers'),
              name: 'notifiers',
              icon: '/icons/menu/notifier-gray.svg',
              selectedIcon: '/icons/menu/notifier-white.svg',
              onClick: () => setSelectedTab('notifiers'),
            },
          ].map((tab) => (
            <div key={tab.text} className="flex justify-center">
              <div
                className={`flex h-[50px] w-[50px] cursor-pointer items-center justify-center rounded ${selectedTab === tab.name ? 'bg-blue-600' : 'hover:bg-blue-50'}`}
                onClick={tab.onClick}
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

        {selectedTab === 'notifiers' && <NotifiersComponent contentHeight={contentHeight} />}
        {selectedTab === 'storages' && <StoragesComponent contentHeight={contentHeight} />}
        {selectedTab === 'databases' && <DatabasesComponent contentHeight={contentHeight} />}

        <div className="absolute bottom-1 left-2 mb-[0px] text-sm text-gray-400">
          v{APP_VERSION}
        </div>
      </div>
    </div>
  );
};
