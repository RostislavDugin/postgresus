import GitHubButton from 'react-github-btn';
import { useTranslation } from 'react-i18next';

export function AuthNavbarComponent() {
  const { t } = useTranslation(['common']);

  return (
    <div className="m-3 flex h-[65px] items-center p-3">
      <div className="flex items-center gap-3 hover:opacity-80">
        <a href="https://postgresus.com" target="_blank" rel="noreferrer">
          <img className="h-[35px] w-[35px]" src="/logo.svg" />
        </a>

        <div className="text-xl font-bold">
          <a href="https://postgresus.com" className="text-black" target="_blank" rel="noreferrer">
            Postgresus
          </a>
        </div>
      </div>

      <div className="mr-3 ml-auto flex items-center gap-5">
        <a
          className="hover:opacity-80"
          href="https://postgresus.com/community"
          target="_blank"
          rel="noreferrer"
        >
          {t('common:navigation.community')}
        </a>

        <div className="mt-1">
          <GitHubButton
            href="https://github.com/RostislavDugin/postgresus"
            data-icon="octicon-star"
            data-size="large"
            data-show-count="true"
            aria-label="Star RostislavDugin/postgresus on GitHub"
          >
            &nbsp;{t('common:navigation.star_on_github')}
          </GitHubButton>
        </div>
      </div>
    </div>
  );
}
