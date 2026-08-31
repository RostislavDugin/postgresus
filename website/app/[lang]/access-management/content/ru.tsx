import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Управление доступом - документация Databasus",
  description:
    "Как управлять доступом, ролями и правами в Databasus: кто может регистрироваться, создавать рабочие пространства и управлять базами данных. Роли задаются на уровне рабочего пространства и всей системы.",
  keywords: [
    "управление доступом Databasus",
    "роли пользователей",
    "права рабочих пространств",
    "журнал аудита",
    "безопасность бекапов PostgreSQL",
    "командная работа",
    "контроль доступа",
    "управление рабочими пространствами",
  ],
  openGraph: {
    title: "Управление доступом - документация Databasus",
    description:
      "Как управлять доступом, ролями и правами в Databasus: кто может регистрироваться, создавать рабочие пространства и управлять базами данных. Роли задаются на уровне рабочего пространства и всей системы.",
    type: "article",
    url: getLocalizedUrl("ru", "access-management"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Управление доступом - документация Databasus",
    description:
      "Как управлять доступом, ролями и правами в Databasus: кто может регистрироваться, создавать рабочие пространства и управлять базами данных. Роли задаются на уровне рабочего пространства и всей системы.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "access-management"),
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
            headline: "Управление доступом - документация Databasus",
            description:
              "Как управлять доступом, ролями и правами в Databasus: кто может регистрироваться, создавать рабочие пространства и управлять базами данных. Роли задаются на уровне рабочего пространства и всей системы.",
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

      <DocsNavbarComponent lang="ru" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="ru" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="settings">Настройки</h1>

              <p>
                Databasus подходит и одиночным пользователям, и командам. Этот
                раздел посвящен управлению доступом для команд.{" "}
                <strong>
                  Так что если вы единственный пользователь своего экземпляра
                  Databasus
                </strong>
                , этот раздел можно пропустить.
              </p>

              <p>Настроек в Databasus немного. По сути, вы решаете только:</p>

              <ul>
                <li>кто может регистрироваться в вашем экземпляре Databasus</li>
                <li>кто может создавать рабочие пространства</li>
                <li>
                  кто может управлять базами данных, уведомлениями и хранилищами
                  внутри рабочих пространств
                </li>
              </ul>

              <h2 id="workspaces">Рабочие пространства</h2>

              <p>
                Рабочее пространство — это место, где вы{" "}
                <strong>
                  группируете базы данных, уведомления и хранилища
                </strong>
                . В рабочие пространства можно добавлять участников (и создавать
                несколько рабочих пространств).
              </p>

              <p>
                Доступом можно управлять отдельно в каждом рабочем пространстве.
                Например:
              </p>

              <ul>
                <li>
                  у вас есть DevOps-команда, отвечающая за 10 БД проекта
                  (несколько пользователей в одном рабочем пространстве);
                </li>
                <li>
                  у вас 3 разных проекта с разными БД и хранилищами (несколько
                  рабочих пространств с разными пользователями);
                </li>
                <li>
                  у вас 5 независимых БД, и к каждой имеют доступ разные
                  пользователи (пользователь A имеет доступ к DB1, пользователь
                  B — к DB2, пользователь C — к DB3 и т.д.).
                </li>
              </ul>

              <img
                src="/images/access-management/users.png"
                alt="Рабочие пространства"
                width={550}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                Если вы разрешите пользователям регистрироваться в вашем
                Databasus и создавать рабочие пространства (см.{" "}
                <a href="#global-settings">глобальные настройки</a>), у каждого
                из них появятся свои.
              </p>

              <p>
                <strong>
                  Пользователи не видят чужие рабочие пространства, пока их туда
                  не пригласят.
                </strong>
              </p>

              <h2 id="audit-logs">Журнал аудита</h2>

              <p>
                Журнал аудита — это записи о действиях пользователей. По ним
                видно, кто что менял, и легче заметить подозрительную
                активность.
              </p>

              <p>Например:</p>

              <ul>
                <li>пользователь создал новую базу данных</li>
                <li>пользователь удалил базу данных</li>
                <li>пользователь запустил новый бекап</li>
                <li>пользователь скачал бекап</li>
                <li>пользователь создал новое уведомление</li>
                <li>пользователь создал рабочее пространство</li>
                <li>пользователь удалил рабочее пространство</li>
                <li>и т.д.</li>
              </ul>

              <p>Журнал аудита можно просматривать с фильтрами:</p>

              <ul>
                <li>по рабочему пространству;</li>
                <li>по пользователю (в нескольких рабочих пространствах);</li>
              </ul>

              <img
                src="/images/access-management/audit-logs.png"
                alt="Журнал аудита"
                width={1000}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="user-roles">Роли пользователей</h2>

              <p>
                У всех пользователей Databasus есть роли{" "}
                <u>на уровне системы</u>:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Возможность</th>
                    <th>Admin</th>
                    <th>Member</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Управление всеми настройками и пользователями</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">❌</td>
                  </tr>
                  <tr>
                    <td>Создание рабочих пространств</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">✅ (если разрешено настройками)</td>
                  </tr>
                </tbody>
              </table>

              <p>
                Обычно в системе только один пользователь <code>admin</code>,
                которого вы создаете при первом запуске Databasus.
              </p>

              <p>
                <u>Внутри рабочего пространства</u> тоже есть роли:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Возможность</th>
                    <th>Viewer</th>
                    <th>Member</th>
                    <th>Admin</th>
                    <th>Owner</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Просмотр баз данных, уведомлений, хранилищ</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Запуск и скачивание бекапов</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>
                      Управление базами данных, уведомлениями, хранилищами
                    </td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Управление пользователями</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Управление администраторами</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">❌</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                </tbody>
              </table>

              <p>
                Помните: <strong>чувствительные данные</strong> (пароли, токены
                и т.д.) БД, хранилищ и уведомлений{" "}
                <strong>всегда скрыты от любого пользователя</strong>. После
                создания секреты не видит никто.
              </p>

              <h2 id="global-settings">Глобальные настройки</h2>

              <p>В глобальных настройках есть 3 свойства:</p>

              <ol>
                <li>
                  <strong>Allow external registrations</strong> — по умолчанию
                  любой пользователь может зарегистрироваться в вашем Databasus
                  (но у него не будет доступа ни к одному рабочему пространству,
                  пока его не пригласят или он не создаст собственное).
                  <br />
                  <br />
                  Если вы хотите разрешить регистрацию только приглашенным
                  пользователям, отключите эту опцию. Тогда форма регистрации
                  будет закрыта, пока вы не пригласите пользователя в одно из
                  рабочих пространств.
                  <br />
                  <br />
                  Чтобы пригласить пользователя в рабочее пространство, нажмите
                  &quot;Add user&quot; и введите email. После этого пользователь
                  с этим email сможет завершить регистрацию.
                </li>
                <li>
                  <strong>Allow member invitations</strong> — эта настройка
                  нужна, когда внешняя регистрация отключена.
                  <br />
                  <br />
                  Представьте, что у вас уже есть пользователи, которым вы
                  доверяете (например, ваша команда). Вы хотите разрешить им
                  приглашать других пользователей в Databasus. В этом случае
                  включите эту опцию, и они смогут приглашать других
                  пользователей в рабочие пространства.
                  <br />
                  <br />
                  Если она отключена, приглашать пользователей могут только
                  администраторы.
                </li>
                <li>
                  <strong>Allow member workspace creation</strong> — по
                  умолчанию все участники могут создавать собственные рабочие
                  пространства. Если вы хотите, чтобы рабочие пространства
                  создавали только администраторы, отключите эту опцию.
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
