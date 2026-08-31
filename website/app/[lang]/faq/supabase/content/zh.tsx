import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "如何用 Databasus 备份 Supabase | Databasus",
  description:
    "了解如何用 Databasus 备份 Supabase 的 PostgreSQL 数据库。配置 session pooler 或 IPv4 地址进行 Supabase 备份的分步指南。",
  keywords: [
    "Databasus",
    "Supabase 备份",
    "Supabase PostgreSQL 备份",
    "备份 Supabase 数据库",
    "Supabase session pooler",
    "Supabase IPv4",
    "PostgreSQL 备份",
    "数据库备份",
  ],
  openGraph: {
    title: "如何用 Databasus 备份 Supabase | Databasus",
    description:
      "了解如何用 Databasus 备份 Supabase 的 PostgreSQL 数据库。配置 session pooler 或 IPv4 地址进行 Supabase 备份的分步指南。",
    type: "article",
    url: getLocalizedUrl("zh", "faq/supabase"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "如何用 Databasus 备份 Supabase | Databasus",
    description:
      "了解如何用 Databasus 备份 Supabase 的 PostgreSQL 数据库。配置 session pooler 或 IPv4 地址进行 Supabase 备份的分步指南。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "faq/supabase"),
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
            name: "如何用 Databasus 备份 Supabase",
            description:
              "用 Databasus 备份 Supabase 的 PostgreSQL 数据库的分步指南",
            step: [
              {
                "@type": "HowToStep",
                name: "从 Supabase 获取连接信息",
                text: "打开 Supabase 项目设置，找到数据库连接信息。",
              },
              {
                "@type": "HowToStep",
                name: "使用带 IPv4 的 Session Pooler",
                text: "复制 Session Pooler 连接字符串，并确认 'Use IPv4 Address' 已启用。",
              },
              {
                "@type": "HowToStep",
                name: "配置 Databasus",
                text: "在 Databasus 中填入 Supabase 连接信息，开始备份数据库。",
              },
              {
                "@type": "HowToStep",
                name: "了解 schema 限制",
                text: "默认只备份 public schema，因为 Supabase 的其他 schema 访问受限。",
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
              <h1 id="supabase-backup">如何备份 Supabase</h1>

              <p className="text-lg text-gray-400">
                Databasus 支持备份 Supabase 的 PostgreSQL 数据库。主要
                要求是使用 IPv4 地址连接你的 Supabase 实例。
              </p>

              <h2 id="connection-options">连接方式</h2>

              <p>
                Databasus 连接 Supabase 数据库有两种方式：
              </p>

              <ol>
                <li>
                  <strong>使用带 IPv4 的 Session Pooler</strong>
                  （推荐）— 所有 Supabase 项目都可免费使用
                </li>
                <li>
                  <strong>购买 IPv4 附加组件</strong> — Supabase
                  提供的直连方式
                </li>
              </ol>

              <h2 id="session-pooler">
                方式 1：使用 Session Pooler（推荐）
              </h2>

              <p>
                Session Pooler 免费为你的 Supabase 数据库连接提供 IPv4
                地址。配置方法如下：
              </p>

              <h3 id="step-1">1. 找到 pooler 连接</h3>

              <p>
                打开你的 Supabase 项目，进入{" "}
                <strong>Project Settings</strong> → <strong>Database</strong>
                。向下滚动到 <strong>Connection string</strong> 区域，选择{" "}
                <strong>Session pooler</strong> 模式。
              </p>

              <img
                src="/images/faq/supabase/image-1.png"
                alt="在 Supabase 中选择 Session pooler 模式"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h3 id="step-2">2. 复制连接信息</h3>

              <p>
                复制连接信息，在 Databasus 添加数据库时填入。参考截图
                区分各项连接信息。
              </p>

              <img
                src="/images/faq/supabase/image-2.png"
                alt="在 Supabase 中启用 IPv4 Address 开关"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h2 id="ipv4-addon">方式 2：购买 IPv4 附加组件</h2>

              <p>
                Supabase 提供付费的 IPv4 附加组件，为你的数据库分配专用
                IPv4 地址。这种方式不经过连接池，直接连接数据库。
              </p>

              <p>启用方法：</p>

              <ol>
                <li>打开你的 Supabase 项目控制台</li>
                <li>
                  进入 <strong>Project Settings</strong> →{" "}
                  <strong>Add-ons</strong>
                </li>
                <li>
                  启用 <strong>IPv4</strong> 附加组件
                </li>
                <li>在 Databasus 中使用直连数据库的连接信息</li>
              </ol>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 提示：</strong>
                  对于大多数场景，免费的带 IPv4 的 Session Pooler
                  已经完全能满足备份需要。只有当你出于其他原因需要直连时，
                  才有必要购买付费的 IPv4 附加组件。
                </p>
              </div>

              <h2 id="default-schema">默认 schema 的限制</h2>

              <p>
                备份 Supabase 数据库时，Databasus 默认只备份{" "}
                <code>public</code> schema。这是因为出于安全考虑，Supabase
                限制了对其他 schema（如 <code>auth</code>、
                <code>storage</code> 和 <code>realtime</code>）的访问。
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-blue-400">ℹ️ 说明：</strong>
                  <code>public</code> schema
                  包含你的应用数据和自定义表。像 <code>auth</code> 和{" "}
                  <code>storage</code> 这样由 Supabase 托管的 schema
                  受到保护，由 Supabase 自己管理。
                </p>
              </div>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/zh/faq"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← 返回常见问题
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
