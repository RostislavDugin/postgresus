import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "Como fazer backup de bases de dados no localhost | Databasus",
  description:
    "Aprenda a fazer backup PostgreSQL de bases de dados rodando no localhost com o Databasus. Configure o modo de rede host do Docker para backups de bases de dados locais.",
  keywords: [
    "Databasus",
    "backup localhost",
    "backup PostgreSQL local",
    "backup de base de dados local",
    "rede host Docker",
    "backup PostgreSQL",
    "backup de base de dados",
    "base de dados localhost",
  ],
  openGraph: {
    title: "Como fazer backup de bases de dados no localhost | Databasus",
    description:
      "Aprenda a fazer backup PostgreSQL de bases de dados rodando no localhost com o Databasus. Configure o modo de rede host do Docker para backups de bases de dados locais.",
    type: "article",
    url: getLocalizedUrl("pt", "faq/localhost"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Como fazer backup de bases de dados no localhost | Databasus",
    description:
      "Aprenda a fazer backup PostgreSQL de bases de dados rodando no localhost com o Databasus. Configure o modo de rede host do Docker para backups de bases de dados locais.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "faq/localhost"),
    languages: getLanguageAlternates("faq/localhost"),
  },
  robots: "index, follow",
};

export default function LocalhostPage() {
  const dockerComposeHost = `services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    network_mode: host
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped`;

  const dockerRunHost = `docker run -d \\
  --name databasus \\
  --network host \\
  -v ./databasus-data:/databasus-data \\
  --restart unless-stopped \\
  databasus/databasus:latest`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Como fazer backup de bases de dados no localhost com o Databasus",
            description:
              "Guia passo a passo para fazer backup de bases de dados PostgreSQL rodando no localhost com o Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Configure o modo de rede host do Docker",
                text: "Atualize a configuração do Docker para usar o modo de rede host, para que o container possa acessar os serviços no localhost.",
              },
              {
                "@type": "HowToStep",
                name: "Use Docker Compose ou Docker run",
                text: "Aplique a definição network_mode: host no Docker Compose ou use a flag --network host com o Docker run.",
              },
              {
                "@type": "HowToStep",
                name: "Conecte-se à base de dados no localhost",
                text: "Use 127.0.0.1 ou localhost como host da base de dados na configuração de backup do Databasus.",
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
              <h1 id="localhost-backup">
                Como fazer backup de bases de dados no localhost
              </h1>

              <p className="text-lg text-gray-400">
                Aprenda a configurar o Databasus para fazer cópias de segurança
                de bases de dados PostgreSQL rodando na sua máquina host
                (localhost) quando você usa Docker.
              </p>

              <h2 id="the-problem">O problema</h2>

              <p>
                Se você roda o Databasus em Docker e quer fazer backup de bases
                de dados rodando na sua máquina host (localhost), precisa
                configurar o Docker para usar o{" "}
                <strong>modo de rede host</strong>.
              </p>

              <p>
                Por padrão, os containers do Docker rodam numa rede isolada e
                não conseguem acessar os serviços no <code>localhost</code>. O
                modo de rede host permite que o container compartilhe o
                namespace de rede do host.
              </p>

              <h2 id="docker-compose-solution">Solução para Docker Compose</h2>

              <p>
                Atualize o seu <code>docker-compose.yml</code> para usar{" "}
                <code>network_mode: host</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerComposeHost} lang="pt" />
                </div>
              </div>

              <h2 id="docker-run-solution">Solução para Docker run</h2>

              <p>
                Use a flag <code>--network host</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRunHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerRunHost} lang="pt" />
                </div>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Nota:</strong> ao usar o
                  modo de rede host, você pode se conectar à sua base de dados
                  no localhost usando{" "}
                  <code className="bg-[#374151] text-gray-200">127.0.0.1</code>{" "}
                  ou{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>{" "}
                  como host na configuração de backup do Databasus. Você também
                  acessará a interface do Databasus diretamente em{" "}
                  <code className="bg-[#374151] text-gray-200">
                    http://localhost:4005
                  </code>{" "}
                  sem mapeamento de portas.
                </p>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">
                    ⚠️ Importante para quem usa Windows e macOS:
                  </strong>{" "}
                  o modo de rede{" "}
                  <code className="bg-[#374151] text-red-400">host</code> só
                  funciona nativamente em Linux. No Windows e no macOS, o Docker
                  roda dentro de uma VM Linux, por isso você deve usar{" "}
                  <code className="bg-[#374151] text-gray-200">
                    host.docker.internal
                  </code>{" "}
                  em vez de{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>{" "}
                  como endereço do host da base de dados na sua configuração de
                  backup.
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
