import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "Как подключить Google Drive к Databasus | Databasus",
  description:
    "Пошаговое руководство по настройке хранилища Google Drive для бекапов PostgreSQL в Databasus. Как создать проект Google Cloud и настроить OAuth.",
  keywords: [
    "Databasus",
    "Google Drive",
    "резервное копирование PostgreSQL",
    "Google Cloud",
    "OAuth",
    "облачное хранилище",
    "бекап базы данных",
  ],
  openGraph: {
    title: "Как подключить Google Drive к Databasus | Databasus",
    description:
      "Пошаговое руководство по настройке хранилища Google Drive для бекапов PostgreSQL в Databasus. Как создать проект Google Cloud и настроить OAuth.",
    type: "article",
    url: getLocalizedUrl("ru", "storages/google-drive"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Как подключить Google Drive к Databasus | Databasus",
    description:
      "Пошаговое руководство по настройке хранилища Google Drive для бекапов PostgreSQL в Databasus. Как создать проект Google Cloud и настроить OAuth.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "storages/google-drive"),
    languages: getLanguageAlternates("storages/google-drive"),
  },
  robots: "index, follow",
};

export default function GoogleDrivePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Как подключить Google Drive к Databasus",
            description:
              "Пошаговое руководство по настройке хранилища Google Drive для бекапов PostgreSQL в Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Создайте новый проект",
                text: "Откройте Google Cloud Console и создайте новый проект.",
              },
              {
                "@type": "HowToStep",
                name: "Включите Google Drive API",
                text: "Перейдите на вкладку API & Services, затем в API library и включите Google Drive API.",
              },
              {
                "@type": "HowToStep",
                name: "Настройте экран согласия",
                text: "Перейдите в Credentials → Create credentials → Configure consent screen и заполните обязательные поля.",
              },
              {
                "@type": "HowToStep",
                name: "Создайте OAuth client ID",
                text: "Перейдите в Credentials → Create credentials → OAuth client ID.",
              },
              {
                "@type": "HowToStep",
                name: "Настройте параметры приложения",
                text: "Выберите тип приложения Web application и задайте authorized origins и redirect URIs.",
              },
              {
                "@type": "HowToStep",
                name: "Добавьте scope",
                text: 'Перейдите в Data Access и добавьте scope "/auth/drive.file".',
              },
              {
                "@type": "HowToStep",
                name: "Опубликуйте приложение",
                text: "Перейдите в Audience и опубликуйте приложение.",
              },
              {
                "@type": "HowToStep",
                name: "Войдите через аккаунт Google",
                text: "Внесите в Databasus учетные данные и войдите через свой аккаунт Google.",
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
              <h1 id="google-drive">Хранилище Google Drive</h1>

              <p className="text-lg text-gray-400">
                Чтобы хранить бекапы в Google Drive, нужно создать проект Google
                Cloud для доступа к Google Drive API, а затем войти через свой
                аккаунт Google.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Важно:</strong> для
                  подключения Google Drive ваш экземпляр Databasus должен
                  работать по HTTPS. Как настроить HTTPS, читайте в{" "}
                  <a
                    href="/ru/installation#caddy-reverse-proxy"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    руководстве по настройке Caddy reverse proxy →
                  </a>
                  <br />В этом руководстве мы будем считать, что ваш Databasus
                  доступен по адресу <code>databasus.yourdomain.com</code>.
                </p>
              </div>

              <h2 id="create-google-cloud-project">
                Создайте проект Google Cloud
              </h2>

              <h3 id="create-new-project">1. Создайте новый проект</h3>

              <p>
                Откройте{" "}
                <a
                  href="https://console.cloud.google.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://console.cloud.google.com/
                </a>{" "}
                и выберите <strong>&quot;new project&quot;</strong> (сверху
                слева).
              </p>

              <h3 id="enable-google-drive-api">2. Включите Google Drive API</h3>

              <p>
                Перейдите на вкладку{" "}
                <strong>&quot;API &amp; Services&quot;</strong>, затем в{" "}
                <strong>&quot;API library&quot;</strong>. Выберите{" "}
                <strong>Google Drive API</strong> и включите его:
              </p>

              <Image
                src="/images/google-drive-storage/image-1.webp"
                alt="Включение Google Drive API"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-consent-screen">3. Настройте экран согласия</h3>

              <p>
                Перейдите в <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;Configure consent screen&quot;</strong> и
                заполните поля:
              </p>

              <Image
                src="/images/google-drive-storage/image-2.webp"
                alt="Настройка экрана согласия"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-oauth-client-id">4. Создайте OAuth client ID</h3>

              <p>
                Перейдите в <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;OAuth client ID&quot;</strong>:
              </p>

              <Image
                src="/images/google-drive-storage/image-3.webp"
                alt="Создание OAuth client ID"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-application-settings">
                5. Настройте параметры приложения
              </h3>

              <p>Заполните следующие поля:</p>

              <ul>
                <li>
                  <strong>Application type:</strong> Web application
                </li>
                <li>
                  <strong>Authorized JavaScript origins:</strong>{" "}
                  <code>https://databasus.yourdomain.com</code>
                </li>
                <li>
                  <strong>Authorized redirect URIs:</strong>{" "}
                  <code>
                    https://databasus.yourdomain.com/storages/google-oauth
                  </code>
                </li>
              </ul>

              <p>
                <strong>Примечание:</strong> замените{" "}
                <code>databasus.yourdomain.com</code> на реальный домен, где
                размещен ваш Databasus (например,{" "}
                <code>backup.example.com</code>).
              </p>

              <p>Затем скопируйте учетные данные:</p>

              <Image
                src="/images/google-drive-storage/image-4.png"
                alt="Настройка параметров приложения — часть 1"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <Image
                src="/images/google-drive-storage/image-5.png"
                alt="Настройка параметров приложения — часть 2"
                width={450}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="add-scope">6. Добавьте scope</h3>

              <p>
                Перейдите в <strong>&quot;Data Access&quot;</strong> и добавьте
                scope <code>&quot;/auth/drive.file&quot;</code>:
              </p>

              <Image
                src="/images/google-drive-storage/image-6.png"
                alt="Добавление scope"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="publish-app">7. Опубликуйте приложение</h3>

              <p>
                Перейдите в <strong>&quot;Audience&quot;</strong> и опубликуйте
                приложение:
              </p>

              <Image
                src="/images/google-drive-storage/image-7.png"
                alt="Публикация приложения"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="sign-in-google-account">Войдите через аккаунт Google</h2>

              <h3 id="fill-credentials">1. Заполните учетные данные</h3>

              <p>
                Внесите в Databasus учетные данные, полученные на предыдущих
                шагах:
              </p>

              <Image
                src="/images/google-drive-storage/image-8.png"
                alt="Заполнение учетных данных"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="choose-account">2. Выберите аккаунт</h3>

              <p>Выберите аккаунт Google для входа.</p>

              <h3 id="handle-security-warning">
                3. Обойдите предупреждение безопасности
              </h3>

              <p>
                Если появится предупреждение, нажмите{" "}
                <strong>&quot;Advanced&quot;</strong> (внизу слева) и выберите{" "}
                <strong>&quot;Proceed anyway&quot;</strong>.
              </p>

              <p>
                <strong>Примечание:</strong> предупреждение появляется потому,
                что приложение еще не проверено Google. Это ваше собственное
                приложение, так что продолжать безопасно.
              </p>

              <p>
                Готово! Google Drive подключен к Databasus и готов хранить ваши
                бекапы PostgreSQL.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/ru/storages"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Назад к хранилищам
                </a>
              </div>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
