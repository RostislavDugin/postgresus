import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Сброс пароля - документация Databasus",
  description:
    "Как сбросить пароль пользователя в Databasus встроенной консольной утилитой. Быстрый и безопасный сброс пароля в вашей системе резервного копирования PostgreSQL.",
  keywords: [
    "сброс пароля Databasus",
    "сбросить пароль пользователя",
    "пароль системы бекапов PostgreSQL",
    "восстановление пароля в Docker",
    "восстановление пароля",
    "аутентификация Databasus",
  ],
  openGraph: {
    title: "Сброс пароля - документация Databasus",
    description:
      "Как сбросить пароль пользователя в Databasus встроенной консольной утилитой. Быстрый и безопасный сброс пароля в вашей системе резервного копирования PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("ru", "password"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Сброс пароля - документация Databasus",
    description:
      "Как сбросить пароль пользователя в Databasus встроенной консольной утилитой. Быстрый и безопасный сброс пароля в вашей системе резервного копирования PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "password"),
    languages: getLanguageAlternates("password"),
  },
  robots: "index, follow",
};

export default function PasswordResetPage() {
  const resetPasswordCommand = `docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="admin"`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Сброс пароля - документация Databasus",
            description:
              "Как сбросить пароль пользователя в Databasus встроенной консольной утилитой.",
            author: {
              "@type": "Organization",
              name: "Databasus",
            },
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: {
                "@type": "ImageObject",
                url: "https://databasus.com/logo.svg",
              },
            },
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Как сбросить пароль пользователя Databasus",
            description:
              "Пошаговое руководство по сбросу пароля пользователя в Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Выполните команду сброса пароля",
                text: "Выполните команду docker exec с новым паролем и email пользователя.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Запустите команду сброса пароля внутри контейнера Databasus через docker exec",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Проверьте смену пароля",
                text: "Войдите в Databasus с новым паролем и убедитесь, что смена прошла успешно.",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="ru" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="ru" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="reset-password">Сброс пароля пользователя</h1>

              <h2 id="reset-password-command">Команда сброса пароля</h2>

              <p>
                Чтобы сбросить пароль пользователя, выполните следующую команду
                на сервере, где запущен Databasus:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={resetPasswordCommand} lang="ru" />
                </div>
              </div>

              <h2 id="parameters">Параметры</h2>

              <p>Команда принимает следующие параметры:</p>

              <ul>
                <li>
                  <strong>--new-password</strong>: новый пароль. Убедитесь, что
                  он надежный и содержит буквы, цифры и специальные символы.
                </li>
                <li>
                  <strong>--email</strong>: email пользователя, чей пароль вы
                  хотите сбросить (например, <code>admin</code>,{" "}
                  <code>user@example.com</code>).
                </li>
              </ul>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
