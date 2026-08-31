import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Comment configurer les notifications Slack pour Databasus | Databasus",
  description:
    "Guide pas à pas pour configurer les notifications Slack des alertes de sauvegarde PostgreSQL avec Databasus. Apprenez à créer une app bot Slack et à configurer les notifications.",
  keywords: [
    "Databasus",
    "notifications Slack",
    "sauvegarde PostgreSQL",
    "token bot Slack",
    "API Slack",
    "alertes de sauvegarde",
    "notifications de base de données",
  ],
  openGraph: {
    title:
      "Comment configurer les notifications Slack pour Databasus | Databasus",
    description:
      "Guide pas à pas pour configurer les notifications Slack des alertes de sauvegarde PostgreSQL avec Databasus. Apprenez à créer une app bot Slack et à configurer les notifications.",
    type: "article",
    url: getLocalizedUrl("fr", "notifiers/slack"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title:
      "Comment configurer les notifications Slack pour Databasus | Databasus",
    description:
      "Guide pas à pas pour configurer les notifications Slack des alertes de sauvegarde PostgreSQL avec Databasus. Apprenez à créer une app bot Slack et à configurer les notifications.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "notifiers/slack"),
    languages: getLanguageAlternates("notifiers/slack"),
  },
  robots: "index, follow",
};

export default function SlackPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Comment configurer les notifications Slack pour Databasus",
            description:
              "Guide pas à pas pour configurer les notifications Slack des alertes de sauvegarde PostgreSQL avec Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Accéder à l'API Slack",
                text: "Rendez-vous sur https://api.slack.com/apps et connectez-vous à votre espace de travail Slack.",
              },
              {
                "@type": "HowToStep",
                name: "Créer une nouvelle app",
                text: "Cliquez sur le bouton 'Create New App' et choisissez 'From scratch'.",
              },
              {
                "@type": "HowToStep",
                name: "Configurer les permissions du bot",
                text: "Allez dans OAuth & Permissions et ajoutez les scopes requis sous Bot Token Scopes : chat:write, channels:join, im:write et groups:write.",
              },
              {
                "@type": "HowToStep",
                name: "Installer dans l'espace de travail",
                text: "Installez l'app dans votre espace de travail et autorisez-la.",
              },
              {
                "@type": "HowToStep",
                name: "Copier le token du bot",
                text: "Copiez le Bot User OAuth Token qui commence par 'xoxb-'.",
              },
              {
                "@type": "HowToStep",
                name: "Récupérer l'ID du canal",
                text: "Ouvrez le canal cible et récupérez le Channel ID dans les détails du canal.",
              },
              {
                "@type": "HowToStep",
                name: "Ajouter le bot à un canal privé",
                text: "Si vous utilisez un canal privé, invitez le bot dans le canal en le mentionnant.",
              },
              {
                "@type": "HowToStep",
                name: "Configurer dans Databasus",
                text: "Dans Databasus, ajoutez le Bot Token et le Channel ID à la configuration du notificateur Slack.",
              },
              {
                "@type": "HowToStep",
                name: "Tester la notification",
                text: "Testez la notification pour vérifier qu'elle fonctionne correctement.",
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
              <h1 id="slack-notifications">Notifications Slack</h1>

              <p className="text-lg text-gray-400">
                Configurez Slack pour recevoir des notifications instantanées
                sur le statut de vos sauvegardes PostgreSQL. Les réussites, les
                échecs et les avertissements arrivent directement dans vos
                canaux Slack.
              </p>

              <h2 id="create-slack-app">Créer une app Slack</h2>

              <h3 id="go-to-slack-api">1. Accédez à l&apos;API Slack</h3>

              <p>
                Rendez-vous sur{" "}
                <a
                  href="https://api.slack.com/apps"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://api.slack.com/apps
                </a>{" "}
                et connectez-vous à votre espace de travail Slack.
              </p>

              <h3 id="create-new-app">2. Créez une nouvelle app</h3>

              <p>
                Cliquez sur le bouton{" "}
                <strong>&quot;Create New App&quot;</strong>.
              </p>

              <h3 id="choose-from-scratch">
                3. Choisissez &quot;From scratch&quot;
              </h3>

              <p>
                Sélectionnez l&apos;option{" "}
                <strong>&quot;From scratch&quot;</strong> lorsqu&apos;elle vous
                est proposée.
              </p>

              <h3 id="name-your-app">4. Nommez votre app</h3>

              <p>
                Saisissez un nom pour votre app (par exemple &quot;Databasus
                Notifications&quot;) et sélectionnez l&apos;espace de travail où
                vous voulez l&apos;installer. Cliquez sur{" "}
                <strong>&quot;Create App&quot;</strong>.
              </p>

              <h2 id="configure-bot-permissions">
                Configurer les permissions du bot
              </h2>

              <h3 id="navigate-to-oauth">
                5. Allez dans OAuth &amp; Permissions
              </h3>

              <p>
                Dans la barre latérale gauche, cliquez sur{" "}
                <strong>&quot;OAuth &amp; Permissions&quot;</strong>.
              </p>

              <img
                src="/images/notifier-slack/image-1.png"
                alt="Navigate to OAuth &amp; Permissions"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="add-bot-scopes">
                6. Ajoutez les Bot Token Scopes (obligatoire)
              </h3>

              <p>
                Faites défiler jusqu&apos;à la section{" "}
                <strong>&quot;Scopes&quot;</strong> et, sous{" "}
                <strong>&quot;Bot Token Scopes&quot;</strong>, cliquez sur{" "}
                <strong>&quot;Add an OAuth Scope&quot;</strong>.
              </p>

              <p>Ajoutez tous les scopes requis suivants :</p>

              <ul>
                <li>
                  <code>chat:write</code> - pour envoyer des messages dans les
                  canaux
                </li>
                <li>
                  <code>channels:join</code> - pour permettre au bot de
                  rejoindre automatiquement les canaux publics
                </li>
                <li>
                  <code>im:write</code> - pour envoyer des messages directs aux
                  utilisateurs
                </li>
                <li>
                  <code>groups:write</code> - pour envoyer des messages dans les
                  canaux privés
                </li>
                <li>
                  <code>channels:history</code> - pour lire l&apos;historique
                  des canaux
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-2.png"
                alt="Add Bot Token Scopes"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h2 id="install-app">
                Installer l&apos;app dans l&apos;espace de travail
              </h2>

              <h3 id="install-to-workspace">
                7. Installez dans l&apos;espace de travail
              </h3>

              <p>
                Remontez en haut de la page{" "}
                <strong>&quot;OAuth &amp; Permissions&quot;</strong> et cliquez
                sur <strong>&quot;Install to Workspace&quot;</strong>.
              </p>

              <img
                src="/images/notifier-slack/image-3.png"
                alt="Install to Workspace"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="authorize-app">8. Autorisez l&apos;app</h3>

              <p>
                Vérifiez les permissions et cliquez sur{" "}
                <strong>&quot;Allow&quot;</strong> pour autoriser l&apos;app.
              </p>

              <h3 id="copy-bot-token">9. Copiez le Bot User OAuth Token</h3>

              <p>
                Après l&apos;installation, vous verrez le{" "}
                <strong>&quot;Bot User OAuth Token&quot;</strong>. Il commence
                par <code>xoxb-</code>. Copiez ce token : vous en aurez besoin
                pour la configuration de Databasus.
              </p>

              <h2 id="get-channel-id">Récupérer l&apos;ID du canal</h2>

              <h3 id="open-channel">10. Ouvrez votre canal cible</h3>

              <p>
                Dans votre espace de travail Slack, ouvrez le canal où vous
                voulez recevoir les notifications de sauvegarde.
              </p>

              <h3 id="get-channel-info">11. Récupérez l&apos;ID du canal</h3>

              <p>
                Cliquez sur le nom du canal en haut, puis faites défiler les
                détails du canal. Vous trouverez le <strong>Channel ID</strong>{" "}
                en bas de la section &quot;About&quot;. Il commence par{" "}
                <code>C</code> (pour les canaux publics) ou <code>G</code> (pour
                les canaux privés).
              </p>

              <p>Copiez ce Channel ID.</p>

              <img
                src="/images/notifier-slack/image-4.png"
                alt="Get Channel ID"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[500px]"
                loading="lazy"
              />

              <h3 id="add-bot-to-channel">
                12. Ajoutez le bot au canal (obligatoire pour les canaux privés)
              </h3>

              <p>
                <strong>
                  Si vous utilisez un canal privé, vous devez inviter
                  manuellement le bot dans le canal :
                </strong>
              </p>

              <ol>
                <li>
                  Dans le canal privé, tapez{" "}
                  <code>@Databasus Notifications</code> (ou le nom que vous avez
                  donné à votre app)
                </li>
                <li>
                  Cliquez sur le nom du bot lorsqu&apos;il apparaît et
                  sélectionnez <strong>&quot;Add to Channel&quot;</strong> ou{" "}
                  <strong>&quot;Invite to Channel&quot;</strong>
                </li>
              </ol>

              <p>
                Pour les <strong>canaux publics</strong>, le bot les rejoint
                automatiquement lors de l&apos;envoi du premier message (grâce à
                la permission <code>channels:join</code>), cette étape
                n&apos;est donc pas nécessaire.
              </p>

              <h2 id="configure-databasus">Configurer dans Databasus</h2>

              <h3 id="add-slack-notifier">13. Ajoutez le notificateur Slack</h3>

              <p>
                Dans Databasus, allez dans les paramètres des notificateurs et
                ajoutez un nouveau notificateur Slack :
              </p>

              <ul>
                <li>
                  <strong>Bot Token :</strong> collez le Bot User OAuth Token
                  copié précédemment (il commence par <code>xoxb-</code>)
                </li>
                <li>
                  <strong>Target Channel ID :</strong> collez le Channel ID
                  copié précédemment (il commence par <code>C</code>,{" "}
                  <code>G</code>, <code>D</code> ou <code>U</code>)
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-5.png"
                alt="Add Slack notifier"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="test-notification">14. Testez la notification</h3>

              <p>
                Après avoir configuré le notificateur, testez-le pour vérifier
                qu&apos;il fonctionne correctement. Vous devriez recevoir un
                message de test dans le canal Slack sélectionné.
              </p>

              <p>
                C&apos;est terminé ! Votre espace de travail Slack est
                maintenant configuré pour recevoir les notifications de
                sauvegarde PostgreSQL de Databasus.
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
