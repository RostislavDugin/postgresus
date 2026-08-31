import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "如何在 Databasus 中使用 Cloudflare R2 | Databasus",
  description:
    "在 Databasus 中为 PostgreSQL 备份配置 Cloudflare R2 存储的分步指南。了解如何用 R2 搭建 S3 兼容存储。",
  keywords: [
    "Databasus",
    "Cloudflare R2",
    "PostgreSQL 备份",
    "S3 存储",
    "云存储",
    "数据库备份",
  ],
  openGraph: {
    title: "如何在 Databasus 中使用 Cloudflare R2 | Databasus",
    description:
      "在 Databasus 中为 PostgreSQL 备份配置 Cloudflare R2 存储的分步指南。了解如何用 R2 搭建 S3 兼容存储。",
    type: "article",
    url: getLocalizedUrl("zh", "storages/cloudflare-r2"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "如何在 Databasus 中使用 Cloudflare R2 | Databasus",
    description:
      "在 Databasus 中为 PostgreSQL 备份配置 Cloudflare R2 存储的分步指南。了解如何用 R2 搭建 S3 兼容存储。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "storages/cloudflare-r2"),
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
            name: "如何在 Databasus 中使用 Cloudflare R2",
            description:
              "在 Databasus 中为 PostgreSQL 备份配置 Cloudflare R2 存储的分步指南",
            step: [
              {
                "@type": "HowToStep",
                name: "填写存储桶名称",
                text: "在存储配置中填写你的 R2 存储桶名称。",
              },
              {
                "@type": "HowToStep",
                name: "设置区域",
                text: '在区域字段中填写 "auto"。',
              },
              {
                "@type": "HowToStep",
                name: "生成 Access Key ID 和 Secret Access Key",
                text: "在 Cloudflare 控制台进入 R2 → API → Manage API Tokens，创建令牌并授予所需权限。",
              },
              {
                "@type": "HowToStep",
                name: "找到账户 ID",
                text: "在控制台的任意 R2 页面顶部附近可以看到你的账户 ID。",
              },
              {
                "@type": "HowToStep",
                name: "拼出 S3 端点",
                text: "按 https://<ACCOUNT_ID>.r2.cloudflarestorage.com 的格式，把 <ACCOUNT_ID> 替换成控制台中的值。",
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
              <h1 id="cloudflare-r2">Cloudflare R2 存储</h1>

              <p className="text-lg text-gray-400">
                要把 Cloudflare R2 作为 S3 兼容存储来存放 PostgreSQL
                备份，需要配置 R2 存储桶的凭据和端点。
              </p>

              <h2 id="configuration-steps">配置步骤</h2>

              <h3 id="fill-bucket-name">1. 填写存储桶名称</h3>

              <p>在存储配置中填写你的 R2 存储桶名称：</p>

              <Image
                src="/images/cloudflare-r2-storage/image-1.webp"
                alt="填写 Cloudflare R2 存储桶名称"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="set-region">2. 设置区域</h3>

              <p>
                在区域字段中填写 <code>&quot;auto&quot;</code>。
              </p>

              <h3 id="generate-access-key">
                3. 生成 Access Key ID 和 Secret Access Key
              </h3>

              <p>
                在 Cloudflare 控制台进入{" "}
                <strong>R2 → API → Manage API Tokens</strong>，创建一个
                新令牌并授予所需权限（例如{" "}
                <strong>&quot;Object Read &amp; Write&quot;</strong>）。
              </p>

              <p>令牌创建后会显示：</p>

              <ul>
                <li>
                  <strong>Access Key ID</strong>（令牌的 ID）
                </li>
                <li>
                  <strong>Secret Access Key</strong>（令牌值的 SHA-256 哈希）
                </li>
              </ul>

              <p>把这两个值复制到 Databasus：</p>

              <Image
                src="/images/cloudflare-r2-storage/image-2.gif"
                alt="生成 Access Key ID 和 Secret Access Key"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="find-account-id">4. 找到账户 ID</h3>

              <p>
                在控制台的任意 R2 页面顶部附近（或账户设置中）可以看到 你的账户
                ID：
              </p>

              <Image
                src="/images/cloudflare-r2-storage/image-3.webp"
                alt="在 Cloudflare 控制台找到账户 ID"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="construct-endpoint">5. 拼出 S3 端点</h3>

              <p>S3 端点使用以下格式：</p>

              <pre>
                <code>https://&lt;ACCOUNT_ID&gt;.r2.cloudflarestorage.com</code>
              </pre>

              <p>
                把 <code>&lt;ACCOUNT_ID&gt;</code> 替换成控制台中的值， 填入
                Databasus。
              </p>

              <p>大功告成！你的配置现在应该是这样：</p>

              <Image
                src="/images/cloudflare-r2-storage/image-4.png"
                alt="配置完成"
                width={500}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                你的 Databasus 现在可以使用 Cloudflare R2 存放 PostgreSQL
                备份了。
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
