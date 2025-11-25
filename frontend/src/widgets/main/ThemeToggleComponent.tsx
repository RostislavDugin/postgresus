import { Dropdown } from 'antd';
import type { MenuProps } from 'antd';

import { type ThemeMode, useTheme } from '../../shared/theme';

const SunIcon = () => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="18"
    height="18"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <circle cx="12" cy="12" r="4" />
    <path d="M12 2v2" />
    <path d="M12 20v2" />
    <path d="m4.93 4.93 1.41 1.41" />
    <path d="m17.66 17.66 1.41 1.41" />
    <path d="M2 12h2" />
    <path d="M20 12h2" />
    <path d="m6.34 17.66-1.41 1.41" />
    <path d="m19.07 4.93-1.41 1.41" />
  </svg>
);

const MoonIcon = () => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="18"
    height="18"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z" />
  </svg>
);

const SystemIcon = () => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="18"
    height="18"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <rect width="20" height="14" x="2" y="3" rx="2" />
    <line x1="8" x2="16" y1="21" y2="21" />
    <line x1="12" x2="12" y1="17" y2="21" />
  </svg>
);

export function ThemeToggleComponent() {
  const { theme, setTheme, resolvedTheme } = useTheme();

  const items: MenuProps['items'] = [
    {
      key: 'light',
      label: (
        <div className="flex items-center gap-2">
          <SunIcon />
          <span>Light</span>
        </div>
      ),
      onClick: () => setTheme('light'),
    },
    {
      key: 'dark',
      label: (
        <div className="flex items-center gap-2">
          <MoonIcon />
          <span>Dark</span>
        </div>
      ),
      onClick: () => setTheme('dark'),
    },
    {
      key: 'system',
      label: (
        <div className="flex items-center gap-2">
          <SystemIcon />
          <span>System</span>
        </div>
      ),
      onClick: () => setTheme('system'),
    },
  ];

  const getCurrentIcon = () => {
    if (theme === 'system') {
      return <SystemIcon />;
    }
    return resolvedTheme === 'dark' ? <MoonIcon /> : <SunIcon />;
  };

  const getLabel = (mode: ThemeMode) => {
    switch (mode) {
      case 'light':
        return 'Light';
      case 'dark':
        return 'Dark';
      case 'system':
        return 'System';
    }
  };

  return (
    <Dropdown menu={{ items, selectedKeys: [theme] }} trigger={['click']} placement="bottomRight">
      <button
        className="flex cursor-pointer items-center gap-1.5 rounded-md border border-gray-200 bg-white px-2.5 py-1 text-sm text-gray-700 transition-colors hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600"
        title={`Theme: ${getLabel(theme)}`}
      >
        {getCurrentIcon()}
        <span className="hidden sm:inline">{getLabel(theme)}</span>
      </button>
    </Dropdown>
  );
}
