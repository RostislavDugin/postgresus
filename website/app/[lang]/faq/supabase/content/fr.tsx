import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Comment sauvegarder Supabase avec Databasus | Databasus",
  description:
    "Apprenez à sauvegarder votre base de données PostgreSQL Supabase avec Databasus. Guide pas à pas pour configurer le session pooler ou une adresse IPv4 pour les sauvegardes Supabase.",
  keywords: [
    "Databasus",
    "sauvegarde Supabase",
    "sauvegarde PostgreSQL Supabase",
    "sauvegarder base Supabase",
    "session pooler Supabase",
    "IPv4 Supabase",
    "sauvegarde PostgreSQL",
    "sauvegarde de base de données",
  ],
  openGraph: {
    title: "Comment sauvegarder Supabase avec Databasus | Databasus",
    description:
      "Apprenez à sauvegarder votre base de données PostgreSQL Supabase avec Databasus. Guide pas à pas pour configurer le session pooler ou une adresse IPv4 pour les sauvegardes Supabase.",
    type: "article",
    url: getLocalizedUrl("fr", "faq/supabase"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Comment sauvegarder Supabase avec Databasus | Databasus",
    description:
      "Apprenez à sauvegarder votre base de données PostgreSQL Supabase avec Databasus. Guide pas à pas pour configurer le session pooler ou une adresse IPv4 pour les sauvegardes Supabase.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "faq/supabase"),
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
            name: "Comment sauvegarder Supabase avec Databasus",
            description:
              "Guide pas à pas pour sauvegarder votre base de données PostgreSQL Supabase avec Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Récupérer les informations de connexion depuis Supabase",
                text: "Accédez aux paramètres de votre projet Supabase et trouvez les informations de connexion à la base de données.",
              },
              {
                "@type": "HowToStep",
                name: "Utiliser le Session Pooler avec IPv4",
                text: "Copiez la chaîne de connexion du Session Pooler et vérifiez que 'Use IPv4 Address' est activé.",
              },
              {
                "@type": "HowToStep",
                name: "Configurer Databasus",
                text: "Saisissez les informations de connexion Supabase dans Databasus pour commencer à sauvegarder votre base.",
              },
              {
                "@type": "HowToStep",
                name: "Comprendre les limitations de schéma",
                text: "Par défaut, seul le schéma public est sauvegardé, les autres schémas Supabase étant restreints.",
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
              <h1 id="supabase-backup">Comment sauvegarder Supabase</h1>

              <p className="text-lg text-gray-400">
                Databasus prend en charge les sauvegardes des bases de données
                PostgreSQL Supabase. Il faut simplement utiliser une adresse
                IPv4 pour se connecter à votre instance Supabase.
              </p>

              <h2 id="connection-options">Options de connexion</h2>

              <p>
                Il existe deux façons de connecter Databasus à votre base de
                données Supabase :
              </p>

              <ol>
                <li>
                  <strong>Utiliser le Session Pooler avec IPv4</strong>{" "}
                  (recommandé) : option gratuite disponible dans tous les
                  projets Supabase
                </li>
                <li>
                  <strong>Acheter l&apos;add-on IPv4</strong> : option de
                  connexion directe proposée par Supabase
                </li>
              </ol>

              <h2 id="session-pooler">
                Option 1 : utiliser le Session Pooler (recommandé)
              </h2>

              <p>
                Le Session Pooler fournit une adresse IPv4 pour la connexion à
                votre base Supabase sans coût supplémentaire. Voici comment le
                configurer :
              </p>

              <h3 id="step-1">1. Trouvez la connexion du pooler</h3>

              <p>
                Accédez à votre projet Supabase, allez dans{" "}
                <strong>Project Settings</strong> → <strong>Database</strong>.
                Faites défiler jusqu&apos;à la section{" "}
                <strong>Connection string</strong> et sélectionnez le mode{" "}
                <strong>Session pooler</strong>.
              </p>

              <img
                src="/images/faq/supabase/image-1.png"
                alt="Select Session pooler mode in Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h3 id="step-2">2. Copiez les informations de connexion</h3>

              <p>
                Copiez les informations de connexion et utilisez-les dans
                Databasus lors de l&apos;ajout de votre base de données. La
                capture d&apos;écran ci-dessous montre où trouver chaque
                information.
              </p>

              <img
                src="/images/faq/supabase/image-2.png"
                alt="Enable IPv4 Address toggle in Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h2 id="ipv4-addon">Option 2 : acheter l&apos;add-on IPv4</h2>

              <p>
                Supabase propose un add-on IPv4 payant qui fournit une adresse
                IPv4 dédiée pour votre base de données. Cette option vous donne
                une connexion directe sans passer par le pooler de connexions.
              </p>

              <p>Pour activer cette option :</p>

              <ol>
                <li>Allez dans le tableau de bord de votre projet Supabase</li>
                <li>
                  Accédez à <strong>Project Settings</strong> →{" "}
                  <strong>Add-ons</strong>
                </li>
                <li>
                  Activez l&apos;add-on <strong>IPv4</strong>
                </li>
                <li>
                  Utilisez les informations de connexion directe à la base dans
                  Databasus
                </li>
              </ol>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Astuce :</strong> dans
                  la plupart des cas, le Session Pooler gratuit avec IPv4 suffit
                  parfaitement pour les sauvegardes. L&apos;add-on IPv4 payant
                  n&apos;est nécessaire que si vous avez besoin d&apos;une
                  connexion directe pour d&apos;autres raisons.
                </p>
              </div>

              <h2 id="default-schema">Limitation du schéma par défaut</h2>

              <p>
                Par défaut, Databasus ne sauvegarde que le schéma{" "}
                <code>public</code> lorsqu&apos;il travaille avec des bases
                Supabase. En effet, Supabase restreint l&apos;accès aux autres
                schémas (comme <code>auth</code>, <code>storage</code> et{" "}
                <code>realtime</code>) pour des raisons de sécurité.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-blue-400">ℹ️ Remarque :</strong> le
                  schéma <code>public</code> contient les données de votre
                  application et vos tables personnalisées. Les schémas comme{" "}
                  <code>auth</code> et <code>storage</code> sont protégés et
                  gérés par Supabase lui-même.
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
