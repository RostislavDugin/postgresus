import { getLocalizedHref, type Locale } from "@/app/i18n";
import { LANDING_LABELS } from "./landingLabels";

export default function LandingFooterComponent({
  lang = "en",
}: {
  lang?: Locale | "en";
}) {
  const labels = LANDING_LABELS[lang];
  const homeHref = getLocalizedHref(lang, "");

  return (
    <footer className="py-8 md:py-12 border-t border-[#ffffff20] px-4 md:px-6 lg:px-0">
      <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
        <div className="flex flex-col items-center">
          <a href={homeHref} className="flex items-center gap-2.5 mb-6">
            <img
              src="/logo.svg"
              alt={labels.logoAlt}
              width={32}
              height={32}
              className="h-7 w-7 md:h-8 md:w-8"
            />

            <span className="text-base md:text-lg font-semibold">
              Databasus
            </span>
          </a>

          <div className="flex flex-col gap-3 mb-4 text-sm md:text-base">
            {/* First row - Database backup links */}
            <div className="flex flex-wrap items-center justify-center gap-4 md:gap-6">
              <a
                href={homeHref}
                className="hover:text-gray-200 transition-colors"
              >
                {labels.postgresqlBackup}
              </a>
              <a
                href={getLocalizedHref(lang, "mysql-backup")}
                className="hover:text-gray-200 transition-colors"
              >
                {labels.mysqlBackup}
              </a>
              <a
                href={getLocalizedHref(lang, "mongodb-backup")}
                className="hover:text-gray-200 transition-colors"
              >
                {labels.mongodbBackup}
              </a>
            </div>

            {/* Second row - General links */}
            <div className="flex flex-wrap items-center justify-center gap-4 md:gap-6">
              <a
                href={getLocalizedHref(lang, "installation")}
                className="hover:text-gray-200 transition-colors"
              >
                {labels.documentation}
              </a>
              <a
                href="https://github.com/databasus/databasus"
                target="_blank"
                rel="noopener noreferrer"
                className="hover:text-gray-200 transition-colors"
              >
                GitHub
              </a>
              <a
                href="https://t.me/databasus_community"
                target="_blank"
                rel="noopener noreferrer"
                className="hover:text-gray-200 transition-colors"
              >
                {labels.community}
              </a>
              <a
                href="https://rostislav-dugin.com"
                target="_blank"
                rel="noopener noreferrer"
                className="hover:text-gray-200 transition-colors"
              >
                {labels.developer}
              </a>
            </div>

            {/* Third row - Legal links */}
            <div className="flex flex-wrap items-center justify-center gap-4 md:gap-6">
              <a
                href="/privacy"
                className="hover:text-gray-200 transition-colors"
              >
                {labels.privacy}
              </a>
              <a
                href="/terms-of-use"
                className="hover:text-gray-200 transition-colors"
              >
                {labels.termsOfUse}
              </a>
            </div>
          </div>

          <a
            href="mailto:info@databasus.com"
            className="hover:text-gray-200 transition-colors text-sm md:text-base mb-4"
          >
            info@databasus.com
          </a>

          <p className="text-gray-400 text-sm md:text-base text-center">
            {labels.copyright}
          </p>
        </div>
      </div>
    </footer>
  );
}
