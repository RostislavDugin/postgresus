import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Como configurar notificações do Slack no Databasus | Databasus",
  description:
    "Guia passo a passo para configurar notificações do Slack para alertas de backup PostgreSQL com o Databasus. Aprenda a criar um bot do Slack e configurar as notificações.",
  keywords: [
    "Databasus",
    "notificações Slack",
    "backup PostgreSQL",
    "token de bot Slack",
    "API do Slack",
    "alertas de backup",
    "notificações de base de dados",
  ],
  openGraph: {
    title: "Como configurar notificações do Slack no Databasus | Databasus",
    description:
      "Guia passo a passo para configurar notificações do Slack para alertas de backup PostgreSQL com o Databasus. Aprenda a criar um bot do Slack e configurar as notificações.",
    type: "article",
    url: getLocalizedUrl("pt", "notifiers/slack"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Como configurar notificações do Slack no Databasus | Databasus",
    description:
      "Guia passo a passo para configurar notificações do Slack para alertas de backup PostgreSQL com o Databasus. Aprenda a criar um bot do Slack e configurar as notificações.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "notifiers/slack"),
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
            name: "Como configurar notificações do Slack no Databasus",
            description:
              "Guia passo a passo para configurar notificações do Slack para alertas de backup PostgreSQL com o Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Abra a API do Slack",
                text: "Vá a https://api.slack.com/apps e faça login no seu workspace do Slack.",
              },
              {
                "@type": "HowToStep",
                name: "Crie um novo app",
                text: "Clique no botão 'Create New App' e escolha 'From scratch'.",
              },
              {
                "@type": "HowToStep",
                name: "Configure as permissões do bot",
                text: "Vá a OAuth & Permissions e adicione os scopes necessários em Bot Token Scopes: chat:write, channels:join, im:write e groups:write.",
              },
              {
                "@type": "HowToStep",
                name: "Instale no workspace",
                text: "Instale o app no seu workspace e autorize-o.",
              },
              {
                "@type": "HowToStep",
                name: "Copie o token do bot",
                text: "Copie o Bot User OAuth Token, que começa com 'xoxb-'.",
              },
              {
                "@type": "HowToStep",
                name: "Obtenha o ID do canal",
                text: "Abra o canal de destino e obtenha o Channel ID nos detalhes do canal.",
              },
              {
                "@type": "HowToStep",
                name: "Adicione o bot ao canal privado",
                text: "Se você usar um canal privado, convide o bot para o canal mencionando-o.",
              },
              {
                "@type": "HowToStep",
                name: "Configure no Databasus",
                text: "No Databasus, adicione o Bot Token e o Channel ID na configuração do notificador Slack.",
              },
              {
                "@type": "HowToStep",
                name: "Teste a notificação",
                text: "Teste a notificação para garantir que está funcionando corretamente.",
              },
            ],
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
              <h1 id="slack-notifications">Notificações do Slack</h1>

              <p className="text-lg text-gray-400">
                Configure o Slack para receber notificações instantâneas sobre o
                status do seu backup PostgreSQL. Receba alertas de backups
                bem-sucedidos, falhas e avisos diretamente nos seus canais do
                Slack.
              </p>

              <h2 id="create-slack-app">Criar um app do Slack</h2>

              <h3 id="go-to-slack-api">1. Abra a API do Slack</h3>

              <p>
                Vá a{" "}
                <a
                  href="https://api.slack.com/apps"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://api.slack.com/apps
                </a>{" "}
                e faça login no seu workspace do Slack.
              </p>

              <h3 id="create-new-app">2. Crie um novo app</h3>

              <p>
                Clique no botão <strong>&quot;Create New App&quot;</strong>.
              </p>

              <h3 id="choose-from-scratch">
                3. Escolha &quot;From scratch&quot;
              </h3>

              <p>
                Selecione a opção <strong>&quot;From scratch&quot;</strong>{" "}
                quando solicitado.
              </p>

              <h3 id="name-your-app">4. Dê um nome ao app</h3>

              <p>
                Digite um nome para o app (por exemplo, &quot;Databasus
                Notifications&quot;) e selecione o workspace onde você quer
                instalá-lo. Clique em <strong>&quot;Create App&quot;</strong>.
              </p>

              <h2 id="configure-bot-permissions">
                Configurar as permissões do bot
              </h2>

              <h3 id="navigate-to-oauth">5. Vá a OAuth &amp; Permissions</h3>

              <p>
                Na barra lateral esquerda, clique em{" "}
                <strong>&quot;OAuth &amp; Permissions&quot;</strong>.
              </p>

              <img
                src="/images/notifier-slack/image-1.png"
                alt="Ir para OAuth &amp; Permissions"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="add-bot-scopes">
                6. Adicione os Bot Token Scopes (obrigatório)
              </h3>

              <p>
                Role até a seção <strong>&quot;Scopes&quot;</strong> e, em{" "}
                <strong>&quot;Bot Token Scopes&quot;</strong>, clique em{" "}
                <strong>&quot;Add an OAuth Scope&quot;</strong>.
              </p>

              <p>Adicione todos os scopes obrigatórios seguintes:</p>

              <ul>
                <li>
                  <code>chat:write</code> - para enviar mensagens para canais
                </li>
                <li>
                  <code>channels:join</code> - para permitir que o bot entre
                  automaticamente em canais públicos
                </li>
                <li>
                  <code>im:write</code> - para enviar mensagens diretas a
                  usuários
                </li>
                <li>
                  <code>groups:write</code> - para enviar mensagens para canais
                  privados
                </li>
                <li>
                  <code>channels:history</code> - para ler o histórico dos
                  canais
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-2.png"
                alt="Adicionar Bot Token Scopes"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h2 id="install-app">Instalar o app no workspace</h2>

              <h3 id="install-to-workspace">7. Instale no workspace</h3>

              <p>
                Volte ao topo da página{" "}
                <strong>&quot;OAuth &amp; Permissions&quot;</strong> e clique em{" "}
                <strong>&quot;Install to Workspace&quot;</strong>.
              </p>

              <img
                src="/images/notifier-slack/image-3.png"
                alt="Instalar no workspace"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="authorize-app">8. Autorize o app</h3>

              <p>
                Revise as permissões e clique em{" "}
                <strong>&quot;Allow&quot;</strong> para autorizar o app.
              </p>

              <h3 id="copy-bot-token">9. Copie o Bot User OAuth Token</h3>

              <p>
                Depois da instalação, você verá o{" "}
                <strong>&quot;Bot User OAuth Token&quot;</strong>. Começa com{" "}
                <code>xoxb-</code>. Copie este token: você vai precisar dele
                para configurar o Databasus.
              </p>

              <h2 id="get-channel-id">Obter o ID do canal</h2>

              <h3 id="open-channel">10. Abra o canal de destino</h3>

              <p>
                No seu workspace do Slack, abra o canal onde você quer receber
                as notificações de backup.
              </p>

              <h3 id="get-channel-info">11. Obtenha o ID do canal</h3>

              <p>
                Clique no nome do canal no topo e depois role pelos detalhes do
                canal. Você encontrará o <strong>Channel ID</strong> no fim da
                seção &quot;About&quot;. Começa com <code>C</code> (canais
                públicos) ou <code>G</code> (canais privados).
              </p>

              <p>Copie este Channel ID.</p>

              <img
                src="/images/notifier-slack/image-4.png"
                alt="Obter o Channel ID"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[500px]"
                loading="lazy"
              />

              <h3 id="add-bot-to-channel">
                12. Adicione o bot ao canal (obrigatório para canais privados)
              </h3>

              <p>
                <strong>
                  Se você estiver usando um canal privado, precisa convidar o
                  bot manualmente para o canal:
                </strong>
              </p>

              <ol>
                <li>
                  No canal privado, escreva{" "}
                  <code>@Databasus Notifications</code> (ou o nome que você deu
                  ao seu app)
                </li>
                <li>
                  Clique no nome do bot quando aparecer e selecione{" "}
                  <strong>&quot;Add to Channel&quot;</strong> ou{" "}
                  <strong>&quot;Invite to Channel&quot;</strong>
                </li>
              </ol>

              <p>
                Em <strong>canais públicos</strong>, o bot entra automaticamente
                ao enviar a primeira mensagem (graças à permissão{" "}
                <code>channels:join</code>), por isso este passo não é
                necessário.
              </p>

              <h2 id="configure-databasus">Configurar no Databasus</h2>

              <h3 id="add-slack-notifier">13. Adicione o notificador Slack</h3>

              <p>
                No Databasus, vá às configurações de notificadores e adicione um
                novo notificador Slack:
              </p>

              <ul>
                <li>
                  <strong>Bot Token:</strong> cole o Bot User OAuth Token que
                  você copiou (começa com <code>xoxb-</code>)
                </li>
                <li>
                  <strong>Target Channel ID:</strong> cole o Channel ID que você
                  copiou (começa com <code>C</code>, <code>G</code>,{" "}
                  <code>D</code> ou <code>U</code>)
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-5.png"
                alt="Adicionar notificador Slack"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="test-notification">14. Teste a notificação</h3>

              <p>
                Depois de configurar o notificador, teste-o para garantir que
                está funcionando corretamente. Você deverá receber uma mensagem
                de teste no canal do Slack selecionado.
              </p>

              <p>
                E pronto! O seu workspace do Slack está agora configurado para
                receber notificações de backup PostgreSQL do Databasus.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/pt/notifiers"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Voltar aos notificadores
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
