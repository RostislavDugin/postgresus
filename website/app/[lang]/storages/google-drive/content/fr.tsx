import type { Metadata } from "next";
import {
  OG_LOCALES,
  getLanguageAlternates,
  getLocalizedUrl,
} from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "Comment connecter Google Drive à Databasus | Databasus",
  description:
    "Guide pas à pas pour configurer le stockage Google Drive pour les sauvegardes PostgreSQL avec Databasus. Apprenez à créer un projet Google Cloud et à configurer OAuth.",
  keywords: [
    "Databasus",
    "Google Drive",
    "sauvegarde PostgreSQL",
    "Google Cloud",
    "OAuth",
    "stockage cloud",
    "sauvegarde de base de données",
  ],
  openGraph: {
    title: "Comment connecter Google Drive à Databasus | Databasus",
    description:
      "Guide pas à pas pour configurer le stockage Google Drive pour les sauvegardes PostgreSQL avec Databasus. Apprenez à créer un projet Google Cloud et à configurer OAuth.",
    type: "article",
    url: getLocalizedUrl("fr", "storages/google-drive"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Comment connecter Google Drive à Databasus | Databasus",
    description:
      "Guide pas à pas pour configurer le stockage Google Drive pour les sauvegardes PostgreSQL avec Databasus. Apprenez à créer un projet Google Cloud et à configurer OAuth.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "storages/google-drive"),
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
            name: "Comment connecter Google Drive à Databasus",
            description:
              "Guide pas à pas pour configurer le stockage Google Drive pour les sauvegardes PostgreSQL avec Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Créer un nouveau projet",
                text: "Rendez-vous dans la Google Cloud Console et créez un nouveau projet.",
              },
              {
                "@type": "HowToStep",
                name: "Activer l'API Google Drive",
                text: "Allez dans l'onglet API & Services, puis dans la bibliothèque d'API et activez l'API Google Drive.",
              },
              {
                "@type": "HowToStep",
                name: "Configurer l'écran de consentement",
                text: "Allez dans Credentials → Create credentials → Configure consent screen et remplissez les données requises.",
              },
              {
                "@type": "HowToStep",
                name: "Créer un ID client OAuth",
                text: "Allez dans Credentials → Create credentials → OAuth client ID.",
              },
              {
                "@type": "HowToStep",
                name: "Configurer les paramètres de l'application",
                text: "Définissez le type d'application sur Web application et configurez les origines autorisées et les URI de redirection.",
              },
              {
                "@type": "HowToStep",
                name: "Ajouter le scope",
                text: 'Allez dans Data Access et ajoutez le scope "/auth/drive.file".',
              },
              {
                "@type": "HowToStep",
                name: "Publier l'application",
                text: "Allez dans Audience et publiez l'application.",
              },
              {
                "@type": "HowToStep",
                name: "Se connecter via un compte Google",
                text: "Remplissez les identifiants et connectez-vous avec votre compte Google.",
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
              <h1 id="google-drive">Stockage Google Drive</h1>

              <p className="text-lg text-gray-400">
                Pour conserver vos sauvegardes dans Google Drive, vous devez
                créer un projet Google Cloud afin d&apos;accéder à l&apos;API
                Google Drive, puis vous connecter via votre compte Google.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Important :</strong> pour
                  connecter Google Drive, votre instance Databasus doit
                  fonctionner en HTTPS. Pour en savoir plus sur la mise en place
                  du HTTPS, consultez le{" "}
                  <a
                    href="/fr/installation#caddy-reverse-proxy"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    guide de configuration du reverse proxy Caddy →
                  </a>
                  <br />
                  Pour ce guide, nous supposerons que votre instance Databasus
                  est accessible sur <code>databasus.yourdomain.com</code>.
                </p>
              </div>

              <h2 id="create-google-cloud-project">
                Créer un projet Google Cloud
              </h2>

              <h3 id="create-new-project">1. Créez un nouveau projet</h3>

              <p>
                Rendez-vous sur{" "}
                <a
                  href="https://console.cloud.google.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://console.cloud.google.com/
                </a>{" "}
                et choisissez <strong>&quot;new project&quot;</strong> (en haut
                à gauche).
              </p>

              <h3 id="enable-google-drive-api">
                2. Activez l&apos;API Google Drive
              </h3>

              <p>
                Allez dans l&apos;onglet{" "}
                <strong>&quot;API &amp; Services&quot;</strong>, puis dans{" "}
                <strong>&quot;API library&quot;</strong>. Choisissez{" "}
                <strong>Google Drive API</strong> et activez-la :
              </p>

              <Image
                src="/images/google-drive-storage/image-1.webp"
                alt="Enable Google Drive API"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-consent-screen">
                3. Configurez l&apos;écran de consentement
              </h3>

              <p>
                Allez dans <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;Configure consent screen&quot;</strong> et
                remplissez les données demandées :
              </p>

              <Image
                src="/images/google-drive-storage/image-2.webp"
                alt="Configure consent screen"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-oauth-client-id">4. Créez un ID client OAuth</h3>

              <p>
                Allez dans <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;OAuth client ID&quot;</strong> :
              </p>

              <Image
                src="/images/google-drive-storage/image-3.webp"
                alt="Create OAuth client ID"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-application-settings">
                5. Configurez les paramètres de l&apos;application
              </h3>

              <p>Remplissez les données suivantes :</p>

              <ul>
                <li>
                  <strong>Application type :</strong> Web application
                </li>
                <li>
                  <strong>Authorized JavaScript origins :</strong>{" "}
                  <code>https://databasus.yourdomain.com</code>
                </li>
                <li>
                  <strong>Authorized redirect URIs :</strong>{" "}
                  <code>
                    https://databasus.yourdomain.com/storages/google-oauth
                  </code>
                </li>
              </ul>

              <p>
                <strong>Remarque :</strong> remplacez{" "}
                <code>databasus.yourdomain.com</code> par le domaine réel où
                Databasus est hébergé (par exemple{" "}
                <code>backup.example.com</code>).
              </p>

              <p>Copiez ensuite les identifiants :</p>

              <Image
                src="/images/google-drive-storage/image-4.png"
                alt="Configure application settings - part 1"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <Image
                src="/images/google-drive-storage/image-5.png"
                alt="Configure application settings - part 2"
                width={450}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="add-scope">6. Ajoutez le scope</h3>

              <p>
                Allez dans <strong>&quot;Data Access&quot;</strong> et ajoutez
                le scope <code>&quot;/auth/drive.file&quot;</code> :
              </p>

              <Image
                src="/images/google-drive-storage/image-6.png"
                alt="Add scope"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="publish-app">7. Publiez l&apos;application</h3>

              <p>
                Allez dans <strong>&quot;Audience&quot;</strong> et publiez
                l&apos;application :
              </p>

              <Image
                src="/images/google-drive-storage/image-7.png"
                alt="Publish the app"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="sign-in-google-account">
                Se connecter via un compte Google
              </h2>

              <h3 id="fill-credentials">1. Renseignez les identifiants</h3>

              <p>
                Renseignez dans Databasus les identifiants obtenus aux étapes
                précédentes :
              </p>

              <Image
                src="/images/google-drive-storage/image-8.png"
                alt="Fill credentials data"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="choose-account">2. Choisissez votre compte</h3>

              <p>Choisissez votre compte Google pour vous connecter.</p>

              <h3 id="handle-security-warning">
                3. Gérez l&apos;avertissement de sécurité
              </h3>

              <p>
                Si un avertissement s&apos;affiche, cliquez sur{" "}
                <strong>&quot;Advanced&quot;</strong> (en bas à gauche) et
                choisissez <strong>&quot;Proceed anyway&quot;</strong>.
              </p>

              <p>
                <strong>Remarque :</strong> cet avertissement apparaît parce que
                votre application n&apos;est pas encore vérifiée par Google.
                Vous pouvez continuer sans risque pour votre propre application.
              </p>

              <p>
                C&apos;est terminé ! Votre Google Drive est maintenant connecté
                à Databasus et prêt à stocker vos sauvegardes PostgreSQL.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/fr/storages"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Retour aux stockages
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
