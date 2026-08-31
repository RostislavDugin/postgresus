import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "Как использовать Databasus с Cloudflare R2 | Databasus",
  description:
    "Пошаговое руководство по настройке хранилища Cloudflare R2 для бекапов PostgreSQL в Databasus. Как настроить S3-совместимое хранилище на базе R2.",
  keywords: [
    "Databasus",
    "Cloudflare R2",
    "резервное копирование PostgreSQL",
    "хранилище S3",
    "облачное хранилище",
    "бекап базы данных",
  ],
  openGraph: {
    title: "Как использовать Databasus с Cloudflare R2 | Databasus",
    description:
      "Пошаговое руководство по настройке хранилища Cloudflare R2 для бекапов PostgreSQL в Databasus. Как настроить S3-совместимое хранилище на базе R2.",
    type: "article",
    url: getLocalizedUrl("ru", "storages/cloudflare-r2"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Как использовать Databasus с Cloudflare R2 | Databasus",
    description:
      "Пошаговое руководство по настройке хранилища Cloudflare R2 для бекапов PostgreSQL в Databasus. Как настроить S3-совместимое хранилище на базе R2.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "storages/cloudflare-r2"),
    languages: getLanguageAlternates("storages/cloudflare-r2"),
  },
  robots: "index, follow",
};

export default function CloudflareR2Page() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Как использовать Databasus с Cloudflare R2",
            description:
              "Пошаговое руководство по настройке хранилища Cloudflare R2 для бекапов PostgreSQL в Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Укажите имя бакета",
                text: "Введите имя вашего бакета R2 в настройках хранилища.",
              },
              {
                "@type": "HowToStep",
                name: "Задайте регион",
                text: 'В поле региона укажите "auto"',
              },
              {
                "@type": "HowToStep",
                name: "Сгенерируйте access key ID и secret access key",
                text: "В панели Cloudflare перейдите в R2 → API → Manage API Tokens. Создайте токен и выдайте ему нужные права.",
              },
              {
                "@type": "HowToStep",
                name: "Найдите ID аккаунта",
                text: "Account ID показан вверху любой страницы R2 в панели Cloudflare.",
              },
              {
                "@type": "HowToStep",
                name: "Соберите S3 endpoint",
                text: "Подставьте значение <ACCOUNT_ID> из панели в формат: https://<ACCOUNT_ID>.r2.cloudflarestorage.com",
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
              <h1 id="cloudflare-r2">Хранилище Cloudflare R2</h1>

              <p className="text-lg text-gray-400">
                Чтобы использовать Cloudflare R2 как S3-совместимое хранилище
                для бекапов PostgreSQL, нужно указать учетные данные вашего
                бакета R2 и endpoint.
              </p>

              <h2 id="configuration-steps">Шаги настройки</h2>

              <h3 id="fill-bucket-name">1. Укажите имя бакета</h3>

              <p>Введите имя вашего бакета R2 в настройках хранилища:</p>

              <Image
                src="/images/cloudflare-r2-storage/image-1.webp"
                alt="Указание имени бакета в Cloudflare R2"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="set-region">2. Задайте регион</h3>

              <p>
                В поле региона укажите <code>&quot;auto&quot;</code>
              </p>

              <h3 id="generate-access-key">
                3. Сгенерируйте Access Key ID и Secret Access Key
              </h3>

              <p>
                В панели Cloudflare перейдите в{" "}
                <strong>R2 → API → Manage API Tokens</strong>. Создайте новый
                токен и выдайте ему нужные права (например,{" "}
                <strong>&quot;Object Read &amp; Write&quot;</strong>).
              </p>

              <p>После создания токена вы увидите:</p>

              <ul>
                <li>
                  <strong>Access Key ID</strong> (ID токена)
                </li>
                <li>
                  <strong>Secret Access Key</strong> (SHA-256-хеш значения
                  токена)
                </li>
              </ul>

              <p>Скопируйте оба значения в Databasus:</p>

              <Image
                src="/images/cloudflare-r2-storage/image-2.gif"
                alt="Генерация Access Key ID и Secret Access Key"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="find-account-id">4. Найдите ID аккаунта</h3>

              <p>
                Ваш Account ID показан вверху любой страницы R2 в панели (или в
                настройках аккаунта):
              </p>

              <Image
                src="/images/cloudflare-r2-storage/image-3.webp"
                alt="Поиск Account ID в панели Cloudflare"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="construct-endpoint">5. Соберите S3 endpoint</h3>

              <p>Используйте следующий формат S3 endpoint:</p>

              <pre>
                <code>https://&lt;ACCOUNT_ID&gt;.r2.cloudflarestorage.com</code>
              </pre>

              <p>
                Замените <code>&lt;ACCOUNT_ID&gt;</code> на значение из вашей
                панели и введите его в Databasus.
              </p>

              <p>Готово! Настройка должна выглядеть так:</p>

              <Image
                src="/images/cloudflare-r2-storage/image-4.png"
                alt="Настройка завершена"
                width={500}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                Теперь Databasus готов использовать Cloudflare R2 как хранилище
                для ваших бекапов PostgreSQL.
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
