import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title:
    "Comment configurer les notifications Microsoft Teams pour Databasus | Databasus",
  description:
    "Guide pas à pas pour configurer les notifications Microsoft Teams des alertes de sauvegarde PostgreSQL avec Databasus. Apprenez à créer un webhook Teams et à configurer les notifications.",
  keywords: [
    "Databasus",
    "notifications Microsoft Teams",
    "sauvegarde PostgreSQL",
    "webhook Teams",
    "alertes de sauvegarde",
    "notifications de base de données",
  ],
  openGraph: {
    title:
      "Comment configurer les notifications Microsoft Teams pour Databasus | Databasus",
    description:
      "Guide pas à pas pour configurer les notifications Microsoft Teams des alertes de sauvegarde PostgreSQL avec Databasus. Apprenez à créer un webhook Teams et à configurer les notifications.",
    type: "article",
    url: getLocalizedUrl("fr", "notifiers/teams"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title:
      "Comment configurer les notifications Microsoft Teams pour Databasus | Databasus",
    description:
      "Guide pas à pas pour configurer les notifications Microsoft Teams des alertes de sauvegarde PostgreSQL avec Databasus. Apprenez à créer un webhook Teams et à configurer les notifications.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "notifiers/teams"),
    languages: getLanguageAlternates("notifiers/teams"),
  },
  robots: "index, follow",
};

export default function TeamsPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Comment configurer les notifications Microsoft Teams pour Databasus",
            description:
              "Guide pas à pas pour configurer les notifications Microsoft Teams des alertes de sauvegarde PostgreSQL avec Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Ouvrir le canal Teams",
                text: "Accédez au canal Microsoft Teams où vous voulez recevoir les notifications.",
              },
              {
                "@type": "HowToStep",
                name: "Accéder aux workflows",
                text: "Ouvrez la fonctionnalité Workflows dans votre canal Teams.",
              },
              {
                "@type": "HowToStep",
                name: "Créer un nouveau workflow",
                text: "Créez un nouveau workflow pour les webhooks entrants.",
              },
              {
                "@type": "HowToStep",
                name: "Sélectionner le modèle de webhook",
                text: "Choisissez le modèle de webhook entrant parmi les options disponibles.",
              },
              {
                "@type": "HowToStep",
                name: "Configurer le webhook",
                text: "Définissez le nom du webhook et le canal.",
              },
              {
                "@type": "HowToStep",
                name: "Copier l'URL du webhook",
                text: "Copiez l'URL du webhook générée par Teams.",
              },
              {
                "@type": "HowToStep",
                name: "Configurer dans Databasus",
                text: "Collez l'URL du webhook dans la configuration du notificateur Databasus.",
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
              <h1 id="teams-notifications">Notifications Microsoft Teams</h1>

              <p className="text-lg text-gray-400">
                Configurez Microsoft Teams pour recevoir des notifications
                instantanées sur le statut de vos sauvegardes PostgreSQL. Les
                réussites, les échecs et les avertissements arrivent directement
                dans vos canaux Teams.
              </p>

              <h2 id="setup-teams-webhook">Configurer le webhook Teams</h2>

              <h3 id="open-teams-channel">1. Ouvrez votre canal Teams</h3>

              <p>
                Accédez au canal Microsoft Teams où vous voulez recevoir les
                notifications de sauvegarde. Cliquez sur les trois points (
                <strong>•••</strong>) à côté du nom du canal.
              </p>

              <Image
                src="/images/notifier-teams/image-01.png"
                alt="Open Teams channel"
                width={800}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="access-workflows">2. Accédez aux workflows</h3>

              <p>
                Dans le menu du canal, sélectionnez{" "}
                <strong>&quot;Workflows&quot;</strong> pour ouvrir
                l&apos;intégration Power Automate.
              </p>

              <Image
                src="/images/notifier-teams/image-02.png"
                alt="Access Workflows"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-new-workflow">3. Créez un nouveau workflow</h3>

              <p>
                Dans le panneau Workflows, cliquez sur{" "}
                <strong>&quot;Create&quot;</strong> ou recherchez le modèle{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>
                .
              </p>

              <Image
                src="/images/notifier-teams/image-03.png"
                alt="Create new workflow"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="select-webhook-template">
                4. Sélectionnez le modèle de webhook
              </h3>

              <p>
                Choisissez le modèle{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>{" "}
                parmi les options disponibles.
              </p>

              <Image
                src="/images/notifier-teams/image-04.png"
                alt="Select webhook template"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-webhook">5. Configurez le webhook</h3>

              <p>
                Configurez le webhook en lui donnant un nom (par exemple{" "}
                <strong>&quot;Databasus Backup Notifications&quot;</strong>) et
                confirmez le canal où les notifications seront publiées.
              </p>

              <Image
                src="/images/notifier-teams/image-05.png"
                alt="Configure webhook"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="copy-webhook-url">6. Copiez l&apos;URL du webhook</h3>

              <p>
                Après avoir créé le workflow, vous verrez l&apos;
                <strong>HTTP POST URL</strong>. Copiez cette URL : vous en aurez
                besoin pour la configuration de Databasus.
              </p>

              <Image
                src="/images/notifier-teams/image-06.png"
                alt="Copy webhook URL"
                width={500}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="configure-databasus">Configurer dans Databasus</h2>

              <h3 id="add-teams-notifier">1. Ajoutez le notificateur Teams</h3>

              <p>
                Dans Databasus, allez dans les paramètres des notificateurs et
                ajoutez un nouveau notificateur Microsoft Teams. Collez
                l&apos;URL du webhook copiée depuis Teams.
              </p>

              <Image
                src="/images/notifier-teams/image-07.png"
                alt="Configure Teams in Databasus"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="test-notification">2. Testez la notification</h3>

              <p>
                Après avoir configuré le webhook, testez la notification pour
                vérifier qu&apos;elle fonctionne correctement. Vous devriez
                recevoir un message de test dans le canal Teams sélectionné.
              </p>

              <p>
                C&apos;est terminé ! Votre canal Microsoft Teams est maintenant
                configuré pour recevoir les notifications de sauvegarde
                PostgreSQL de Databasus.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/fr/notifiers"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Retour aux notificateurs
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
