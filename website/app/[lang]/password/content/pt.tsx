import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Redefinir senha - Documentação do Databasus",
  description:
    "Aprenda a redefinir senhas de usuários no Databasus com a ferramenta de linha de comando integrada. Recuperação de senha rápida e segura para o seu sistema de backup PostgreSQL.",
  keywords: [
    "redefinir senha Databasus",
    "redefinir senha de usuário",
    "senha de backup PostgreSQL",
    "recuperação de senha no Docker",
    "recuperação de senha",
    "autenticação Databasus",
  ],
  openGraph: {
    title: "Redefinir senha - Documentação do Databasus",
    description:
      "Aprenda a redefinir senhas de usuários no Databasus com a ferramenta de linha de comando integrada. Recuperação de senha rápida e segura para o seu sistema de backup PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("pt", "password"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Redefinir senha - Documentação do Databasus",
    description:
      "Aprenda a redefinir senhas de usuários no Databasus com a ferramenta de linha de comando integrada. Recuperação de senha rápida e segura para o seu sistema de backup PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "password"),
    languages: getLanguageAlternates("password"),
  },
  robots: "index, follow",
};

export default function PasswordResetPage() {
  const resetPasswordCommand = `docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="admin"`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Redefinir senha - Documentação do Databasus",
            description:
              "Aprenda a redefinir senhas de usuários no Databasus com a ferramenta de linha de comando integrada.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Como redefinir a senha de um usuário do Databasus",
            description:
              "Guia passo a passo para redefinir senhas de usuários no Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Execute o comando de redefinição de senha",
                text: "Execute o comando docker exec com a nova senha e o email do usuário.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Use docker exec para executar o comando de redefinição de senha dentro do container do Databasus",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Verifique a alteração da senha",
                text: "Faça login no Databasus com a nova senha para confirmar que a alteração foi bem-sucedida.",
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
              <h1 id="reset-password">Redefinir a senha de um usuário</h1>

              <h2 id="reset-password-command">
                Comando de redefinição de senha
              </h2>

              <p>
                Para redefinir a senha de um usuário, use o seguinte comando no
                servidor onde o Databasus está rodando:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={resetPasswordCommand} lang="pt" />
                </div>
              </div>

              <h2 id="parameters">Parâmetros</h2>

              <p>O comando aceita os seguintes parâmetros:</p>

              <ul>
                <li>
                  <strong>--new-password</strong>: a nova senha. Garanta que é
                  segura e contém uma mistura de letras, números e caracteres
                  especiais.
                </li>
                <li>
                  <strong>--email</strong>: o endereço de email do usuário cuja
                  senha você quer redefinir (por exemplo, <code>admin</code>,{" "}
                  <code>user@example.com</code>).
                </li>
              </ul>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
