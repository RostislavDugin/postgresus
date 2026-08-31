import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Configuration avancée - Documentation Databasus",
  description:
    "Variables d'environnement optionnelles pour l'auto-hébergement de Databasus : connexion Google et GitHub, e-mail SMTP, captcha Cloudflare Turnstile, télémétrie, export des journaux OpenTelemetry et script d'analytique personnalisé. Inutiles pour une installation par défaut.",
  keywords: [
    "variables d'environnement Databasus",
    "configuration avancée Databasus",
    "configuration auto-hébergée",
    "OAuth GitHub",
    "OAuth Google",
    "configuration e-mail SMTP",
    "Cloudflare Turnstile",
    "variables d'environnement Docker",
    "journaux OpenTelemetry",
  ],
  openGraph: {
    title: "Configuration avancée - Documentation Databasus",
    description:
      "Variables d'environnement optionnelles pour l'auto-hébergement de Databasus : connexion Google et GitHub, e-mail SMTP, captcha Cloudflare Turnstile, télémétrie, export des journaux OpenTelemetry et script d'analytique personnalisé. Inutiles pour une installation par défaut.",
    type: "article",
    url: getLocalizedUrl("fr", "advanced-config"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Configuration avancée - Documentation Databasus",
    description:
      "Variables d'environnement optionnelles pour l'auto-hébergement de Databasus : connexion Google et GitHub, e-mail SMTP, captcha Cloudflare Turnstile, télémétrie, export des journaux OpenTelemetry et script d'analytique personnalisé. Inutiles pour une installation par défaut.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "advanced-config"),
    languages: getLanguageAlternates("advanced-config"),
  },
  robots: "index, follow",
};

export default function AdvancedConfigPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Configuration avancée - Documentation Databasus",
            description:
              "Variables d'environnement optionnelles pour l'auto-hébergement de Databasus : connexion Google et GitHub, e-mail SMTP, captcha Cloudflare Turnstile, télémétrie, export des journaux OpenTelemetry et script d'analytique personnalisé. Inutiles pour une installation par défaut.",
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

      <DocsNavbarComponent lang="fr" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="fr" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="advanced-config">Configuration avancée</h1>

              <p className="text-lg text-gray-400">
                Databasus fonctionne d&apos;emblée avec des valeurs par défaut
                raisonnables : une installation standard en un seul conteneur ne
                demande aucune configuration. Chaque variable de cette page est{" "}
                <strong>optionnelle</strong> et inutile dans 99 % des
                déploiements en production.
              </p>

              <h2 id="oauth">OAuth</h2>

              <p>
                Par défaut, Databasus utilise la connexion par e-mail et mot de
                passe. Vous pouvez en plus permettre la connexion avec un compte
                Google ou GitHub. Le bouton d&apos;un fournisseur apparaît dès
                que son client ID est défini, mais la connexion n&apos;aboutit
                que lorsque <strong>les deux</strong> valeurs, client ID et
                client secret, sont présentes.
              </p>

              <p>
                Quand vous enregistrez l&apos;application OAuth, définissez son
                URL de redirection (callback) sur{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>. À cause
                de cette redirection, la connexion OAuth exige que votre
                instance soit servie en HTTPS sur un domaine public (voir la
                note ci-dessous).
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>
                      HTTPS est requis pour la connexion et l&apos;e-mail.
                    </strong>{" "}
                    La connexion OAuth et l&apos;e-mail exigent tous deux que
                    votre instance soit joignable en HTTPS sur un domaine public
                    : les fournisseurs OAuth redirigent le navigateur vers{" "}
                    <code>https://&lt;your-domain&gt;/auth/callback</code>, et
                    les liens contenus dans les e-mails doivent s&apos;ouvrir
                    chez leurs destinataires. Une instance limitée à localhost
                    ou en HTTP simple ne peut pas utiliser ces fonctionnalités.
                    Le moyen le plus simple d&apos;obtenir HTTPS est la
                    configuration{" "}
                    <a
                      href="/fr/installation/#caddy-reverse-proxy"
                      className="text-blue-400 hover:text-blue-300"
                    >
                      reverse proxy Caddy
                    </a>
                    .
                  </p>
                </div>
              </div>

              <h3 id="oauth-google">Google</h3>

              <p>
                Créez un client OAuth dans la{" "}
                <a
                  href="https://console.cloud.google.com/apis/credentials"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Google Cloud Console
                </a>{" "}
                (APIs &amp; Services → Credentials → Create credentials → OAuth
                client ID, type d&apos;application <em>Web application</em>) et
                ajoutez <code>https://&lt;your-domain&gt;/auth/callback</code>{" "}
                comme URI de redirection autorisée.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID de votre client OAuth Google. Le définir affiche
                      le bouton &quot;Sign in with Google&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret de votre client OAuth Google. Requis avec le
                      client ID pour que la connexion fonctionne.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h3 id="oauth-github">GitHub</h3>

              <p>
                Créez une application OAuth dans les{" "}
                <a
                  href="https://github.com/settings/developers"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  paramètres développeur GitHub
                </a>{" "}
                (Settings → Developer settings → OAuth Apps → New OAuth App) et
                définissez l&apos;URL de callback d&apos;autorisation sur{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID de votre application OAuth GitHub. Le définir
                      affiche le bouton &quot;Sign in with GitHub&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret de votre application OAuth GitHub. Requis
                      avec le client ID pour que la connexion fonctionne.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="email-smtp">E-mail (SMTP)</h2>

              <p>
                Connectez un serveur SMTP pour que Databasus puisse envoyer des
                e-mails transactionnels, comme les liens de réinitialisation de
                mot de passe et les invitations aux espaces de travail.
                L&apos;e-mail est considéré comme configuré{" "}
                <strong>
                  uniquement quand <code>SMTP_HOST</code> et{" "}
                  <code>DATABASUS_URL</code> sont tous deux définis
                </strong>{" "}
                ; jusque-là, les fonctionnalités e-mail restent masquées dans
                l&apos;interface.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>SMTP_HOST</code>
                    </td>
                    <td data-label="Description">
                      Nom d&apos;hôte du serveur SMTP (par ex.{" "}
                      <code>smtp.gmail.com</code>). Active l&apos;e-mail avec{" "}
                      <code>DATABASUS_URL</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PORT</code>
                    </td>
                    <td data-label="Description">
                      Port du serveur SMTP (par ex. <code>587</code>). Doit être
                      un entier positif quand <code>SMTP_HOST</code> est défini.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_USER</code>
                    </td>
                    <td data-label="Description">
                      Nom d&apos;utilisateur pour l&apos;authentification SMTP.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PASSWORD</code>
                    </td>
                    <td data-label="Description">
                      Mot de passe pour l&apos;authentification SMTP. Pour
                      Gmail, utilisez un App Password, pas le mot de passe de
                      votre compte.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_FROM</code>
                    </td>
                    <td data-label="Description">
                      L&apos;adresse &quot;From&quot; des e-mails sortants.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_INSECURE_SKIP_VERIFY</code>
                    </td>
                    <td data-label="Description">
                      Mettez <code>true</code> pour ignorer la vérification du
                      certificat TLS lors de la connexion au serveur SMTP. Vaut{" "}
                      <code>false</code> par défaut. À réserver aux serveurs
                      avec un certificat auto-signé sur un réseau de confiance :
                      cette option désactive la protection contre les attaques
                      de l&apos;homme du milieu.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>DATABASUS_URL</code>
                    </td>
                    <td data-label="Description">
                      URL de base publique de votre instance (par ex.{" "}
                      <code>https://backup.example.com</code>). Sert à
                      construire les liens dans les e-mails. Requis avec{" "}
                      <code>SMTP_HOST</code>.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="signup-captcha">
                Captcha à l&apos;inscription (Cloudflare Turnstile)
              </h2>

              <p>
                Si votre instance est joignable depuis l&apos;internet public,
                vous pouvez placer un défi{" "}
                <a
                  href="https://www.cloudflare.com/products/turnstile/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Cloudflare Turnstile
                </a>{" "}
                sur les formulaires d&apos;inscription et de connexion pour
                tenir les bots à l&apos;écart. Les deux clés viennent du tableau
                de bord Turnstile, et le défi ne s&apos;active que lorsque les
                deux sont définies.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    Pour bloquer complètement les inscriptions externes au lieu
                    de simplement les filtrer, vous n&apos;avez pas besoin de
                    captcha : ouvrez{" "}
                    <strong>Databasus settings → Allow sign up</strong> dans
                    l&apos;interface et désactivez l&apos;option. Cela ferme
                    entièrement le formulaire d&apos;inscription.
                  </p>
                </div>
              </div>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SITE_KEY</code>
                    </td>
                    <td data-label="Description">
                      Clé de site Turnstile publique, utilisée pour afficher le
                      widget dans le navigateur.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SECRET_KEY</code>
                    </td>
                    <td data-label="Description">
                      Clé Turnstile secrète, utilisée par le backend pour
                      valider les réponses au défi.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="telemetry">Télémétrie</h2>

              <p>
                Databasus envoie par défaut une télémétrie d&apos;usage anonyme
                et non identifiante. Elle ne contient aucune donnée personnelle
                et nous aide à comprendre comment le projet est utilisé. Vous
                pouvez lire exactement ce qui est collecté dans la{" "}
                <a
                  href="/privacy"
                  className="text-blue-400 hover:text-blue-300"
                >
                  politique de confidentialité
                </a>
                , et vous pouvez la désactiver complètement.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Défaut</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>IS_DISABLE_ANONYMOUS_TELEMETRY</code>
                    </td>
                    <td data-label="Défaut">
                      <code>false</code>
                    </td>
                    <td data-label="Description">
                      Mettez <code>true</code> pour désactiver la télémétrie
                      d&apos;usage anonyme.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="logging">Journalisation</h2>

              <p>
                Databasus écrit ses journaux sur stdout et les duplique en JSON
                dans <code>databasus.log</code> sur le volume de données.
                Définissez <code>OPEN_TELEMETRY_URL</code> et il les exporte
                aussi via Open Telemetry vers un backend comme VictoriaLogs,
                Graylog, SigNoz, Grafana Loki, Datadog ou Honeycomb, ou vers un
                OpenTelemetry Collector, qui est lui-même un récepteur OTLP.
              </p>

              <ul>
                <li>
                  <strong>Le transport</strong> suit le schéma de l&apos;URL.{" "}
                  <code>http://</code> et <code>https://</code> envoient en
                  OTLP/HTTP et utilisent l&apos;URL telle quelle, chemin compris
                  ; <code>grpc://</code> et <code>grpcs://</code> envoient en
                  OTLP/gRPC et n&apos;utilisent que l&apos;hôte et le port.
                </li>
                <li>
                  <strong>L&apos;authentification</strong> se met dans{" "}
                  <code>OPEN_TELEMETRY_HEADERS</code> ou dans l&apos;URL sous la
                  forme <code>user:password@host</code>.
                </li>
                <li>
                  <strong>Les secrets</strong> (mots de passe, jetons,
                  identifiants) dans les URL sont caviardés avant qu&apos;un
                  enregistrement ne quitte le processus.
                </li>
                <li>
                  <strong>Les entrées d&apos;audit</strong> partent avec les
                  journaux applicatifs, marquées <code>log_type=audit</code>, et
                  ignorent <code>LOG_LEVEL</code> : élever le niveau ne fait
                  donc jamais disparaître la piste d&apos;audit.
                </li>
              </ul>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Défaut</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_URL</code>
                    </td>
                    <td data-label="Défaut">—</td>
                    <td data-label="Description">
                      URL complète du point de terminaison OTLP, chemin compris.
                      Laissez vide pour garder les journaux dans le conteneur.
                      Une chaîne de requête, un hôte manquant ou un schéma
                      inconnu arrête le conteneur au démarrage plutôt que
                      d&apos;exporter dans le vide.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </td>
                    <td data-label="Défaut">—</td>
                    <td data-label="Description">
                      Paires <code>key=value</code> séparées par des virgules,
                      envoyées avec chaque export, en général une clé API. Les
                      valeurs sont décodées en pourcent, conformément au format
                      standard <code>OTEL_EXPORTER_OTLP_HEADERS</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_LEVEL</code>
                    </td>
                    <td data-label="Défaut">
                      <code>info</code>
                    </td>
                    <td data-label="Description">
                      L&apos;une des valeurs <code>debug</code>,{" "}
                      <code>info</code>, <code>warn</code> ou <code>error</code>
                      . Une valeur non reconnue retombe sur <code>info</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_FILE_IS_ENABLED</code>
                    </td>
                    <td data-label="Défaut">
                      <code>true</code>
                    </td>
                    <td data-label="Description">
                      Écrit <code>databasus.log</code> à côté du reste des
                      données, avec rotation à 5 Mo et conservation de 3 anciens
                      fichiers. Mettez <code>false</code> si votre plateforme
                      collecte déjà stdout.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Valeurs pour les backends courants, chacune avec l&apos;en-tête
                qui l&apos;authentifie. Remplacez les hôtes, régions et clés par
                les vôtres :
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Backend</th>
                    <th>
                      <code>OPEN_TELEMETRY_URL</code>
                    </th>
                    <th>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>VictoriaLogs</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>
                        http://victoria-logs:9428/insert/opentelemetry/v1/logs
                      </code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20dXNlcjpwYXNzd29yZA==</code> :
                      les identifiants attendus par votre <code>vmauth</code> ou
                      votre reverse proxy, puisque VictoriaLogs lui-même
                      n&apos;a pas d&apos;authentification sur le chemin
                      d&apos;ingestion.
                    </td>
                  </tr>
                  <tr>
                    <td>OpenTelemetry Collector</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://otel-collector:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> :
                      correspond à une extension <code>bearertokenauth</code> ou{" "}
                      <code>basicauth</code> sur le récepteur. Un Collector
                      joignable uniquement dans votre réseau n&apos;en a en
                      général pas besoin.
                    </td>
                  </tr>
                  <tr>
                    <td>Graylog 6.2+</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://graylog:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> : le jeton
                      défini sur l&apos;input OpenTelemetry (gRPC). L&apos;input
                      accepte aussi le mTLS à la place.
                    </td>
                  </tr>
                  <tr>
                    <td>SigNoz Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpcs://ingest.eu.signoz.cloud:443</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>signoz-ingestion-key=your-ingestion-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Grafana Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>
                        https://otlp-gateway-prod-eu-west-0.grafana.net/otlp/v1/logs
                      </code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20&lt;base64&gt;</code> : base64
                      de <code>instance-id:api-token</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Honeycomb</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>https://api.honeycomb.io/v1/logs</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>x-honeycomb-team=your-api-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Datadog Agent</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://datadog-agent:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      Aucun : l&apos;Agent détient la clé API et transmet pour
                      vous.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Les valeurs d&apos;en-tête sont décodées en pourcent, donc
                l&apos;espace après <code>Basic</code> ou <code>Bearer</code>{" "}
                s&apos;écrit <code>%20</code> et une virgule dans une valeur{" "}
                <code>%2C</code>. L&apos;authentification Basic peut aussi aller
                directement dans l&apos;URL sous la forme{" "}
                <code>https://user:password@host/path</code> : Databasus la
                transforme en le même en-tête et la garde hors des journaux. Sur{" "}
                <code>http://</code> et <code>grpc://</code>, les clés et mots
                de passe circulent en clair, utilisez donc <code>https://</code>{" "}
                ou <code>grpcs://</code> en dehors d&apos;un réseau de
                confiance.
              </p>

              <h2 id="analytics-script">Script d&apos;analytique</h2>

              <p>
                Databasus peut injecter votre propre extrait d&apos;analytique
                ou de suivi (Google Analytics, Plausible, Umami et similaires)
                dans l&apos;application. Quand <code>ANALYTICS_SCRIPT</code> est
                défini, sa valeur est insérée dans le <code>&lt;head&gt;</code>{" "}
                de la page au démarrage.
              </p>

              <p>
                <strong>Avertissement de sécurité :</strong> la valeur est
                injectée telle quelle, comme HTML et JavaScript bruts, et
                s&apos;exécute avec un accès complet à l&apos;interface
                Databasus dans le navigateur de chaque visiteur. Ne la
                définissez que sur un extrait que vous contrôlez et auquel vous
                faites entièrement confiance.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>ANALYTICS_SCRIPT</code>
                    </td>
                    <td data-label="Description">
                      Balisage <code>&lt;script&gt;</code> personnalisé injecté
                      avant la balise fermante <code>&lt;/head&gt;</code>.
                      Laissez vide pour n&apos;ajouter aucune analytique.
                    </td>
                  </tr>
                </tbody>
              </table>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
