import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title:
    "Como configurar notificações do Microsoft Teams no Databasus | Databasus",
  description:
    "Guia passo a passo para configurar notificações do Microsoft Teams para alertas de backup PostgreSQL com o Databasus. Aprenda a criar um webhook do Teams e configurar as notificações.",
  keywords: [
    "Databasus",
    "notificações Microsoft Teams",
    "backup PostgreSQL",
    "webhook Teams",
    "alertas de backup",
    "notificações de base de dados",
  ],
  openGraph: {
    title:
      "Como configurar notificações do Microsoft Teams no Databasus | Databasus",
    description:
      "Guia passo a passo para configurar notificações do Microsoft Teams para alertas de backup PostgreSQL com o Databasus. Aprenda a criar um webhook do Teams e configurar as notificações.",
    type: "article",
    url: getLocalizedUrl("pt", "notifiers/teams"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title:
      "Como configurar notificações do Microsoft Teams no Databasus | Databasus",
    description:
      "Guia passo a passo para configurar notificações do Microsoft Teams para alertas de backup PostgreSQL com o Databasus. Aprenda a criar um webhook do Teams e configurar as notificações.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "notifiers/teams"),
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
            name: "Como configurar notificações do Microsoft Teams no Databasus",
            description:
              "Guia passo a passo para configurar notificações do Microsoft Teams para alertas de backup PostgreSQL com o Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Abra o canal do Teams",
                text: "Vá ao canal do Microsoft Teams onde você quer receber as notificações.",
              },
              {
                "@type": "HowToStep",
                name: "Abra os workflows",
                text: "Abra a funcionalidade Workflows no seu canal do Teams.",
              },
              {
                "@type": "HowToStep",
                name: "Crie um novo workflow",
                text: "Crie um novo workflow para webhooks recebidos.",
              },
              {
                "@type": "HowToStep",
                name: "Selecione o modelo de webhook",
                text: "Escolha o modelo de webhook recebido entre as opções disponíveis.",
              },
              {
                "@type": "HowToStep",
                name: "Configure o webhook",
                text: "Defina o nome do webhook e o canal.",
              },
              {
                "@type": "HowToStep",
                name: "Copie a URL do webhook",
                text: "Copie a URL de webhook gerada pelo Teams.",
              },
              {
                "@type": "HowToStep",
                name: "Configure no Databasus",
                text: "Cole a URL do webhook na configuração do notificador do Databasus.",
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
              <h1 id="teams-notifications">Notificações do Microsoft Teams</h1>

              <p className="text-lg text-gray-400">
                Configure o Microsoft Teams para receber notificações
                instantâneas sobre o status do seu backup PostgreSQL. Receba
                alertas de backups bem-sucedidos, falhas e avisos diretamente
                nos seus canais do Teams.
              </p>

              <h2 id="setup-teams-webhook">Configurar o webhook do Teams</h2>

              <h3 id="open-teams-channel">1. Abra o seu canal do Teams</h3>

              <p>
                Vá ao canal do Microsoft Teams onde você quer receber as
                notificações de backup. Clique nos três pontos (
                <strong>•••</strong>) ao lado do nome do canal.
              </p>

              <Image
                src="/images/notifier-teams/image-01.png"
                alt="Abrir canal do Teams"
                width={800}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="access-workflows">2. Abra os workflows</h3>

              <p>
                No menu do canal, selecione{" "}
                <strong>&quot;Workflows&quot;</strong> para abrir a integração
                com o Power Automate.
              </p>

              <Image
                src="/images/notifier-teams/image-02.png"
                alt="Abrir Workflows"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-new-workflow">3. Crie um novo workflow</h3>

              <p>
                No painel de Workflows, clique em{" "}
                <strong>&quot;Create&quot;</strong> ou procure o modelo{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>
                .
              </p>

              <Image
                src="/images/notifier-teams/image-03.png"
                alt="Criar novo workflow"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="select-webhook-template">
                4. Selecione o modelo de webhook
              </h3>

              <p>
                Escolha o modelo{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>{" "}
                entre as opções disponíveis.
              </p>

              <Image
                src="/images/notifier-teams/image-04.png"
                alt="Selecionar modelo de webhook"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-webhook">5. Configure o webhook</h3>

              <p>
                Configure o webhook indicando um nome (por exemplo,{" "}
                <strong>&quot;Databasus Backup Notifications&quot;</strong>) e
                confirme o canal onde as notificações serão publicadas.
              </p>

              <Image
                src="/images/notifier-teams/image-05.png"
                alt="Configurar webhook"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="copy-webhook-url">6. Copie a URL do webhook</h3>

              <p>
                Depois de criar o workflow, você verá o{" "}
                <strong>HTTP POST URL</strong>. Copie essa URL: você vai
                precisar dela para configurar o Databasus.
              </p>

              <Image
                src="/images/notifier-teams/image-06.png"
                alt="Copiar URL do webhook"
                width={500}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="configure-databasus">Configurar no Databasus</h2>

              <h3 id="add-teams-notifier">1. Adicione o notificador Teams</h3>

              <p>
                No Databasus, vá às configurações de notificadores e adicione um
                novo notificador Microsoft Teams. Cole a URL do webhook que você
                copiou do Teams.
              </p>

              <Image
                src="/images/notifier-teams/image-07.png"
                alt="Configurar Teams no Databasus"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="test-notification">2. Teste a notificação</h3>

              <p>
                Depois de configurar o webhook, teste a notificação para
                garantir que está funcionando corretamente. Você deverá receber
                uma mensagem de teste no canal do Teams selecionado.
              </p>

              <p>
                E pronto! O seu canal do Microsoft Teams está agora configurado
                para receber notificações de backup PostgreSQL do Databasus.
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
