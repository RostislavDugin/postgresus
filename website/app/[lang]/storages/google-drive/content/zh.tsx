import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "如何将 Google Drive 连接到 Databasus | Databasus",
  description:
    "在 Databasus 中为 PostgreSQL 备份配置 Google Drive 存储的分步指南。了解如何创建 Google Cloud 项目并配置 OAuth。",
  keywords: [
    "Databasus",
    "Google Drive",
    "PostgreSQL 备份",
    "Google Cloud",
    "OAuth",
    "云存储",
    "数据库备份",
  ],
  openGraph: {
    title: "如何将 Google Drive 连接到 Databasus | Databasus",
    description:
      "在 Databasus 中为 PostgreSQL 备份配置 Google Drive 存储的分步指南。了解如何创建 Google Cloud 项目并配置 OAuth。",
    type: "article",
    url: getLocalizedUrl("zh", "storages/google-drive"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "如何将 Google Drive 连接到 Databasus | Databasus",
    description:
      "在 Databasus 中为 PostgreSQL 备份配置 Google Drive 存储的分步指南。了解如何创建 Google Cloud 项目并配置 OAuth。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "storages/google-drive"),
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
            name: "如何将 Google Drive 连接到 Databasus",
            description:
              "在 Databasus 中为 PostgreSQL 备份配置 Google Drive 存储的分步指南",
            step: [
              {
                "@type": "HowToStep",
                name: "创建新项目",
                text: "打开 Google Cloud Console 并创建一个新项目。",
              },
              {
                "@type": "HowToStep",
                name: "启用 Google Drive API",
                text: "进入 API & Services 标签页，打开 API library 并启用 Google Drive API。",
              },
              {
                "@type": "HowToStep",
                name: "配置同意屏幕",
                text: "进入 Credentials → Create credentials → Configure consent screen 并填写必要信息。",
              },
              {
                "@type": "HowToStep",
                name: "创建 OAuth 客户端 ID",
                text: "进入 Credentials → Create credentials → OAuth client ID。",
              },
              {
                "@type": "HowToStep",
                name: "配置应用设置",
                text: "将应用类型设为 Web application，配置授权来源和重定向 URI。",
              },
              {
                "@type": "HowToStep",
                name: "添加权限范围",
                text: '进入 Data Access 并添加权限范围 "/auth/drive.file"。',
              },
              {
                "@type": "HowToStep",
                name: "发布应用",
                text: "进入 Audience 并发布应用。",
              },
              {
                "@type": "HowToStep",
                name: "使用 Google 账号登录",
                text: "在 Databasus 中填写凭据并用你的 Google 账号登录。",
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
              <h1 id="google-drive">Google Drive 存储</h1>

              <p className="text-lg text-gray-400">
                要把备份保存到 Google Drive，需要先创建一个 Google Cloud
                项目来访问 Google Drive API，然后用你的 Google 账号登录。
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">重要：</strong>连接
                  Google Drive 要求你的 Databasus 实例通过 HTTPS
                  运行。关于配置 HTTPS 的更多信息，请参考{" "}
                  <a
                    href="/zh/installation#caddy-reverse-proxy"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Caddy 反向代理配置指南 →
                  </a>
                  <br />
                  本指南假设你的 Databasus 实例运行在{" "}
                  <code>databasus.yourdomain.com</code>。
                </p>
              </div>

              <h2 id="create-google-cloud-project">
                创建 Google Cloud 项目
              </h2>

              <h3 id="create-new-project">1. 创建新项目</h3>

              <p>
                打开{" "}
                <a
                  href="https://console.cloud.google.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://console.cloud.google.com/
                </a>{" "}
                并选择 <strong>&quot;new project&quot;</strong>（左上角）。
              </p>

              <h3 id="enable-google-drive-api">2. 启用 Google Drive API</h3>

              <p>
                进入 <strong>&quot;API &amp; Services&quot;</strong>{" "}
                标签页，打开 <strong>&quot;API library&quot;</strong>，选择{" "}
                <strong>Google Drive API</strong> 并启用：
              </p>

              <Image
                src="/images/google-drive-storage/image-1.webp"
                alt="启用 Google Drive API"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-consent-screen">3. 配置同意屏幕</h3>

              <p>
                进入 <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;Configure consent screen&quot;</strong>{" "}
                并填写相应信息：
              </p>

              <Image
                src="/images/google-drive-storage/image-2.webp"
                alt="配置同意屏幕"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-oauth-client-id">4. 创建 OAuth 客户端 ID</h3>

              <p>
                进入 <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;OAuth client ID&quot;</strong>：
              </p>

              <Image
                src="/images/google-drive-storage/image-3.webp"
                alt="创建 OAuth 客户端 ID"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-application-settings">
                5. 配置应用设置
              </h3>

              <p>填写以下信息：</p>

              <ul>
                <li>
                  <strong>Application type：</strong>Web application
                </li>
                <li>
                  <strong>Authorized JavaScript origins：</strong>{" "}
                  <code>https://databasus.yourdomain.com</code>
                </li>
                <li>
                  <strong>Authorized redirect URIs：</strong>{" "}
                  <code>
                    https://databasus.yourdomain.com/storages/google-oauth
                  </code>
                </li>
              </ul>

              <p>
                <strong>注意：</strong>把{" "}
                <code>databasus.yourdomain.com</code> 替换成你实际部署
                Databasus 的域名（例如 <code>backup.example.com</code>）。
              </p>

              <p>然后复制凭据：</p>

              <Image
                src="/images/google-drive-storage/image-4.png"
                alt="配置应用设置 - 第 1 部分"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <Image
                src="/images/google-drive-storage/image-5.png"
                alt="配置应用设置 - 第 2 部分"
                width={450}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="add-scope">6. 添加权限范围</h3>

              <p>
                进入 <strong>&quot;Data Access&quot;</strong>{" "}
                并添加权限范围{" "}
                <code>&quot;/auth/drive.file&quot;</code>：
              </p>

              <Image
                src="/images/google-drive-storage/image-6.png"
                alt="添加权限范围"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="publish-app">7. 发布应用</h3>

              <p>
                进入 <strong>&quot;Audience&quot;</strong> 并发布应用：
              </p>

              <Image
                src="/images/google-drive-storage/image-7.png"
                alt="发布应用"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="sign-in-google-account">使用 Google 账号登录</h2>

              <h3 id="fill-credentials">1. 填写凭据</h3>

              <p>把前面步骤中获取的凭据填入 Databasus：</p>

              <Image
                src="/images/google-drive-storage/image-8.png"
                alt="填写凭据"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="choose-account">2. 选择账号</h3>

              <p>选择你的 Google 账号完成登录。</p>

              <h3 id="handle-security-warning">3. 处理安全警告</h3>

              <p>
                如果出现警告，点击{" "}
                <strong>&quot;Advanced&quot;</strong>（左下角），然后选择{" "}
                <strong>&quot;Proceed anyway&quot;</strong>。
              </p>

              <p>
                <strong>注意：</strong>出现这个警告是因为你的应用还没有
                通过 Google 的验证。对于你自己的应用，继续操作是安全的。
              </p>

              <p>
                大功告成！你的 Google Drive 已经连接到 Databasus，可以
                开始存放 PostgreSQL 备份了。
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/zh/storages"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← 返回存储列表
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
