import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "重置密码 - Databasus 文档",
  description:
    "了解如何使用内置命令行工具在 Databasus 中重置用户密码。为你的 PostgreSQL 备份系统提供快速安全的密码找回方式。",
  keywords: [
    "Databasus 重置密码",
    "重置用户密码",
    "PostgreSQL 备份密码",
    "Docker 密码找回",
    "密码找回",
    "Databasus 认证",
  ],
  openGraph: {
    title: "重置密码 - Databasus 文档",
    description:
      "了解如何使用内置命令行工具在 Databasus 中重置用户密码。为你的 PostgreSQL 备份系统提供快速安全的密码找回方式。",
    type: "article",
    url: getLocalizedUrl("zh", "password"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "重置密码 - Databasus 文档",
    description:
      "了解如何使用内置命令行工具在 Databasus 中重置用户密码。为你的 PostgreSQL 备份系统提供快速安全的密码找回方式。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "password"),
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
            headline: "重置密码 - Databasus 文档",
            description:
              "了解如何使用内置命令行工具在 Databasus 中重置用户密码。",
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
            name: "如何重置 Databasus 用户密码",
            description: "在 Databasus 中重置用户密码的分步指南",
            step: [
              {
                "@type": "HowToStep",
                name: "运行重置密码命令",
                text: "执行 docker exec 命令，传入新密码和用户邮箱。",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "使用 docker exec 在 Databasus 容器内运行重置密码命令",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "验证密码修改",
                text: "用新密码登录 Databasus，确认修改已生效。",
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
              <h1 id="reset-password">重置用户密码</h1>

              <h2 id="reset-password-command">重置密码命令</h2>

              <p>
                要重置用户密码，在运行 Databasus 的服务器上执行以下命令：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={resetPasswordCommand} />
                </div>
              </div>

              <h2 id="parameters">参数</h2>

              <p>命令接受以下参数：</p>

              <ul>
                <li>
                  <strong>--new-password</strong>：新密码。请确保密码足够
                  安全，包含字母、数字和特殊字符。
                </li>
                <li>
                  <strong>--email</strong>：要重置密码的用户邮箱（例如{" "}
                  <code>admin</code>、<code>user@example.com</code>）。
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
