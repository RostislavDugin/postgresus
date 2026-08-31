import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "访问管理 - Databasus 文档",
  description:
    "了解如何在 Databasus 中管理访问权限、角色和授权。通过工作区级和系统级角色控制谁可以注册、创建工作区和管理数据库。",
  keywords: [
    "Databasus 访问管理",
    "用户角色",
    "工作区权限",
    "审计日志",
    "PostgreSQL 备份安全",
    "团队协作",
    "访问控制",
    "工作区管理",
  ],
  openGraph: {
    title: "访问管理 - Databasus 文档",
    description:
      "了解如何在 Databasus 中管理访问权限、角色和授权。通过工作区级和系统级角色控制谁可以注册、创建工作区和管理数据库。",
    type: "article",
    url: getLocalizedUrl("zh", "access-management"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "访问管理 - Databasus 文档",
    description:
      "了解如何在 Databasus 中管理访问权限、角色和授权。通过工作区级和系统级角色控制谁可以注册、创建工作区和管理数据库。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "access-management"),
    languages: getLanguageAlternates("access-management"),
  },
  robots: "index, follow",
};

export default function AccessManagementPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "访问管理 - Databasus 文档",
            description:
              "了解如何在 Databasus 中管理访问权限、角色和授权。通过工作区级和系统级角色控制谁可以注册、创建工作区和管理数据库。",
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
              <h1 id="settings">设置</h1>

              <p>
                Databasus 既适合单人使用，也适合团队。本节介绍团队的
                访问管理。
                <strong>
                  如果你是 Databasus 实例中的唯一用户
                </strong>
                ，可以跳过本节。
              </p>

              <p>
                Databasus 的设置并不多。实际上它只允许你控制：
              </p>

              <ul>
                <li>谁可以在你的 Databasus 实例中注册</li>
                <li>谁可以创建工作区</li>
                <li>谁可以在工作区内管理数据库、通知和存储</li>
              </ul>

              <h2 id="workspaces">工作区</h2>

              <p>
                工作区用来<strong>组织数据库、通知和存储</strong>。你可以
                向工作区添加成员（也可以创建多个工作区）。
              </p>

              <p>访问管理可以按工作区分别设置。例如：</p>

              <ul>
                <li>
                  你有一个 DevOps 团队负责项目的 10 个数据库（一个工作区
                  里放几个用户）；
                </li>
                <li>
                  你有 3 个项目，各自使用不同的数据库和存储（几个工作区，
                  各自有不同用户）；
                </li>
                <li>
                  你有 5 个独立的数据库，不同用户访问各自的库（用户 A
                  访问 DB1，用户 B 访问 DB2，用户 C 访问 DB3，以此类推）。
                </li>
              </ul>

              <img
                src="/images/access-management/users.png"
                alt="工作区"
                width={550}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                如果你允许用户在你的 Databasus 中注册并创建自己的
                工作区（见<a href="#global-settings">全局设置</a>
                ），他们就能创建自己的工作区。
              </p>

              <p>
                <strong>
                  在被邀请加入之前，用户永远看不到自己以外的工作区。
                </strong>
              </p>

              <h2 id="audit-logs">审计日志</h2>

              <p>
                审计日志记录用户执行的操作。它们用于追踪用户的变更和
                行为，也用于发现可疑活动。
              </p>

              <p>例如：</p>

              <ul>
                <li>用户创建了新数据库</li>
                <li>用户删除了数据库</li>
                <li>用户发起了新备份</li>
                <li>用户下载了备份</li>
                <li>用户创建了新通知</li>
                <li>用户创建了工作区</li>
                <li>用户删除了工作区</li>
                <li>等等</li>
              </ul>

              <p>审计日志可以按条件筛选查看：</p>

              <ul>
                <li>按工作区；</li>
                <li>按用户（跨多个工作区）；</li>
              </ul>

              <img
                src="/images/access-management/audit-logs.png"
                alt="审计日志"
                width={1000}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="user-roles">用户角色</h2>

              <p>
                Databasus 中的所有用户都有<u>系统级</u>角色：
              </p>

              <table>
                <thead>
                  <tr>
                    <th>功能</th>
                    <th>Admin</th>
                    <th>Member</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>管理所有设置和用户</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">❌</td>
                  </tr>
                  <tr>
                    <td>创建工作区</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">✅（若设置允许）</td>
                  </tr>
                </tbody>
              </table>

              <p>
                通常系统里只有一个 <code>admin</code> 用户，即你首次启动
                Databasus 时创建的那个。
              </p>

              <p>
                <u>工作区内</u>也有角色：
              </p>

              <table>
                <thead>
                  <tr>
                    <th>功能</th>
                    <th>Viewer</th>
                    <th>Member</th>
                    <th>Admin</th>
                    <th>Owner</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>查看数据库、通知、存储</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>发起和下载备份</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>管理数据库、通知、存储</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>管理用户</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>管理管理员</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">❌</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                </tbody>
              </table>

              <p>
                请注意：数据库、存储和通知的<strong>敏感数据</strong>
                （密码、令牌等）<strong>对所有用户始终隐藏</strong>
                。创建之后任何人都无法查看这些密钥。
              </p>

              <h2 id="global-settings">全局设置</h2>

              <p>全局设置中有 3 个选项：</p>

              <ol>
                <li>
                  <strong>允许外部注册</strong> - 默认情况下，任何人都可以
                  在你的 Databasus 中注册（但在被邀请或创建自己的工作区
                  之前，他们无法访问任何工作区）。
                  <br />
                  <br />
                  如果只想允许受邀用户注册，可以关闭这个选项。此时注册
                  表单会保持关闭，直到你把该用户邀请进某个工作区。
                  <br />
                  <br />
                  要邀请用户加入工作区，点击 &quot;Add
                  user&quot; 并输入邮箱。之后拥有该邮箱的用户就能完成
                  注册。
                </li>
                <li>
                  <strong>允许成员邀请</strong> - 这个设置在外部注册被
                  关闭时有用。
                  <br />
                  <br />
                  假设你已有一些可信的用户（例如你的团队），你希望允许
                  他们邀请其他用户加入 Databasus。此时可以开启这个选项，
                  他们就能通过邀请让其他用户加入工作区。
                  <br />
                  <br />
                  如果关闭，只有管理员能邀请用户。
                </li>
                <li>
                  <strong>允许成员创建工作区</strong> - 默认情况下，所有
                  成员都能创建自己的工作区。如果只想让管理员创建工作区，
                  可以关闭这个选项。
                </li>
              </ol>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
