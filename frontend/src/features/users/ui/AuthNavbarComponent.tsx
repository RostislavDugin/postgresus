import GitHubButton from 'react-github-btn';

import { ThemeToggleComponent } from '../../../widgets/main/ThemeToggleComponent';

export function AuthNavbarComponent() {
  return (
    <div className="flex h-[65px] items-center justify-center px-5 pt-5 sm:justify-start">
      <div className="flex items-center gap-3 hover:opacity-80">
        <a href="https://databasus.com" target="_blank" rel="noreferrer">
          <img className="h-[45px] w-[45px]" src="/logo.svg" />
        </a>

        <div className="text-xl font-bold">
          <a
            href="https://databasus.com"
            className="!text-blue-600"
            target="_blank"
            rel="noreferrer"
          >
            Databasus
          </a>
        </div>
      </div>

      <div className="mr-3 ml-auto hidden items-center gap-5 sm:flex">
        <a
          className="!text-black hover:opacity-80 dark:!text-gray-200"
          href="https://t.me/databasus_community"
          target="_blank"
          rel="noreferrer"
        >
          Community
        </a>

        <div className="mt-[7px]">
          <GitHubButton
            href="https://github.com/databasus/databasus"
            data-icon="octicon-star"
            data-size="large"
            data-show-count="true"
            aria-label="Star Databasus on GitHub"
          >
            &nbsp;Star on GitHub
          </GitHubButton>
        </div>

        <ThemeToggleComponent />
      </div>
    </div>
  );
}
