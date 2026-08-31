import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "Comment utiliser Databasus avec Cloudflare R2 | Databasus",
  description:
    "Guide pas à pas pour configurer le stockage Cloudflare R2 pour les sauvegardes PostgreSQL avec Databasus. Apprenez à configurer un stockage compatible S3 avec R2.",
  keywords: [
    "Databasus",
    "Cloudflare R2",
    "sauvegarde PostgreSQL",
    "stockage S3",
    "stockage cloud",
    "sauvegarde de base de données",
  ],
  openGraph: {
    title: "Comment utiliser Databasus avec Cloudflare R2 | Databasus",
    description:
      "Guide pas à pas pour configurer le stockage Cloudflare R2 pour les sauvegardes PostgreSQL avec Databasus. Apprenez à configurer un stockage compatible S3 avec R2.",
    type: "article",
    url: getLocalizedUrl("fr", "storages/cloudflare-r2"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Comment utiliser Databasus avec Cloudflare R2 | Databasus",
    description:
      "Guide pas à pas pour configurer le stockage Cloudflare R2 pour les sauvegardes PostgreSQL avec Databasus. Apprenez à configurer un stockage compatible S3 avec R2.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "storages/cloudflare-r2"),
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
            name: "Comment utiliser Databasus avec Cloudflare R2",
            description:
              "Guide pas à pas pour configurer le stockage Cloudflare R2 pour les sauvegardes PostgreSQL avec Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Renseigner le nom du bucket",
                text: "Saisissez le nom de votre bucket R2 dans la configuration du stockage.",
              },
              {
                "@type": "HowToStep",
                name: "Définir la région",
                text: 'Dans le champ région, saisissez "auto".',
              },
              {
                "@type": "HowToStep",
                name: "Générer un access key ID et un secret access key",
                text: "Dans le tableau de bord Cloudflare, allez dans R2 → API → Manage API Tokens. Créez le token et accordez-lui les permissions nécessaires.",
              },
              {
                "@type": "HowToStep",
                name: "Trouver votre account ID",
                text: "Sur n'importe quelle page R2 du tableau de bord, votre Account ID est affiché en haut.",
              },
              {
                "@type": "HowToStep",
                name: "Construire l'endpoint S3",
                text: "Remplacez <ACCOUNT_ID> par la valeur de votre tableau de bord au format : https://<ACCOUNT_ID>.r2.cloudflarestorage.com",
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
              <h1 id="cloudflare-r2">Stockage Cloudflare R2</h1>

              <p className="text-lg text-gray-400">
                Pour utiliser Cloudflare R2 comme stockage compatible S3 pour
                vos sauvegardes PostgreSQL, vous devez configurer les
                identifiants de votre bucket R2 et l&apos;endpoint.
              </p>

              <h2 id="configuration-steps">Étapes de configuration</h2>

              <h3 id="fill-bucket-name">1. Renseignez le nom du bucket</h3>

              <p>
                Saisissez le nom de votre bucket R2 dans la configuration du
                stockage :
              </p>

              <Image
                src="/images/cloudflare-r2-storage/image-1.webp"
                alt="Fill your bucket name in Cloudflare R2"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="set-region">2. Définissez la région</h3>

              <p>
                Dans le champ région, saisissez <code>&quot;auto&quot;</code>.
              </p>

              <h3 id="generate-access-key">
                3. Générez un Access Key ID &amp; Secret Access Key
              </h3>

              <p>
                Dans le tableau de bord Cloudflare, allez dans{" "}
                <strong>R2 → API → Manage API Tokens</strong>. Créez un nouveau
                token et accordez-lui les permissions nécessaires (par exemple{" "}
                <strong>&quot;Object Read &amp; Write&quot;</strong>).
              </p>

              <p>Une fois le token créé, vous verrez :</p>

              <ul>
                <li>
                  <strong>Access Key ID</strong> (l&apos;ID du token)
                </li>
                <li>
                  <strong>Secret Access Key</strong> (le hash SHA-256 de la
                  valeur du token)
                </li>
              </ul>

              <p>Copiez ces deux valeurs dans Databasus :</p>

              <Image
                src="/images/cloudflare-r2-storage/image-2.gif"
                alt="Generate Access Key ID and Secret Access Key"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="find-account-id">4. Trouvez votre account ID</h3>

              <p>
                Sur n&apos;importe quelle page R2 du tableau de bord, votre
                Account ID est affiché en haut (ou dans les paramètres de votre
                compte) :
              </p>

              <Image
                src="/images/cloudflare-r2-storage/image-3.webp"
                alt="Find your Account ID in Cloudflare dashboard"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="construct-endpoint">5. Construisez l&apos;endpoint S3</h3>

              <p>Utilisez le format suivant pour votre endpoint S3 :</p>

              <pre>
                <code>https://&lt;ACCOUNT_ID&gt;.r2.cloudflarestorage.com</code>
              </pre>

              <p>
                Remplacez <code>&lt;ACCOUNT_ID&gt;</code> par la valeur de votre
                tableau de bord et saisissez-la dans Databasus.
              </p>

              <p>
                C&apos;est terminé ! Votre configuration devrait maintenant
                ressembler à ceci :
              </p>

              <Image
                src="/images/cloudflare-r2-storage/image-4.png"
                alt="Configuration complete"
                width={500}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                Votre Databasus est maintenant prêt à utiliser Cloudflare R2
                comme stockage pour vos sauvegardes PostgreSQL.
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
