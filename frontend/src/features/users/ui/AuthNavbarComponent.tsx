import { useTranslation } from 'react-i18next';

import { getWebsitePageUrl, useLocale } from '../../../shared/i18n';
import { LanguageThemeControlComponent } from '../../../shared/ui';
import { StarButtonComponent } from '../../../shared/ui/StarButtonComponent';

export function AuthNavbarComponent() {
  const { t } = useTranslation();
  const { locale } = useLocale();

  const homePageUrl = getWebsitePageUrl('home', locale);

  return (
    <div className="flex h-[65px] items-center justify-center px-5 pt-5 sm:justify-start">
      <div className="flex items-center gap-3 hover:opacity-80">
        <a href={homePageUrl} target="_blank" rel="noreferrer">
          <img className="h-[45px] w-[45px] p-1" src="/logo.svg" />
        </a>

        <div className="text-xl font-bold">
          <a href={homePageUrl} className="!text-blue-600" target="_blank" rel="noreferrer">
            {/* eslint-disable-next-line i18next/no-literal-string -- product name, the same in every language */}
            {'Databasus'}
          </a>
        </div>
      </div>

      <div className="mr-3 ml-auto hidden items-center gap-5 sm:flex">
        <a
          className="!text-black hover:opacity-80 dark:!text-gray-200"
          href={getWebsitePageUrl('installation', locale)}
          target="_blank"
          rel="noreferrer"
        >
          {t('app.navigation.docs')}
        </a>

        <a
          className="!text-black hover:opacity-80 dark:!text-gray-200"
          href="https://t.me/databasus_community"
          target="_blank"
          rel="noreferrer"
        >
          {t('app.navigation.community')}
        </a>

        <div className="flex items-center gap-2">
          <StarButtonComponent />

          <LanguageThemeControlComponent />
        </div>
      </div>
    </div>
  );
}
