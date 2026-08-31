import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Как делать бекапы Supabase с Databasus | Databasus",
  description:
    "Как настроить резервное копирование PostgreSQL в Supabase с помощью Databasus. Пошаговое руководство по настройке session pooler или IPv4-адреса для бекапов Supabase.",
  keywords: [
    "Databasus",
    "бекап Supabase",
    "резервное копирование PostgreSQL в Supabase",
    "бекап базы данных Supabase",
    "session pooler Supabase",
    "IPv4 Supabase",
    "резервное копирование PostgreSQL",
    "бекап базы данных",
  ],
  openGraph: {
    title: "Как делать бекапы Supabase с Databasus | Databasus",
    description:
      "Как настроить резервное копирование PostgreSQL в Supabase с помощью Databasus. Пошаговое руководство по настройке session pooler или IPv4-адреса для бекапов Supabase.",
    type: "article",
    url: getLocalizedUrl("ru", "faq/supabase"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Как делать бекапы Supabase с Databasus | Databasus",
    description:
      "Как настроить резервное копирование PostgreSQL в Supabase с помощью Databasus. Пошаговое руководство по настройке session pooler или IPv4-адреса для бекапов Supabase.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "faq/supabase"),
    languages: getLanguageAlternates("faq/supabase"),
  },
  robots: "index, follow",
};

export default function SupabasePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Как делать бекапы Supabase с Databasus",
            description:
              "Пошаговое руководство по резервному копированию базы данных PostgreSQL в Supabase с помощью Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Получите данные подключения из Supabase",
                text: "Откройте настройки проекта Supabase и найдите данные подключения к базе данных.",
              },
              {
                "@type": "HowToStep",
                name: "Используйте Session Pooler с IPv4",
                text: "Скопируйте строку подключения Session Pooler и убедитесь, что опция 'Use IPv4 Address' включена.",
              },
              {
                "@type": "HowToStep",
                name: "Настройте Databasus",
                text: "Введите данные подключения Supabase в Databasus, чтобы начать резервное копирование базы данных.",
              },
              {
                "@type": "HowToStep",
                name: "Учитывайте ограничения по схемам",
                text: "По умолчанию копируется только схема public, так как остальные схемы Supabase закрыты.",
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
              <h1 id="supabase-backup">Как делать бекапы Supabase</h1>

              <p className="text-lg text-gray-400">
                Databasus поддерживает резервное копирование баз данных
                PostgreSQL в Supabase. Главное требование: для подключения к
                вашей базе Supabase нужен IPv4-адрес.
              </p>

              <h2 id="connection-options">Варианты подключения</h2>

              <p>
                Подключить Databasus к базе данных Supabase можно двумя
                способами:
              </p>

              <ol>
                <li>
                  <strong>Session Pooler с IPv4</strong> (рекомендуется) —
                  бесплатный вариант, доступный во всех проектах Supabase
                </li>
                <li>
                  <strong>Платное дополнение IPv4</strong> — прямое подключение
                  от Supabase
                </li>
              </ol>

              <h2 id="session-pooler">
                Вариант 1: Session Pooler (рекомендуется)
              </h2>

              <p>
                Session Pooler бесплатно дает IPv4-адрес для подключения к базе
                данных Supabase. Вот как его настроить:
              </p>

              <h3 id="step-1">1. Найдите подключение через пулер</h3>

              <p>
                Откройте проект Supabase, перейдите в{" "}
                <strong>Project Settings</strong> → <strong>Database</strong>.
                Прокрутите вниз до секции <strong>Connection string</strong> и
                выберите режим <strong>Session pooler</strong>.
              </p>

              <img
                src="/images/faq/supabase/image-1.png"
                alt="Выбор режима Session pooler в Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h3 id="step-2">2. Скопируйте данные подключения</h3>

              <p>
                Скопируйте данные подключения и используйте их в Databasus при
                добавлении базы данных. На скриншоте видно, где какая часть
                данных подключения.
              </p>

              <img
                src="/images/faq/supabase/image-2.png"
                alt="Включение опции IPv4 Address в Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h2 id="ipv4-addon">Вариант 2: платное дополнение IPv4</h2>

              <p>
                Supabase предлагает платное дополнение IPv4, которое дает
                выделенный IPv4-адрес для вашей базы данных. Так вы получаете
                прямое подключение в обход пулера соединений.
              </p>

              <p>Чтобы включить этот вариант:</p>

              <ol>
                <li>Откройте панель проекта Supabase</li>
                <li>
                  Перейдите в <strong>Project Settings</strong> →{" "}
                  <strong>Add-ons</strong>
                </li>
                <li>
                  Включите дополнение <strong>IPv4</strong>
                </li>
                <li>
                  Используйте в Databasus данные прямого подключения к базе
                </li>
              </ol>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Совет:</strong> в
                  большинстве случаев для бекапов достаточно бесплатного Session
                  Pooler с IPv4. Платное дополнение IPv4 нужно только тогда,
                  когда прямое подключение требуется по другим причинам.
                </p>
              </div>

              <h2 id="default-schema">Какая схема копируется по умолчанию</h2>

              <p>
                По умолчанию при работе с базами данных Supabase Databasus
                копирует только схему <code>public</code>. Дело в том, что
                Supabase из соображений безопасности ограничивает доступ к
                другим схемам (таким как <code>auth</code>, <code>storage</code>{" "}
                и <code>realtime</code>).
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-blue-400">ℹ️ Примечание:</strong>{" "}
                  схема <code>public</code> содержит данные вашего приложения и
                  пользовательские таблицы. Служебные схемы Supabase, такие как{" "}
                  <code>auth</code> и <code>storage</code>, защищены и
                  управляются самим Supabase.
                </p>
              </div>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/ru/faq"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Назад к FAQ
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
