import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "如何为 Databasus 配置 Microsoft Teams 通知 | Databasus",
  description:
    "在 Databasus 中为 PostgreSQL 备份提醒配置 Microsoft Teams 通知的分步指南。了解如何创建 Teams webhook 并配置通知。",
  keywords: [
    "Databasus",
    "Microsoft Teams 通知",
    "PostgreSQL 备份",
    "Teams webhook",
    "备份提醒",
    "数据库通知",
  ],
  openGraph: {
    title: "如何为 Databasus 配置 Microsoft Teams 通知 | Databasus",
    description:
      "在 Databasus 中为 PostgreSQL 备份提醒配置 Microsoft Teams 通知的分步指南。了解如何创建 Teams webhook 并配置通知。",
    type: "article",
    url: getLocalizedUrl("zh", "notifiers/teams"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "如何为 Databasus 配置 Microsoft Teams 通知 | Databasus",
    description:
      "在 Databasus 中为 PostgreSQL 备份提醒配置 Microsoft Teams 通知的分步指南。了解如何创建 Teams webhook 并配置通知。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "notifiers/teams"),
    languages: getLanguageAlternates("notifiers/teams"),
  },
  robots: "index, follow",
};

export default function TeamsPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "如何为 Databasus 配置 Microsoft Teams 通知",
            description:
              "在 Databasus 中为 PostgreSQL 备份提醒配置 Microsoft Teams 通知的分步指南",
            step: [
              {
                "@type": "HowToStep",
                name: "打开 Teams 频道",
                text: "打开你想接收通知的 Microsoft Teams 频道。",
              },
              {
                "@type": "HowToStep",
                name: "打开工作流",
                text: "在 Teams 频道中打开 Workflows 功能。",
              },
              {
                "@type": "HowToStep",
                name: "创建新工作流",
                text: "为传入 webhook 创建一个新工作流。",
              },
              {
                "@type": "HowToStep",
                name: "选择 webhook 模板",
                text: "从可用选项中选择传入 webhook 模板。",
              },
              {
                "@type": "HowToStep",
                name: "配置 webhook",
                text: "设置 webhook 的名称和频道。",
              },
              {
                "@type": "HowToStep",
                name: "复制 webhook URL",
                text: "复制 Teams 生成的 webhook URL。",
              },
              {
                "@type": "HowToStep",
                name: "在 Databasus 中配置",
                text: "将 webhook URL 粘贴到 Databasus 的通知配置中。",
              },
            ],
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
              <h1 id="teams-notifications">Microsoft Teams 通知</h1>

              <p className="text-lg text-gray-400">
                配置 Microsoft Teams，即时接收 PostgreSQL 备份的状态
                通知。备份成功、失败和警告都会直接推送到你的 Teams 频道。
              </p>

              <h2 id="setup-teams-webhook">配置 Teams webhook</h2>

              <h3 id="open-teams-channel">1. 打开 Teams 频道</h3>

              <p>
                打开你想接收备份通知的 Microsoft Teams 频道，点击频道
                名称旁边的三个点（<strong>•••</strong>）。
              </p>

              <Image
                src="/images/notifier-teams/image-01.png"
                alt="打开 Teams 频道"
                width={800}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="access-workflows">2. 打开工作流</h3>

              <p>
                在频道菜单中选择 <strong>&quot;Workflows&quot;</strong>{" "}
                打开 Power Automate 集成。
              </p>

              <Image
                src="/images/notifier-teams/image-02.png"
                alt="打开 Workflows"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-new-workflow">3. 创建新工作流</h3>

              <p>
                在 Workflows 面板中点击 <strong>&quot;Create&quot;</strong>{" "}
                或搜索{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>{" "}
                模板。
              </p>

              <Image
                src="/images/notifier-teams/image-03.png"
                alt="创建新工作流"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="select-webhook-template">4. 选择 webhook 模板</h3>

              <p>
                从可用选项中选择{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>{" "}
                模板。
              </p>

              <Image
                src="/images/notifier-teams/image-04.png"
                alt="选择 webhook 模板"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-webhook">5. 配置 webhook</h3>

              <p>
                为 webhook 设置名称（例如{" "}
                <strong>&quot;Databasus Backup Notifications&quot;</strong>
                ），并确认接收通知的频道。
              </p>

              <Image
                src="/images/notifier-teams/image-05.png"
                alt="配置 webhook"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="copy-webhook-url">6. 复制 webhook URL</h3>

              <p>
                创建工作流后会显示 <strong>HTTP POST URL</strong>。复制这个
                URL，稍后在 Databasus 中配置时会用到。
              </p>

              <Image
                src="/images/notifier-teams/image-06.png"
                alt="复制 webhook URL"
                width={500}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="configure-databasus">在 Databasus 中配置</h2>

              <h3 id="add-teams-notifier">1. 添加 Teams 通知</h3>

              <p>
                在 Databasus 中进入通知设置，添加一个新的 Microsoft Teams
                通知，粘贴刚才从 Teams 复制的 webhook URL。
              </p>

              <Image
                src="/images/notifier-teams/image-07.png"
                alt="在 Databasus 中配置 Teams"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="test-notification">2. 测试通知</h3>

              <p>
                配置好 webhook 后测试一下，确认通知正常工作。你应该会在
                选定的 Teams 频道里收到一条测试消息。
              </p>

              <p>
                大功告成！你的 Microsoft Teams 频道现在可以接收来自
                Databasus 的 PostgreSQL 备份通知了。
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/zh/notifiers"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← 返回通知列表
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
