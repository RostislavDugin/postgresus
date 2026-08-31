import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Réinitialiser le mot de passe - Documentation Databasus",
  description:
    "Apprenez à réinitialiser les mots de passe des utilisateurs dans Databasus avec l'outil en ligne de commande intégré. Récupération rapide et sécurisée pour votre système de sauvegarde PostgreSQL.",
  keywords: [
    "réinitialisation mot de passe Databasus",
    "réinitialiser mot de passe utilisateur",
    "mot de passe sauvegarde PostgreSQL",
    "récupération mot de passe Docker",
    "récupération de mot de passe",
    "authentification Databasus",
  ],
  openGraph: {
    title: "Réinitialiser le mot de passe - Documentation Databasus",
    description:
      "Apprenez à réinitialiser les mots de passe des utilisateurs dans Databasus avec l'outil en ligne de commande intégré. Récupération rapide et sécurisée pour votre système de sauvegarde PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("fr", "password"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Réinitialiser le mot de passe - Documentation Databasus",
    description:
      "Apprenez à réinitialiser les mots de passe des utilisateurs dans Databasus avec l'outil en ligne de commande intégré. Récupération rapide et sécurisée pour votre système de sauvegarde PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "password"),
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
            headline: "Réinitialiser le mot de passe - Documentation Databasus",
            description:
              "Apprenez à réinitialiser les mots de passe des utilisateurs dans Databasus avec l'outil en ligne de commande intégré.",
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
            name: "Comment réinitialiser le mot de passe d'un utilisateur Databasus",
            description:
              "Guide pas à pas pour réinitialiser les mots de passe des utilisateurs dans Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Exécuter la commande de réinitialisation",
                text: "Exécutez la commande docker exec avec votre nouveau mot de passe et l'e-mail de l'utilisateur.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Utilisez docker exec pour lancer la commande de réinitialisation du mot de passe dans le conteneur Databasus",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Vérifier le changement de mot de passe",
                text: "Connectez-vous à Databasus avec votre nouveau mot de passe pour confirmer que le changement a réussi.",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="fr" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="fr" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="reset-password">
                Réinitialiser le mot de passe d&apos;un utilisateur
              </h1>

              <h2 id="reset-password-command">
                Commande de réinitialisation du mot de passe
              </h2>

              <p>
                Pour réinitialiser le mot de passe d&apos;un utilisateur,
                utilisez la commande suivante sur le serveur où Databasus est en
                cours d&apos;exécution :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={resetPasswordCommand} />
                </div>
              </div>

              <h2 id="parameters">Paramètres</h2>

              <p>La commande accepte les paramètres suivants :</p>

              <ul>
                <li>
                  <strong>--new-password</strong> : le nouveau mot de passe.
                  Assurez-vous qu&apos;il est robuste et contient un mélange de
                  lettres, de chiffres et de caractères spéciaux.
                </li>
                <li>
                  <strong>--email</strong> : l&apos;adresse e-mail de
                  l&apos;utilisateur dont vous voulez réinitialiser le mot de
                  passe (par exemple <code>admin</code>,{" "}
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
