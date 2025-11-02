import GitHubButton from 'react-github-btn';

export function AuthNavbarComponent() {
  return (
    <div className="m-3 flex h-[65px] items-center justify-center p-3 sm:justify-start">
      <div className="flex items-center gap-3 hover:opacity-80">
        <a href="https://postgresus.com" target="_blank" rel="noreferrer">
          <img className="h-[35px] w-[35px]" src="/logo.svg" />
        </a>

        <div className="text-xl font-bold">
          <a href="https://postgresus.com" className="!text-black" target="_blank" rel="noreferrer">
            Postgresus
          </a>
        </div>
      </div>

      <div className="mr-3 ml-auto hidden items-center gap-5 sm:flex">
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
            aria-label="Star Postgresus on GitHub"
          >
            &nbsp;Star on GitHub
          </GitHubButton>
        </div>
      </div>
    </div>
  );
}
