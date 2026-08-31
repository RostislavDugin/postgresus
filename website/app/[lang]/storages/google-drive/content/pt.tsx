import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "Como conectar o Google Drive ao Databasus | Databasus",
  description:
    "Guia passo a passo para configurar o armazenamento no Google Drive para backups PostgreSQL com o Databasus. Aprenda a criar um projeto no Google Cloud e configurar o OAuth.",
  keywords: [
    "Databasus",
    "Google Drive",
    "backup PostgreSQL",
    "Google Cloud",
    "OAuth",
    "armazenamento na nuvem",
    "backup de base de dados",
  ],
  openGraph: {
    title: "Como conectar o Google Drive ao Databasus | Databasus",
    description:
      "Guia passo a passo para configurar o armazenamento no Google Drive para backups PostgreSQL com o Databasus. Aprenda a criar um projeto no Google Cloud e configurar o OAuth.",
    type: "article",
    url: getLocalizedUrl("pt", "storages/google-drive"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Como conectar o Google Drive ao Databasus | Databasus",
    description:
      "Guia passo a passo para configurar o armazenamento no Google Drive para backups PostgreSQL com o Databasus. Aprenda a criar um projeto no Google Cloud e configurar o OAuth.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "storages/google-drive"),
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
            name: "Como conectar o Google Drive ao Databasus",
            description:
              "Guia passo a passo para configurar o armazenamento no Google Drive para backups PostgreSQL com o Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Crie um novo projeto",
                text: "Vá à Google Cloud Console e crie um novo projeto.",
              },
              {
                "@type": "HowToStep",
                name: "Ative a Google Drive API",
                text: "Vá à aba API & Services, depois à API library, e ative a Google Drive API.",
              },
              {
                "@type": "HowToStep",
                name: "Configure a página de consentimento",
                text: "Vá a Credentials → Create credentials → Configure consent screen e preencha os dados obrigatórios.",
              },
              {
                "@type": "HowToStep",
                name: "Crie o OAuth client ID",
                text: "Vá a Credentials → Create credentials → OAuth client ID.",
              },
              {
                "@type": "HowToStep",
                name: "Configure a aplicação",
                text: "Defina o tipo de aplicação como Web application e configure as origens autorizadas e os URIs de redirecionamento.",
              },
              {
                "@type": "HowToStep",
                name: "Adicione o scope",
                text: 'Vá a Data Access e adicione o scope "/auth/drive.file".',
              },
              {
                "@type": "HowToStep",
                name: "Publique a app",
                text: "Vá a Audience e publique a app.",
              },
              {
                "@type": "HowToStep",
                name: "Faça login com a conta Google",
                text: "Preencha as credenciais e faça login com a sua conta Google.",
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
              <h1 id="google-drive">Armazenamento no Google Drive</h1>

              <p className="text-lg text-gray-400">
                Para salvar os seus backups no Google Drive, você precisa criar
                um projeto no Google Cloud para acessar a Google Drive API e
                depois fazer login com a sua conta Google.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Importante:</strong> para
                  conectar o Google Drive, a sua instância do Databasus precisa
                  estar funcionando com HTTPS. Para mais informações sobre como
                  configurar HTTPS, veja o{" "}
                  <a
                    href="/pt/installation#caddy-reverse-proxy"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    guia de configuração do reverse proxy Caddy →
                  </a>
                  <br />
                  Neste guia, assumimos que a sua instância do Databasus está
                  disponível em <code>databasus.yourdomain.com</code>.
                </p>
              </div>

              <h2 id="create-google-cloud-project">
                Criar o projeto no Google Cloud
              </h2>

              <h3 id="create-new-project">1. Crie um novo projeto</h3>

              <p>
                Vá a{" "}
                <a
                  href="https://console.cloud.google.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://console.cloud.google.com/
                </a>{" "}
                e escolha <strong>&quot;new project&quot;</strong> (no canto
                superior esquerdo).
              </p>

              <h3 id="enable-google-drive-api">2. Ative a Google Drive API</h3>

              <p>
                Vá à aba <strong>&quot;API &amp; Services&quot;</strong> e
                depois a <strong>&quot;API library&quot;</strong>. Escolha{" "}
                <strong>Google Drive API</strong> e ative-a:
              </p>

              <Image
                src="/images/google-drive-storage/image-1.webp"
                alt="Ativar a Google Drive API"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-consent-screen">
                3. Configure a página de consentimento
              </h3>

              <p>
                Vá a <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;Configure consent screen&quot;</strong> e preencha
                os dados solicitados:
              </p>

              <Image
                src="/images/google-drive-storage/image-2.webp"
                alt="Configurar a página de consentimento"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-oauth-client-id">4. Crie o OAuth client ID</h3>

              <p>
                Vá a <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;OAuth client ID&quot;</strong>:
              </p>

              <Image
                src="/images/google-drive-storage/image-3.webp"
                alt="Criar OAuth client ID"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-application-settings">
                5. Configure a aplicação
              </h3>

              <p>Preencha os dados seguintes:</p>

              <ul>
                <li>
                  <strong>Application type:</strong> Web application
                </li>
                <li>
                  <strong>Authorized JavaScript origins:</strong>{" "}
                  <code>https://databasus.yourdomain.com</code>
                </li>
                <li>
                  <strong>Authorized redirect URIs:</strong>{" "}
                  <code>
                    https://databasus.yourdomain.com/storages/google-oauth
                  </code>
                </li>
              </ul>

              <p>
                <strong>Nota:</strong> substitua{" "}
                <code>databasus.yourdomain.com</code> pelo domínio real onde o
                Databasus está hospedado (por exemplo,{" "}
                <code>backup.example.com</code>).
              </p>

              <p>Depois copie as credenciais:</p>

              <Image
                src="/images/google-drive-storage/image-4.png"
                alt="Configurar a aplicação - parte 1"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <Image
                src="/images/google-drive-storage/image-5.png"
                alt="Configurar a aplicação - parte 2"
                width={450}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="add-scope">6. Adicione o scope</h3>

              <p>
                Vá a <strong>&quot;Data Access&quot;</strong> e adicione o scope{" "}
                <code>&quot;/auth/drive.file&quot;</code>:
              </p>

              <Image
                src="/images/google-drive-storage/image-6.png"
                alt="Adicionar scope"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="publish-app">7. Publique a app</h3>

              <p>
                Vá a <strong>&quot;Audience&quot;</strong> e publique a app:
              </p>

              <Image
                src="/images/google-drive-storage/image-7.png"
                alt="Publicar a app"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="sign-in-google-account">
                Fazer login com a conta Google
              </h2>

              <h3 id="fill-credentials">1. Preencha as credenciais</h3>

              <p>
                Preencha no Databasus as credenciais obtidas nos passos
                anteriores:
              </p>

              <Image
                src="/images/google-drive-storage/image-8.png"
                alt="Preencher credenciais"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="choose-account">2. Escolha a sua conta</h3>

              <p>Escolha a sua conta Google para fazer login.</p>

              <h3 id="handle-security-warning">
                3. Lide com o aviso de segurança
              </h3>

              <p>
                Se aparecer um aviso, clique em{" "}
                <strong>&quot;Advanced&quot;</strong> (canto inferior esquerdo)
                e escolha <strong>&quot;Proceed anyway&quot;</strong>.
              </p>

              <p>
                <strong>Nota:</strong> este aviso aparece porque o seu app ainda
                não foi verificado pelo Google. É seguro continuar quando a
                aplicação é sua.
              </p>

              <p>
                E pronto! O seu Google Drive está agora conectado ao Databasus e
                pronto para armazenar os seus backups PostgreSQL.
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
