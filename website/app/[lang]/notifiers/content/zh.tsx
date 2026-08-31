import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "通知 - Databasus 文档",
  description:
    "Databasus 备份提醒支持的通知渠道列表，包括 Slack、Discord、Telegram、Microsoft Teams、邮件和 Webhook。",
  keywords: [
    "Databasus 通知",
    "备份通知",
    "Slack 通知",
    "Discord 提醒",
    "Telegram 通知",
    "Teams 通知",
    "邮件提醒",
    "Webhook 通知",
  ],
  openGraph: {
    title: "通知 - Databasus 文档",
    description:
      "Databasus 备份提醒支持的通知渠道列表，包括 Slack、Discord、Telegram、Microsoft Teams、邮件和 Webhook。",
    type: "article",
    url: getLocalizedUrl("zh", "notifiers"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "通知 - Databasus 文档",
    description:
      "Databasus 备份提醒支持的通知渠道列表，包括 Slack、Discord、Telegram、Microsoft Teams、邮件和 Webhook。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "notifiers"),
    languages: getLanguageAlternates("notifiers"),
  },
  robots: "index, follow",
};

export default function NotifiersPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "通知 - Databasus 文档",
            description:
              "Databasus 备份提醒支持的通知渠道列表，包括 Slack、Discord、Telegram、Microsoft Teams、邮件和 Webhook。",
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

      <DocsNavbarComponent lang="zh" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="zh" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="notifiers">通知</h1>

              <p className="text-lg text-gray-400">
                Databasus 支持多种通知渠道，让你随时掌握 PostgreSQL 备份
                的状态。备份成功、失败或出现问题时都会立刻收到提醒。
              </p>

              <h2 id="supported-notifiers">支持的通知渠道</h2>

              <ul>
                <li>
                  <a
                    href="/zh/notifiers/slack"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Slack
                  </a>{" "}
                  - 通过 webhook 向 Slack 频道发送通知
                </li>
                <li>
                  <strong>Discord</strong> - 向 Discord 频道发送备份提醒
                </li>
                <li>
                  <strong>Telegram</strong> - 通过 Telegram 机器人接收通知
                </li>
                <li>
                  <a
                    href="/zh/notifiers/teams"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Microsoft Teams
                  </a>{" "}
                  - 通过 Microsoft Teams 频道通知团队
                </li>
                <li>
                  <strong>邮件</strong> - 通过电子邮件发送备份事件通知
                </li>
                <li>
                  <strong>Webhook</strong> - 自定义 webhook 集成，接入任意服务
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
