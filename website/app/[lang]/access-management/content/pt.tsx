import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Gestão de acesso - Documentação do Databasus",
  description:
    "Aprenda a gerenciar acessos, papéis e permissões no Databasus. Controle quem pode se registrar, criar workspaces e gerenciar bases de dados com papéis no nível do workspace e do sistema.",
  keywords: [
    "gestão de acesso Databasus",
    "papéis de usuário",
    "permissões de workspace",
    "registros de auditoria",
    "segurança de backup PostgreSQL",
    "colaboração em equipe",
    "controle de acesso",
    "gestão de workspaces",
  ],
  openGraph: {
    title: "Gestão de acesso - Documentação do Databasus",
    description:
      "Aprenda a gerenciar acessos, papéis e permissões no Databasus. Controle quem pode se registrar, criar workspaces e gerenciar bases de dados com papéis no nível do workspace e do sistema.",
    type: "article",
    url: getLocalizedUrl("pt", "access-management"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Gestão de acesso - Documentação do Databasus",
    description:
      "Aprenda a gerenciar acessos, papéis e permissões no Databasus. Controle quem pode se registrar, criar workspaces e gerenciar bases de dados com papéis no nível do workspace e do sistema.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "access-management"),
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
            headline: "Gestão de acesso - Documentação do Databasus",
            description:
              "Aprenda a gerenciar acessos, papéis e permissões no Databasus. Controle quem pode se registrar, criar workspaces e gerenciar bases de dados com papéis no nível do workspace e do sistema.",
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

      <DocsNavbarComponent lang="pt" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="pt" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="settings">Configurações</h1>

              <p>
                O Databasus serve tanto para usuários individuais quanto para
                equipes. Esta seção trata da gestão de acesso para equipes.{" "}
                <strong>
                  Por isso, se você for o único usuário da sua instância do
                  Databasus
                </strong>
                , pode pular esta seção.
              </p>

              <p>
                O Databasus não tem muitas configurações. Na verdade, só dá para
                controlar:
              </p>

              <ul>
                <li>Quem pode se registrar na sua instância do Databasus</li>
                <li>Quem pode criar workspaces</li>
                <li>
                  Quem pode gerenciar bases de dados, notificadores e
                  armazenamentos dentro dos workspaces
                </li>
              </ul>

              <h2 id="workspaces">Workspaces</h2>

              <p>
                Um workspace é o lugar onde{" "}
                <strong>
                  agrupa bases de dados, notificadores e armazenamentos
                </strong>
                . Você pode adicionar membros aos workspaces (e criar vários
                workspaces).
              </p>

              <p>O acesso é gerenciado por workspace. Por exemplo:</p>

              <ul>
                <li>
                  você tem uma equipe de DevOps responsável por 10 bases de
                  dados do projeto (ou seja, alguns usuários dentro de um
                  workspace);
                </li>
                <li>
                  tem 3 projetos diferentes com bases de dados e armazenamentos
                  diferentes (ou seja, alguns workspaces com usuários
                  diferentes);
                </li>
                <li>
                  tem 5 bases de dados independentes, cada uma acessível a
                  usuários diferentes (o usuário A tem acesso à DB1, o usuário B
                  à DB2, o usuário C à DB3, etc.).
                </li>
              </ul>

              <img
                src="/images/access-management/users.png"
                alt="Workspaces"
                width={550}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                Se você permitir o registro de novos usuários e a criação de
                workspaces (veja as{" "}
                <a href="#global-settings">configurações globais</a>), eles
                poderão criar os próprios workspaces.
              </p>

              <p>
                <strong>
                  Os usuários nunca veem workspaces além dos seus até serem
                  convidados a entrar.
                </strong>
              </p>

              <h2 id="audit-logs">Registros de auditoria</h2>

              <p>
                Os registros de auditoria são mensagens sobre ações realizadas
                pelos usuários. Servem para acompanhar as alterações e detectar
                qualquer atividade suspeita.
              </p>

              <p>Por exemplo:</p>

              <ul>
                <li>o usuário criou uma nova base de dados</li>
                <li>o usuário apagou uma base de dados</li>
                <li>o usuário iniciou um novo backup</li>
                <li>o usuário baixou um backup</li>
                <li>o usuário criou um novo notificador</li>
                <li>o usuário criou um workspace</li>
                <li>o usuário apagou um workspace</li>
                <li>etc.</li>
              </ul>

              <p>Você pode consultar os registros de auditoria com filtros:</p>

              <ul>
                <li>por workspace;</li>
                <li>por usuário (em vários workspaces);</li>
              </ul>

              <img
                src="/images/access-management/audit-logs.png"
                alt="Registros de auditoria"
                width={1000}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="user-roles">Papéis de usuário</h2>

              <p>
                Todos os usuários do Databasus têm papéis{" "}
                <u>dentro do sistema</u>:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Funcionalidade</th>
                    <th>Admin</th>
                    <th>Member</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Gerenciar todas as configurações e usuários</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">❌</td>
                  </tr>
                  <tr>
                    <td>Criar workspaces</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">
                      ✅ (se permitido nas configurações)
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Normalmente, há apenas um usuário <code>admin</code> no sistema,
                criado na primeira vez que você inicia o Databasus.
              </p>

              <p>
                <u>Dentro de um workspace</u> também há papéis:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Funcionalidade</th>
                    <th>Viewer</th>
                    <th>Member</th>
                    <th>Admin</th>
                    <th>Owner</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Ver bases de dados, notificadores, armazenamentos</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Iniciar e baixar backups</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>
                      Gerenciar bases de dados, notificadores, armazenamentos
                    </td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Gerenciar usuários</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Gerenciar administradores</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">❌</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                </tbody>
              </table>

              <p>
                Atenção: os <strong>dados sensíveis</strong> (senhas, tokens,
                etc.) das bases de dados, armazenamentos e notificadores{" "}
                <strong>ficam sempre ocultos para qualquer usuário</strong>.
                Ninguém pode ver os segredos depois da criação.
              </p>

              <h2 id="global-settings">Configurações globais</h2>

              <p>Nas configurações globais há 3 opções:</p>

              <ol>
                <li>
                  <strong>Allow external registrations</strong> - por padrão,
                  todos os usuários podem se registrar no seu Databasus (mas
                  continuam sem acesso a qualquer workspace até serem convidados
                  ou criarem os próprios workspaces).
                  <br />
                  <br />
                  Se quiser permitir o registro apenas a usuários convidados,
                  desative esta opção. Nesse caso, o formulário de registro fica
                  fechado até você convidar o usuário para algum dos workspaces.
                  <br />
                  <br />
                  Para convidar usuários para o workspace, clique em &quot;Add
                  user&quot; e insira um email. Depois disso, o usuário com esse
                  email poderá concluir o registro.
                </li>
                <li>
                  <strong>Allow member invitations</strong> - esta configuração
                  é útil quando os registros externos estão desativados.
                  <br />
                  <br />
                  Imagine que você já tem alguns usuários de confiança (por
                  exemplo, a sua equipe) e quer que eles possam trazer outras
                  pessoas para o Databasus. Ative esta opção e eles poderão
                  convidar usuários para os workspaces.
                  <br />
                  <br />
                  Se estiver desativada, só os administradores podem convidar
                  usuários.
                </li>
                <li>
                  <strong>Allow member workspace creation</strong> - por padrão,
                  todos os membros podem criar os próprios workspaces. Se quiser
                  que só os administradores possam criar workspaces, desative
                  esta opção.
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
