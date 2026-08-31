import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "Como usar o Databasus com Cloudflare R2 | Databasus",
  description:
    "Guia passo a passo para configurar o armazenamento Cloudflare R2 para backups PostgreSQL com o Databasus. Aprenda a configurar armazenamento compatível com S3 no R2.",
  keywords: [
    "Databasus",
    "Cloudflare R2",
    "backup PostgreSQL",
    "armazenamento S3",
    "armazenamento na nuvem",
    "backup de base de dados",
  ],
  openGraph: {
    title: "Como usar o Databasus com Cloudflare R2 | Databasus",
    description:
      "Guia passo a passo para configurar o armazenamento Cloudflare R2 para backups PostgreSQL com o Databasus. Aprenda a configurar armazenamento compatível com S3 no R2.",
    type: "article",
    url: getLocalizedUrl("pt", "storages/cloudflare-r2"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Como usar o Databasus com Cloudflare R2 | Databasus",
    description:
      "Guia passo a passo para configurar o armazenamento Cloudflare R2 para backups PostgreSQL com o Databasus. Aprenda a configurar armazenamento compatível com S3 no R2.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "storages/cloudflare-r2"),
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
            name: "Como usar o Databasus com Cloudflare R2",
            description:
              "Guia passo a passo para configurar o armazenamento Cloudflare R2 para backups PostgreSQL com o Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Preencha o nome do bucket",
                text: "Insira o nome do seu bucket R2 na configuração do armazenamento.",
              },
              {
                "@type": "HowToStep",
                name: "Defina a região",
                text: 'No campo da região, preencha "auto"',
              },
              {
                "@type": "HowToStep",
                name: "Gere um access key ID e um secret access key",
                text: "No painel da Cloudflare, vá a R2 → API → Manage API Tokens. Crie o token e atribua a ele as permissões necessárias.",
              },
              {
                "@type": "HowToStep",
                name: "Encontre o seu account ID",
                text: "Em qualquer página do R2 no painel, você verá o seu Account ID perto do topo.",
              },
              {
                "@type": "HowToStep",
                name: "Monte o endpoint S3",
                text: "Substitua <ACCOUNT_ID> pelo valor do seu painel no formato: https://<ACCOUNT_ID>.r2.cloudflarestorage.com",
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
              <h1 id="cloudflare-r2">Armazenamento Cloudflare R2</h1>

              <p className="text-lg text-gray-400">
                Para usar o Cloudflare R2 como armazenamento compatível com S3
                para os seus backups PostgreSQL, você precisa configurar as
                credenciais do seu bucket R2 e o endpoint.
              </p>

              <h2 id="configuration-steps">Passos de configuração</h2>

              <h3 id="fill-bucket-name">1. Preencha o nome do bucket</h3>

              <p>
                Insira o nome do seu bucket R2 na configuração do armazenamento:
              </p>

              <Image
                src="/images/cloudflare-r2-storage/image-1.webp"
                alt="Preencher o nome do bucket no Cloudflare R2"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="set-region">2. Defina a região</h3>

              <p>
                No campo da região, preencha <code>&quot;auto&quot;</code>
              </p>

              <h3 id="generate-access-key">
                3. Gere um Access Key ID e um Secret Access Key
              </h3>

              <p>
                No painel da Cloudflare, vá a{" "}
                <strong>R2 → API → Manage API Tokens</strong>. Crie um novo
                token e atribua a ele as permissões necessárias (por exemplo,{" "}
                <strong>&quot;Object Read &amp; Write&quot;</strong>).
              </p>

              <p>Quando o token for criado, você verá:</p>

              <ul>
                <li>
                  <strong>Access Key ID</strong> (o ID do token)
                </li>
                <li>
                  <strong>Secret Access Key</strong> (o hash SHA-256 do valor do
                  token)
                </li>
              </ul>

              <p>Copie ambos os valores para o Databasus:</p>

              <Image
                src="/images/cloudflare-r2-storage/image-2.gif"
                alt="Gerar Access Key ID e Secret Access Key"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="find-account-id">4. Encontre o seu account ID</h3>

              <p>
                Em qualquer página do R2 no painel, você verá o seu Account ID
                perto do topo (ou nas configurações da conta):
              </p>

              <Image
                src="/images/cloudflare-r2-storage/image-3.webp"
                alt="Encontrar o Account ID no painel da Cloudflare"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="construct-endpoint">5. Monte o endpoint S3</h3>

              <p>Use o seguinte formato para o seu endpoint S3:</p>

              <pre>
                <code>https://&lt;ACCOUNT_ID&gt;.r2.cloudflarestorage.com</code>
              </pre>

              <p>
                Substitua <code>&lt;ACCOUNT_ID&gt;</code> pelo valor do seu
                painel e insira-o no Databasus.
              </p>

              <p>E pronto! A sua configuração deverá agora ficar assim:</p>

              <Image
                src="/images/cloudflare-r2-storage/image-4.png"
                alt="Configuração concluída"
                width={500}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                O seu Databasus está agora pronto para usar o Cloudflare R2 como
                armazenamento para os seus backups PostgreSQL.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/pt/storages"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Voltar aos armazenamentos
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
