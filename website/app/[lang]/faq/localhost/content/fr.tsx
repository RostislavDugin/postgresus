import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "Comment sauvegarder des bases de données localhost | Databasus",
  description:
    "Apprenez à sauvegarder des bases de données PostgreSQL sur localhost avec Databasus. Configurez le mode réseau host de Docker pour la sauvegarde de bases locales.",
  keywords: [
    "Databasus",
    "sauvegarde localhost",
    "sauvegarde PostgreSQL locale",
    "sauvegarder base de données locale",
    "réseau host Docker",
    "sauvegarde PostgreSQL",
    "sauvegarde de base de données",
    "base de données localhost",
  ],
  openGraph: {
    title: "Comment sauvegarder des bases de données localhost | Databasus",
    description:
      "Apprenez à sauvegarder des bases de données PostgreSQL sur localhost avec Databasus. Configurez le mode réseau host de Docker pour la sauvegarde de bases locales.",
    type: "article",
    url: getLocalizedUrl("fr", "faq/localhost"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Comment sauvegarder des bases de données localhost | Databasus",
    description:
      "Apprenez à sauvegarder des bases de données PostgreSQL sur localhost avec Databasus. Configurez le mode réseau host de Docker pour la sauvegarde de bases locales.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "faq/localhost"),
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
            name: "Comment sauvegarder des bases de données localhost avec Databasus",
            description:
              "Guide pas à pas pour sauvegarder des bases de données PostgreSQL sur localhost avec Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Configurer le mode réseau host de Docker",
                text: "Mettez à jour votre configuration Docker pour utiliser le mode réseau host afin que le conteneur puisse accéder aux services sur localhost.",
              },
              {
                "@type": "HowToStep",
                name: "Utiliser Docker Compose ou Docker run",
                text: "Appliquez le paramètre network_mode: host dans Docker Compose ou utilisez l'option --network host avec Docker run.",
              },
              {
                "@type": "HowToStep",
                name: "Se connecter à la base localhost",
                text: "Utilisez 127.0.0.1 ou localhost comme hôte de base de données dans votre configuration de sauvegarde Databasus.",
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
              <h1 id="localhost-backup">
                Comment sauvegarder des bases de données localhost
              </h1>

              <p className="text-lg text-gray-400">
                Apprenez à configurer Databasus pour sauvegarder des bases de
                données PostgreSQL fonctionnant sur votre machine hôte
                (localhost) lorsque vous utilisez Docker.
              </p>

              <h2 id="the-problem">Le problème</h2>

              <p>
                Si vous exécutez Databasus dans Docker et voulez sauvegarder des
                bases de données fonctionnant sur votre machine hôte
                (localhost), vous devez configurer Docker pour utiliser le{" "}
                <strong>mode réseau host</strong>.
              </p>

              <p>
                Par défaut, les conteneurs Docker fonctionnent dans un réseau
                isolé et ne peuvent pas accéder aux services sur{" "}
                <code>localhost</code>. Le mode réseau host permet au conteneur
                de partager la pile réseau de l&apos;hôte.
              </p>

              <h2 id="docker-compose-solution">Solution pour Docker Compose</h2>

              <p>
                Mettez à jour votre fichier <code>docker-compose.yml</code> pour
                utiliser <code>network_mode: host</code> :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={dockerComposeHost} />
                </div>
              </div>

              <h2 id="docker-run-solution">Solution pour Docker run</h2>

              <p>
                Utilisez l&apos;option <code>--network host</code> :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRunHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={dockerRunHost} />
                </div>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Remarque :</strong> en
                  mode réseau host, vous pouvez vous connecter à votre base
                  localhost en utilisant{" "}
                  <code className="bg-[#374151] text-gray-200">127.0.0.1</code>{" "}
                  ou{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>{" "}
                  comme hôte dans votre configuration de sauvegarde Databasus.
                  Vous accéderez aussi à l&apos;interface Databasus directement
                  sur{" "}
                  <code className="bg-[#374151] text-gray-200">
                    http://localhost:4005
                  </code>{" "}
                  sans mappage de port.
                </p>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">
                    ⚠️ Important pour les utilisateurs Windows et macOS :
                  </strong>{" "}
                  le mode réseau{" "}
                  <code className="bg-[#374151] text-red-400">host</code> ne
                  fonctionne nativement que sous Linux. Sous Windows et macOS,
                  Docker s&apos;exécute dans une VM Linux, il faut donc utiliser{" "}
                  <code className="bg-[#374151] text-gray-200">
                    host.docker.internal
                  </code>{" "}
                  à la place de{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>{" "}
                  comme adresse de l&apos;hôte de base de données dans votre
                  configuration de sauvegarde.
                </p>
              </div>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/fr/faq"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Retour à la FAQ
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
