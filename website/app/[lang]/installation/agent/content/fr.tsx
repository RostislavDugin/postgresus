import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Installation de l'agent - Documentation Databasus",
  description:
    "Installez l'agent Databasus pour les sauvegardes physiques, les sauvegardes incrémentales, l'archivage des WAL et la restauration à un instant donné (PITR) des bases PostgreSQL.",
  keywords: [
    "agent Databasus",
    "sauvegarde physique PostgreSQL",
    "archivage WAL",
    "PITR",
    "Point-in-Time Recovery",
    "pg_basebackup",
    "sauvegarde incrémentale",
    "reprise après sinistre",
    "agent PostgreSQL",
    "agent de sauvegarde de base de données",
  ],
  openGraph: {
    title: "Installation de l'agent - Documentation Databasus",
    description:
      "Installez l'agent Databasus pour les sauvegardes physiques, les sauvegardes incrémentales, l'archivage des WAL et la restauration à un instant donné (PITR) des bases PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("fr", "installation/agent"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Installation de l'agent - Documentation Databasus",
    description:
      "Installez l'agent Databasus pour les sauvegardes physiques, les sauvegardes incrémentales, l'archivage des WAL et la restauration à un instant donné (PITR) des bases PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "installation/agent"),
    languages: getLanguageAlternates("installation/agent"),
  },
  robots: "index, follow",
};

export default function AgentInstallationPage() {
  const downloadCommand = `curl -L -o databasus-agent "<DATABASUS_HOST>/api/v1/system/agent?arch=<ARCH>" && chmod +x databasus-agent`;

  const postgresqlConf = `wal_level = replica
archive_mode = on
archive_command = 'cp %p <WAL_QUEUE_DIR>/%f.tmp && mv <WAL_QUEUE_DIR>/%f.tmp <WAL_QUEUE_DIR>/%f'`;

  const postgresqlConfDocker = `wal_level = replica
archive_mode = on
archive_command = 'cp %p /wal-queue/%f.tmp && mv /wal-queue/%f.tmp /wal-queue/%f'`;

  const pgHbaEntry = `host    replication   all   127.0.0.1/32   md5`;

  const grantReplication = `ALTER ROLE <YOUR_PG_USER> WITH REPLICATION;`;

  const createWalDir = `mkdir -p /opt/databasus/wal-queue`;

  const walDirPermissions = `chown postgres:postgres /opt/databasus/wal-queue
chmod 755 /opt/databasus/wal-queue`;

  const dockerVolumeExample = `# In your docker run command:
docker run ... -v /opt/databasus/wal-queue:/wal-queue ...

# Or in docker-compose.yml:
volumes:
  - /opt/databasus/wal-queue:/wal-queue`;

  const dockerWalDirPermissions = `# Inside the container (or via docker exec):
chown postgres:postgres /wal-queue`;

  const startCommandHost = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=host \\
  --pg-wal-dir=/opt/databasus/wal-queue`;

  const startCommandFolder = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=host \\
  --pg-host-bin-dir=<PATH_TO_PG_BIN_DIR> \\
  --pg-wal-dir=/opt/databasus/wal-queue`;

  const startCommandDocker = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=docker \\
  --pg-docker-container-name=<CONTAINER_NAME> \\
  --pg-wal-dir=/opt/databasus/wal-queue`;

  const restoreCommand = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --target-dir=<PGDATA_DIR>`;

  const restoreCommandDocker = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --pg-type=docker \\
  --target-dir=<HOST_PGDATA_PATH>`;

  const restoreCommandPitr = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --target-dir=<PGDATA_DIR> \\
  --target-time=<RFC3339_TIMESTAMP>`;

  const archiveCommandCleanup = `# In <PGDATA_DIR>/postgresql.auto.conf, remove or comment out:
# archive_mode = on
# archive_command = '...'`;

  const dockerVolumeMountExample = `# PostgreSQL 17 and earlier
docker run -d -v <HOST_PGDATA_PATH>:/var/lib/postgresql/data postgres:17

# PostgreSQL 18+
docker run -d -v <HOST_PGDATA_PATH>:/var/lib/postgresql/18/docker postgres:18`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Installation de l'agent - Documentation Databasus",
            description:
              "Installez l'agent Databasus pour les sauvegardes physiques, les sauvegardes incrémentales, l'archivage des WAL et la restauration à un instant donné (PITR) des bases PostgreSQL.",
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
              <h1 id="agent-installation">Mode agent</h1>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-red-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-red-500 mt-0.5 shrink-0"
                >
                  <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <path d="M12 9v4M12 17h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>Les sauvegardes par agent sont obsolètes.</strong>{" "}
                    Databasus exécute désormais les sauvegardes physiques et
                    PITR à distance grâce aux sauvegardes natives de PostgreSQL
                    17, sans aucun agent installé sur le serveur de base de
                    données.{" "}
                    <a
                      href="/fr/faq/#why-no-agent"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      Découvrez pourquoi, et comment les sauvegardes PITR
                      fonctionnent désormais
                    </a>
                    .
                  </p>
                </div>
              </div>

              <p className="text-lg text-gray-400">
                L&apos;agent Databasus permet les sauvegardes physiques, les
                sauvegardes incrémentales, l&apos;archivage des WAL et la
                restauration à un instant donné (PITR) pour les bases
                PostgreSQL.
              </p>

              {/* When to use */}
              <h2 id="when-to-use">Quand utiliser l&apos;agent</h2>

              <p>
                Pour la plupart des bases,{" "}
                <strong>
                  les sauvegardes distantes sont l&apos;option la plus simple
                </strong>
                . Databasus se connecte directement à la base via le réseau,
                effectue des sauvegardes logiques avec pg_dump et ne demande
                aucun logiciel supplémentaire sur le serveur de base de données.
                Les sauvegardes distantes fonctionnent aussi bien avec les bases
                managées dans le cloud (RDS, Cloud SQL, Supabase) qu&apos;avec
                les instances auto-hébergées.
              </p>

              <p>
                L&apos;agent est conçu pour les scénarios où les sauvegardes
                distantes ne suffisent pas :
              </p>

              <ul>
                <li>
                  <strong>Reprise après sinistre avec PITR</strong> : restaurez
                  à n&apos;importe quelle seconde entre deux sauvegardes, avec
                  une perte de données quasi nulle
                </li>
                <li>
                  <strong>Sauvegardes physiques</strong> : copie au niveau
                  fichier de l&apos;ensemble du cluster, sauvegarde et
                  restauration plus rapides pour les gros volumes
                </li>
                <li>
                  <strong>Bases non exposées publiquement</strong> :
                  l&apos;agent se connecte en sortie vers Databasus, la base
                  n&apos;a donc jamais besoin d&apos;un point d&apos;accès
                  public
                </li>
                <li>
                  <strong>Sauvegardes incrémentales</strong> : archivage continu
                  des segments WAL combiné à des sauvegardes de base périodiques
                </li>
              </ul>

              {/* In-app guided setup */}
              <h2 id="in-app-setup">
                Configuration guidée dans l&apos;application
              </h2>

              <p>
                Databasus fournit des instructions interactives
                d&apos;installation et de restauration directement dans
                l&apos;interface. Quand vous ouvrez les réglages de l&apos;agent
                pour une base, toutes les commandes sont pré-remplies avec vos
                valeurs : architecture, identifiant de la base, jeton
                d&apos;agent, hôte Databasus et type de déploiement PostgreSQL.
                Vous pouvez copier chaque commande et l&apos;exécuter sur votre
                serveur.
              </p>

              <p>
                La documentation ci-dessous couvre les mêmes étapes, en guise de
                référence et pour ceux qui préfèrent suivre un guide en dehors
                de l&apos;interface.
              </p>

              {/* Requirements */}
              <h2 id="requirements">Prérequis</h2>

              <ul>
                <li>
                  <strong>PostgreSQL 15 ou plus récent</strong>
                </li>
                <li>
                  <strong>Linux</strong> (amd64 ou arm64)
                </li>
                <li>
                  <strong>Accès réseau</strong> de l&apos;agent vers votre
                  instance Databasus (en sortie uniquement : la base n&apos;a
                  pas besoin d&apos;être joignable depuis Databasus)
                </li>
              </ul>

              {/* Installation */}
              <h2 id="installation">Installation</h2>

              <h3 id="step-1-download">Étape 1 — Télécharger l&apos;agent</h3>

              <p>
                Téléchargez le binaire de l&apos;agent sur le serveur où
                s&apos;exécute PostgreSQL. Remplacez{" "}
                <code>&lt;DATABASUS_HOST&gt;</code> par l&apos;URL de votre
                instance Databasus et <code>&lt;ARCH&gt;</code> par{" "}
                <code>amd64</code> ou <code>arm64</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={downloadCommand} />
                </div>
              </div>

              <h3 id="step-2-postgresql-conf">
                Étape 2 — Configurer postgresql.conf
              </h3>

              <p>
                Ajoutez ou mettez à jour ces réglages dans votre{" "}
                <code>postgresql.conf</code>, puis{" "}
                <strong>redémarrez PostgreSQL</strong>.
              </p>

              <p>
                <strong>Pour les installations sur l&apos;hôte</strong>{" "}
                (remplacez <code>&lt;WAL_QUEUE_DIR&gt;</code> par le chemin
                réel, par ex. <code>/opt/databasus/wal-queue</code>) :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConf}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={postgresqlConf} />
                </div>
              </div>

              <p>
                <strong>Pour les installations Docker</strong>, le chemin de{" "}
                <code>archive_command</code> (<code>/wal-queue</code>) est le
                chemin <strong>à l&apos;intérieur du conteneur</strong>. Il doit
                correspondre à la cible du montage de volume (voir l&apos;étape
                5).
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConfDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={postgresqlConfDocker} />
                </div>
              </div>

              <h3 id="step-3-pg-hba">Étape 3 — Configurer pg_hba.conf</h3>

              <p>
                Ajoutez cette ligne dans <code>pg_hba.conf</code>. Elle est
                requise pour que <code>pg_basebackup</code> puisse faire des
                sauvegardes complètes, pas pour la réplication en streaming.
                Ajustez l&apos;adresse et la méthode d&apos;authentification si
                nécessaire, puis rechargez PostgreSQL.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pgHbaEntry}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={pgHbaEntry} />
                </div>
              </div>

              <h3 id="step-4-replication">
                Étape 4 — Accorder le privilège de réplication
              </h3>

              <p>
                C&apos;est une exigence de PostgreSQL pour exécuter{" "}
                <code>pg_basebackup</code> : cela ne met pas en place un
                réplica.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{grantReplication}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={grantReplication} />
                </div>
              </div>

              <h3 id="step-5-wal-queue">
                Étape 5 — Créer le répertoire de file d&apos;attente WAL
              </h3>

              <p>
                PostgreSQL y dépose les fichiers d&apos;archive WAL que
                l&apos;agent envoie ensuite.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createWalDir}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={createWalDir} />
                </div>
              </div>

              <p>
                Assurez-vous que PostgreSQL peut écrire dans ce répertoire et
                que l&apos;agent peut le lire :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{walDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={walDirPermissions} />
                </div>
              </div>

              <p>
                <strong>Pour les installations Docker</strong>, le répertoire de
                file d&apos;attente WAL doit être un volume partagé entre le
                conteneur PostgreSQL et l&apos;hôte. L&apos;agent lit les
                fichiers WAL depuis le chemin de l&apos;hôte, tandis que
                PostgreSQL écrit dans le chemin du conteneur via{" "}
                <code>archive_command</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={dockerVolumeExample} />
                </div>
              </div>

              <p>
                Vérifiez que le répertoire à l&apos;intérieur du conteneur
                appartient à l&apos;utilisateur <code>postgres</code> :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerWalDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={dockerWalDirPermissions} />
                </div>
              </div>

              <h3 id="step-6-start">Étape 6 — Démarrer l&apos;agent</h3>

              <p>
                Remplacez les valeurs entre <code>&lt;ANGLE_BRACKETS&gt;</code>{" "}
                par vos valeurs réelles.
              </p>

              <p>
                <strong>PostgreSQL installé au niveau du système</strong>{" "}
                (pg_basebackup disponible dans le PATH) :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={startCommandHost} />
                </div>
              </div>

              <p>
                <strong>PostgreSQL dans un dossier spécifique</strong> (par ex.{" "}
                <code>/usr/lib/postgresql/17/bin</code>) :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandFolder}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={startCommandFolder} />
                </div>
              </div>

              <p>
                <strong>Docker</strong> (utilisez le port PostgreSQL{" "}
                <strong>à l&apos;intérieur du conteneur</strong>, en général
                5432, et non le port mappé sur l&apos;hôte) :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={startCommandDocker} />
                </div>
              </div>

              <h3 id="after-installation">Après l&apos;installation</h3>

              <ul>
                <li>
                  L&apos;agent s&apos;exécute en arrière-plan après{" "}
                  <code>start</code>
                </li>
                <li>
                  Vérifier le statut : <code>./databasus-agent status</code>
                </li>
                <li>
                  Consulter les journaux : <code>databasus.log</code> dans le
                  répertoire de travail
                </li>
                <li>
                  Arrêter l&apos;agent : <code>./databasus-agent stop</code>
                </li>
              </ul>

              {/* Restore */}
              <h2 id="restore">
                Restaurer depuis une sauvegarde de l&apos;agent
              </h2>

              <p>
                Restaurez une sauvegarde physique ou incrémentale vers un
                répertoire cible. Pour la restauration à un instant donné,
                ajoutez l&apos;option <code>--target-time</code> afin de
                restaurer à un moment précis.
              </p>

              <h3 id="restore-step-1">Étape 1 — Télécharger l&apos;agent</h3>

              <p>
                Téléchargez le binaire de l&apos;agent sur le serveur où vous
                voulez restaurer (même commande que l&apos;étape 1 de
                l&apos;installation).
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={downloadCommand} />
                </div>
              </div>

              <h3 id="restore-step-2">Étape 2 — Arrêter PostgreSQL</h3>

              <p>
                PostgreSQL doit être arrêté avant la restauration. Le répertoire
                cible doit être vide.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; stop</code>
                </pre>
              </div>

              <p>Pour Docker :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker stop &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>

              <h3 id="restore-step-3">Étape 3 — Lancer la restauration</h3>

              <p>
                Remplacez <code>&lt;YOUR_AGENT_TOKEN&gt;</code> par votre jeton
                d&apos;agent et <code>&lt;PGDATA_DIR&gt;</code> par le chemin
                d&apos;un répertoire de données PostgreSQL vide.
              </p>

              <p>
                <strong>Installation sur l&apos;hôte :</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={restoreCommand} />
                </div>
              </div>

              <p>
                <strong>Installation Docker</strong> (
                <code>&lt;HOST_PGDATA_PATH&gt;</code> est le chemin sur
                l&apos;hôte qui sera monté comme volume pgdata du conteneur) :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={restoreCommandDocker} />
                </div>
              </div>

              <p>
                Montez <code>&lt;HOST_PGDATA_PATH&gt;</code> sur le chemin
                PGDATA du conteneur lors de la (re)création du conteneur
                postgres. Le chemin dépend de la version majeure : PostgreSQL
                18+ utilise{" "}
                <code>/var/lib/postgresql/&lt;major&gt;/docker</code> ;
                PostgreSQL 17 et antérieurs utilisent{" "}
                <code>/var/lib/postgresql/data</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeMountExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={dockerVolumeMountExample} />
                </div>
              </div>

              <p>
                Pour la <strong>restauration à un instant donné</strong> (PITR),
                ajoutez <code>--target-time</code> avec un horodatage RFC 3339
                (par ex. <code>2025-01-15T14:30:00Z</code>) :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandPitr}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={restoreCommandPitr} />
                </div>
              </div>

              <h3 id="restore-step-4">Étape 4 — Gérer archive_command</h3>

              <p>
                La sauvegarde restaurée contient la configuration{" "}
                <code>archive_command</code> d&apos;origine. Après la
                récupération, PostgreSQL échouera à archiver les fichiers WAL,
                sauf si vous faites l&apos;un des deux :
              </p>

              <ul>
                <li>
                  <strong>Rattacher l&apos;agent</strong> : montez le répertoire
                  de file d&apos;attente WAL et démarrez l&apos;agent Databasus
                  sur l&apos;instance restaurée, comme lors de
                  l&apos;installation d&apos;origine.
                </li>
                <li>
                  <strong>Désactiver l&apos;archivage</strong> : si vous
                  n&apos;avez pas encore besoin de sauvegardes continues,
                  commentez ou réinitialisez les réglages d&apos;archivage dans{" "}
                  <code>postgresql.auto.conf</code> :
                </li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{archiveCommandCleanup}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={archiveCommandCleanup} />
                </div>
              </div>

              <h3 id="restore-step-5">Étape 5 — Démarrer PostgreSQL</h3>

              <p>
                Démarrez PostgreSQL pour lancer la récupération des WAL. Il
                rejouera automatiquement les segments WAL.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; start</code>
                </pre>
              </div>

              <p>Pour Docker :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker start &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>

              <h3 id="restore-step-6">Étape 6 — Nettoyer</h3>

              <p>
                Une fois la récupération terminée, supprimez le répertoire de
                restauration des WAL :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>rm -rf &lt;PGDATA_DIR&gt;/databasus-wal-restore/</code>
                </pre>
              </div>

              {/* How it works */}
              <h2 id="how-it-works">Comment ça marche</h2>

              <p>
                L&apos;agent Databasus est un binaire Go léger qui exécute deux
                processus concurrents :
              </p>

              <ul>
                <li>
                  <strong>Streaming des WAL</strong> : il récupère les segments
                  WAL dans le répertoire de file d&apos;attente environ toutes
                  les 10 secondes et les envoie vers Databasus
                </li>
                <li>
                  <strong>Sauvegardes de base périodiques</strong> : il exécute{" "}
                  <code>pg_basebackup</code> selon le planning configuré pour
                  créer des sauvegardes physiques complètes du cluster
                </li>
              </ul>

              <p>
                Pendant la restauration, l&apos;agent télécharge la sauvegarde
                de base et tous les segments WAL nécessaires, puis configure{" "}
                <code>recovery.signal</code> et <code>restore_command</code>{" "}
                dans <code>postgresql.auto.conf</code>. Au démarrage, PostgreSQL
                rejoue les segments WAL jusqu&apos;au point de récupération
                cible.
              </p>

              <p>
                L&apos;agent initie toujours la connexion vers Databasus (en
                sortie). Le serveur de base de données n&apos;a pas besoin
                d&apos;accepter de connexions entrantes depuis Databasus, ce qui
                le rend adapté aux réseaux privés et aux environnements derrière
                un pare-feu.
              </p>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
