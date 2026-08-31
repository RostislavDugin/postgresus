import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "如何为 Databasus 配置 Slack 通知 | Databasus",
  description:
    "在 Databasus 中为 PostgreSQL 备份提醒配置 Slack 通知的分步指南。了解如何创建 Slack 机器人应用并配置通知。",
  keywords: [
    "Databasus",
    "Slack 通知",
    "PostgreSQL 备份",
    "Slack 机器人令牌",
    "Slack API",
    "备份提醒",
    "数据库通知",
  ],
  openGraph: {
    title: "如何为 Databasus 配置 Slack 通知 | Databasus",
    description:
      "在 Databasus 中为 PostgreSQL 备份提醒配置 Slack 通知的分步指南。了解如何创建 Slack 机器人应用并配置通知。",
    type: "article",
    url: getLocalizedUrl("zh", "notifiers/slack"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "如何为 Databasus 配置 Slack 通知 | Databasus",
    description:
      "在 Databasus 中为 PostgreSQL 备份提醒配置 Slack 通知的分步指南。了解如何创建 Slack 机器人应用并配置通知。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "notifiers/slack"),
    languages: getLanguageAlternates("notifiers/slack"),
  },
  robots: "index, follow",
};

export default function SlackPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "如何为 Databasus 配置 Slack 通知",
            description:
              "在 Databasus 中为 PostgreSQL 备份提醒配置 Slack 通知的分步指南",
            step: [
              {
                "@type": "HowToStep",
                name: "打开 Slack API",
                text: "访问 https://api.slack.com/apps 并登录你的 Slack 工作区。",
              },
              {
                "@type": "HowToStep",
                name: "创建新应用",
                text: "点击 'Create New App' 按钮并选择 'From scratch'。",
              },
              {
                "@type": "HowToStep",
                name: "配置机器人权限",
                text: "进入 OAuth & Permissions，在 Bot Token Scopes 下添加所需的权限：chat:write、channels:join、im:write 和 groups:write。",
              },
              {
                "@type": "HowToStep",
                name: "安装到工作区",
                text: "将应用安装到你的工作区并授权。",
              },
              {
                "@type": "HowToStep",
                name: "复制机器人令牌",
                text: "复制以 'xoxb-' 开头的 Bot User OAuth Token。",
              },
              {
                "@type": "HowToStep",
                name: "获取频道 ID",
                text: "打开目标频道，在频道详情中找到频道 ID。",
              },
              {
                "@type": "HowToStep",
                name: "将机器人加入私有频道",
                text: "如果使用私有频道，在频道中提及机器人并邀请它加入。",
              },
              {
                "@type": "HowToStep",
                name: "在 Databasus 中配置",
                text: "在 Databasus 的 Slack 通知配置中填入机器人令牌和频道 ID。",
              },
              {
                "@type": "HowToStep",
                name: "测试通知",
                text: "测试通知，确认一切正常工作。",
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
              <h1 id="slack-notifications">Slack 通知</h1>

              <p className="text-lg text-gray-400">
                配置 Slack，即时接收 PostgreSQL 备份的状态通知。备份成功、
                失败和警告都会直接推送到你的 Slack 频道。
              </p>

              <h2 id="create-slack-app">创建 Slack 应用</h2>

              <h3 id="go-to-slack-api">1. 打开 Slack API</h3>

              <p>
                访问{" "}
                <a
                  href="https://api.slack.com/apps"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://api.slack.com/apps
                </a>{" "}
                并登录你的 Slack 工作区。
              </p>

              <h3 id="create-new-app">2. 创建新应用</h3>

              <p>
                点击 <strong>&quot;Create New App&quot;</strong> 按钮。
              </p>

              <h3 id="choose-from-scratch">
                3. 选择 &quot;From scratch&quot;
              </h3>

              <p>
                在弹出的选项中选择 <strong>&quot;From scratch&quot;</strong>。
              </p>

              <h3 id="name-your-app">4. 为应用命名</h3>

              <p>
                输入应用名称（例如 &quot;Databasus
                Notifications&quot;），选择要安装的工作区，然后点击{" "}
                <strong>&quot;Create App&quot;</strong>。
              </p>

              <h2 id="configure-bot-permissions">配置机器人权限</h2>

              <h3 id="navigate-to-oauth">
                5. 进入 OAuth &amp; Permissions
              </h3>

              <p>
                在左侧边栏点击{" "}
                <strong>&quot;OAuth &amp; Permissions&quot;</strong>。
              </p>

              <img
                src="/images/notifier-slack/image-1.png"
                alt="进入 OAuth &amp; Permissions"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="add-bot-scopes">6. 添加 Bot Token Scopes（必需）</h3>

              <p>
                向下滚动到 <strong>&quot;Scopes&quot;</strong> 区域，在{" "}
                <strong>&quot;Bot Token Scopes&quot;</strong> 下点击{" "}
                <strong>&quot;Add an OAuth Scope&quot;</strong>。
              </p>

              <p>添加以下全部必需权限：</p>

              <ul>
                <li>
                  <code>chat:write</code> - 向频道发送消息
                </li>
                <li>
                  <code>channels:join</code> - 允许机器人自动加入公开频道
                </li>
                <li>
                  <code>im:write</code> - 向用户发送私信
                </li>
                <li>
                  <code>groups:write</code> - 向私有频道发送消息
                </li>
                <li>
                  <code>channels:history</code> - 读取频道历史记录
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-2.png"
                alt="添加 Bot Token Scopes"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h2 id="install-app">将应用安装到工作区</h2>

              <h3 id="install-to-workspace">7. 安装到工作区</h3>

              <p>
                滚动到 <strong>&quot;OAuth &amp; Permissions&quot;</strong>{" "}
                页面顶部，点击{" "}
                <strong>&quot;Install to Workspace&quot;</strong>。
              </p>

              <img
                src="/images/notifier-slack/image-3.png"
                alt="安装到工作区"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="authorize-app">8. 授权应用</h3>

              <p>
                检查权限列表，点击 <strong>&quot;Allow&quot;</strong>{" "}
                完成授权。
              </p>

              <h3 id="copy-bot-token">9. 复制 Bot User OAuth Token</h3>

              <p>
                安装完成后会显示{" "}
                <strong>&quot;Bot User OAuth Token&quot;</strong>，以{" "}
                <code>xoxb-</code> 开头。复制这个令牌，稍后在 Databasus
                中配置时会用到。
              </p>

              <h2 id="get-channel-id">获取频道 ID</h2>

              <h3 id="open-channel">10. 打开目标频道</h3>

              <p>
                在 Slack 工作区中打开你想接收备份通知的频道。
              </p>

              <h3 id="get-channel-info">11. 获取频道 ID</h3>

              <p>
                点击顶部的频道名称，然后在频道详情中向下滚动。在
                &quot;About&quot; 区域底部可以找到{" "}
                <strong>频道 ID</strong>，公开频道以 <code>C</code>{" "}
                开头，私有频道以 <code>G</code> 开头。
              </p>

              <p>复制这个频道 ID。</p>

              <img
                src="/images/notifier-slack/image-4.png"
                alt="获取频道 ID"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[500px]"
                loading="lazy"
              />

              <h3 id="add-bot-to-channel">
                12. 将机器人加入频道（私有频道必需）
              </h3>

              <p>
                <strong>
                  如果使用私有频道，必须手动邀请机器人加入：
                </strong>
              </p>

              <ol>
                <li>
                  在私有频道中输入{" "}
                  <code>@Databasus Notifications</code>
                  （或你为应用取的名字）
                </li>
                <li>
                  在弹出的机器人名称上点击，选择{" "}
                  <strong>&quot;Add to Channel&quot;</strong> 或{" "}
                  <strong>&quot;Invite to Channel&quot;</strong>
                </li>
              </ol>

              <p>
                对于<strong>公开频道</strong>，机器人在发送第一条消息时会
                自动加入（得益于 <code>channels:join</code>{" "}
                权限），无需这一步。
              </p>

              <h2 id="configure-databasus">在 Databasus 中配置</h2>

              <h3 id="add-slack-notifier">13. 添加 Slack 通知</h3>

              <p>
                在 Databasus 中进入通知设置，添加一个新的 Slack 通知：
              </p>

              <ul>
                <li>
                  <strong>Bot Token：</strong>粘贴刚才复制的 Bot User OAuth
                  Token（以 <code>xoxb-</code> 开头）
                </li>
                <li>
                  <strong>Target Channel ID：</strong>粘贴刚才复制的频道
                  ID（以 <code>C</code>、<code>G</code>、<code>D</code> 或{" "}
                  <code>U</code> 开头）
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-5.png"
                alt="添加 Slack 通知"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="test-notification">14. 测试通知</h3>

              <p>
                配置完成后测试一下，确认通知正常工作。你应该会在选定的
                Slack 频道里收到一条测试消息。
              </p>

              <p>
                大功告成！你的 Slack 工作区现在可以接收来自 Databasus 的
                PostgreSQL 备份通知了。
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
