import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Como fazer backup do Supabase com o Databasus | Databasus",
  description:
    "Aprenda a fazer backup da sua base de dados PostgreSQL do Supabase com o Databasus. Guia passo a passo para configurar o session pooler ou o endereço IPv4 para backups do Supabase.",
  keywords: [
    "Databasus",
    "backup Supabase",
    "backup PostgreSQL Supabase",
    "backup de base de dados Supabase",
    "session pooler Supabase",
    "IPv4 Supabase",
    "backup PostgreSQL",
    "backup de base de dados",
  ],
  openGraph: {
    title: "Como fazer backup do Supabase com o Databasus | Databasus",
    description:
      "Aprenda a fazer backup da sua base de dados PostgreSQL do Supabase com o Databasus. Guia passo a passo para configurar o session pooler ou o endereço IPv4 para backups do Supabase.",
    type: "article",
    url: getLocalizedUrl("pt", "faq/supabase"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Como fazer backup do Supabase com o Databasus | Databasus",
    description:
      "Aprenda a fazer backup da sua base de dados PostgreSQL do Supabase com o Databasus. Guia passo a passo para configurar o session pooler ou o endereço IPv4 para backups do Supabase.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "faq/supabase"),
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
            name: "Como fazer backup do Supabase com o Databasus",
            description:
              "Guia passo a passo para fazer backup da sua base de dados PostgreSQL do Supabase com o Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Obtenha os dados de conexão no Supabase",
                text: "Vá às configurações do seu projeto Supabase e encontre os dados de conexão da base de dados.",
              },
              {
                "@type": "HowToStep",
                name: "Use o Session Pooler com IPv4",
                text: "Copie a connection string do Session Pooler e garanta que a opção 'Use IPv4 Address' está ativada.",
              },
              {
                "@type": "HowToStep",
                name: "Configure o Databasus",
                text: "Insira os dados de conexão do Supabase no Databasus para começar a fazer backup da sua base de dados.",
              },
              {
                "@type": "HowToStep",
                name: "Entenda as limitações de schemas",
                text: "Por padrão, só o schema public é incluído no backup, porque os demais schemas do Supabase são restritos.",
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
              <h1 id="supabase-backup">Como fazer backup do Supabase</h1>

              <p className="text-lg text-gray-400">
                O Databasus suporta backups de bases de dados PostgreSQL do
                Supabase. O requisito principal é usar um endereço IPv4 para se
                conectar à sua instância do Supabase.
              </p>

              <h2 id="connection-options">Opções de conexão</h2>

              <p>
                Há duas formas de conectar o Databasus à sua base de dados do
                Supabase:
              </p>

              <ol>
                <li>
                  <strong>Usar o Session Pooler com IPv4</strong> (recomendado)
                  - opção gratuita, disponível em todos os projetos Supabase
                </li>
                <li>
                  <strong>Comprar o add-on IPv4</strong> - opção de conexão
                  direta do Supabase
                </li>
              </ol>

              <h2 id="session-pooler">
                Opção 1: usar o Session Pooler (recomendado)
              </h2>

              <p>
                O Session Pooler fornece um endereço IPv4 para a conexão com a
                sua base de dados do Supabase sem custo adicional. Veja como
                configurá-lo:
              </p>

              <h3 id="step-1">1. Encontre a conexão do pooler</h3>

              <p>
                Abra o seu projeto Supabase e vá a{" "}
                <strong>Project Settings</strong> → <strong>Database</strong>.
                Role até a seção <strong>Connection string</strong> e selecione
                o modo <strong>Session pooler</strong>.
              </p>

              <img
                src="/images/faq/supabase/image-1.png"
                alt="Selecionar o modo Session pooler no Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h3 id="step-2">2. Copie os dados de conexão</h3>

              <p>
                Copie os dados de conexão e use-os no Databasus ao adicionar a
                sua base de dados. Veja a imagem para identificar cada campo da
                conexão.
              </p>

              <img
                src="/images/faq/supabase/image-2.png"
                alt="Ativar a opção IPv4 Address no Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h2 id="ipv4-addon">Opção 2: comprar o add-on IPv4</h2>

              <p>
                O Supabase oferece um add-on IPv4 pago que fornece um endereço
                IPv4 dedicado para a sua base de dados. Essa opção oferece uma
                conexão direta sem passar pelo connection pooler.
              </p>

              <p>Para ativar esta opção:</p>

              <ol>
                <li>Vá ao painel do seu projeto Supabase</li>
                <li>
                  Vá a <strong>Project Settings</strong> →{" "}
                  <strong>Add-ons</strong>
                </li>
                <li>
                  Ative o add-on <strong>IPv4</strong>
                </li>
                <li>
                  Use os dados de conexão direta com a base de dados no
                  Databasus
                </li>
              </ol>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Dica:</strong> para a
                  maioria dos casos, o Session Pooler gratuito com IPv4 funciona
                  perfeitamente para backups. O add-on IPv4 pago só é necessário
                  se você precisar de uma conexão direta por outros motivos.
                </p>
              </div>

              <h2 id="default-schema">Limitação do schema padrão</h2>

              <p>
                Por padrão, o Databasus faz backup apenas do schema{" "}
                <code>public</code> quando trabalha com bases de dados do
                Supabase. Isto acontece porque o Supabase restringe o acesso aos
                outros schemas (como <code>auth</code>, <code>storage</code> e{" "}
                <code>realtime</code>) por razões de segurança.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-blue-400">ℹ️ Nota:</strong> o schema{" "}
                  <code>public</code> contém os dados da sua aplicação e as suas
                  tabelas personalizadas. Os schemas como <code>auth</code> e{" "}
                  <code>storage</code> são protegidos e gerenciados pelo próprio
                  Supabase.
                </p>
              </div>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/pt/faq"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Voltar às perguntas frequentes
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
