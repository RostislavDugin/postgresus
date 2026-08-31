import type { Metadata } from "next";
import {
  OG_LOCALES,
  getLanguageAlternates,
  getLocalizedUrl,
} from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Instalación del agente - Documentación de Databasus",
  description:
    "Instale el agente de Databasus para copias de seguridad físicas, copias incrementales, archivado de WAL y recuperación a un punto en el tiempo (PITR) de bases de datos PostgreSQL.",
  keywords: [
    "agente de Databasus",
    "copia de seguridad física de PostgreSQL",
    "archivado de WAL",
    "PITR",
    "Point-in-Time Recovery",
    "pg_basebackup",
    "copia de seguridad incremental",
    "recuperación ante desastres",
    "agente de PostgreSQL",
    "agente de respaldo de bases de datos",
  ],
  openGraph: {
    title: "Instalación del agente - Documentación de Databasus",
    description:
      "Instale el agente de Databasus para copias de seguridad físicas, copias incrementales, archivado de WAL y recuperación a un punto en el tiempo (PITR) de bases de datos PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("es", "installation/agent"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Instalación del agente - Documentación de Databasus",
    description:
      "Instale el agente de Databasus para copias de seguridad físicas, copias incrementales, archivado de WAL y recuperación a un punto en el tiempo (PITR) de bases de datos PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "installation/agent"),
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
            headline: "Instalación del agente - Documentación de Databasus",
            description:
              "Instale el agente de Databasus para copias de seguridad físicas, copias incrementales, archivado de WAL y recuperación a un punto en el tiempo (PITR) de bases de datos PostgreSQL.",
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

      <DocsNavbarComponent lang="es" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="es" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="agent-installation">Modo agente</h1>

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
                    <strong>
                      Las copias mediante agente están obsoletas.
                    </strong>{" "}
                    Databasus ahora ejecuta las copias físicas y PITR de forma
                    remota usando las copias nativas de PostgreSQL 17, sin
                    ningún agente instalado en el servidor de la base de datos.{" "}
                    <a
                      href="/es/faq/#why-no-agent"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      Lea por qué y cómo funcionan ahora las copias PITR
                    </a>
                    .
                  </p>
                </div>
              </div>

              <p className="text-lg text-gray-400">
                El agente de Databasus habilita copias de seguridad físicas,
                copias incrementales, archivado de WAL y recuperación a un
                punto en el tiempo (PITR) para bases de datos PostgreSQL.
              </p>

              {/* When to use */}
              <h2 id="when-to-use">Cuándo usar el agente</h2>

              <p>
                Para la mayoría de las bases de datos,{" "}
                <strong>las copias remotas son la opción más simple</strong>.
                Databasus se conecta directamente a la base de datos por la red,
                realiza copias lógicas con pg_dump y no requiere ningún software
                adicional en el servidor de la base de datos. Las copias remotas
                funcionan tanto con bases de datos gestionadas en la nube (RDS,
                Cloud SQL, Supabase) como con instancias autoalojadas.
              </p>

              <p>
                El agente está pensado para escenarios donde las copias remotas
                no bastan:
              </p>

              <ul>
                <li>
                  <strong>Recuperación ante desastres con PITR</strong>:
                  restaure a cualquier segundo entre copias con una pérdida de
                  datos casi nula
                </li>
                <li>
                  <strong>Copias físicas</strong>: copia a nivel de archivos de
                  todo el clúster, con copia y restauración más rápidas para
                  conjuntos de datos grandes
                </li>
                <li>
                  <strong>Bases de datos no expuestas públicamente</strong>: el
                  agente se conecta de forma saliente a Databasus, así que la
                  base de datos nunca necesita un punto de acceso público
                </li>
                <li>
                  <strong>Copias incrementales</strong>: archivado continuo de
                  segmentos WAL combinado con copias base periódicas
                </li>
              </ul>

              {/* In-app guided setup */}
              <h2 id="in-app-setup">Configuración guiada en la aplicación</h2>

              <p>
                Databasus ofrece instrucciones interactivas de instalación y
                restauración directamente en la interfaz. Al abrir la
                configuración del agente para una base de datos, todos los
                comandos vienen rellenados con sus valores concretos:
                arquitectura, ID de la base de datos, token del agente, host de
                Databasus y tipo de despliegue de PostgreSQL. Puede copiar cada
                comando y ejecutarlo en su servidor.
              </p>

              <p>
                La documentación siguiente cubre los mismos pasos como
                referencia y para quienes prefieren seguir una guía fuera de la
                interfaz.
              </p>

              {/* Requirements */}
              <h2 id="requirements">Requisitos</h2>

              <ul>
                <li>
                  <strong>PostgreSQL 15 o superior</strong>
                </li>
                <li>
                  <strong>Linux</strong> (amd64 o arm64)
                </li>
                <li>
                  <strong>Acceso de red</strong> desde el agente a su instancia
                  de Databasus (solo saliente; la base de datos no necesita ser
                  accesible desde Databasus)
                </li>
              </ul>

              {/* Installation */}
              <h2 id="installation">Instalación</h2>

              <h3 id="step-1-download">Paso 1 — Descargar el agente</h3>

              <p>
                Descargue el binario del agente en el servidor donde se ejecuta
                PostgreSQL. Reemplace <code>&lt;DATABASUS_HOST&gt;</code> por la
                URL de su instancia de Databasus y <code>&lt;ARCH&gt;</code> por{" "}
                <code>amd64</code> o <code>arm64</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={downloadCommand} />
                </div>
              </div>

              <h3 id="step-2-postgresql-conf">
                Paso 2 — Configurar postgresql.conf
              </h3>

              <p>
                Añada o actualice estos ajustes en su{" "}
                <code>postgresql.conf</code> y luego{" "}
                <strong>reinicie PostgreSQL</strong>.
              </p>

              <p>
                <strong>Para instalaciones en el host</strong> (reemplace{" "}
                <code>&lt;WAL_QUEUE_DIR&gt;</code> por la ruta real, p. ej.{" "}
                <code>/opt/databasus/wal-queue</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConf}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={postgresqlConf} />
                </div>
              </div>

              <p>
                <strong>Para instalaciones en Docker</strong>, la ruta del{" "}
                <code>archive_command</code> (<code>/wal-queue</code>) es la
                ruta <strong>dentro del contenedor</strong>. Debe coincidir con
                el destino del montaje del volumen; consulte el paso 5.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConfDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={postgresqlConfDocker} />
                </div>
              </div>

              <h3 id="step-3-pg-hba">Paso 3 — Configurar pg_hba.conf</h3>

              <p>
                Añada esta línea a <code>pg_hba.conf</code>. Es necesaria para
                que <code>pg_basebackup</code> pueda tomar copias completas, no
                para replicación en streaming. Ajuste la dirección y el método
                de autenticación según convenga y luego recargue PostgreSQL.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pgHbaEntry}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={pgHbaEntry} />
                </div>
              </div>

              <h3 id="step-4-replication">
                Paso 4 — Conceder el privilegio de replicación
              </h3>

              <p>
                Es un requisito de PostgreSQL para ejecutar{" "}
                <code>pg_basebackup</code>; no configura ninguna réplica.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{grantReplication}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={grantReplication} />
                </div>
              </div>

              <h3 id="step-5-wal-queue">
                Paso 5 — Crear el directorio de la cola de WAL
              </h3>

              <p>
                PostgreSQL deja aquí los archivos de WAL archivados para que el
                agente los suba.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createWalDir}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={createWalDir} />
                </div>
              </div>

              <p>
                Asegúrese de que PostgreSQL pueda escribir en el directorio y
                el agente pueda leerlo:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{walDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={walDirPermissions} />
                </div>
              </div>

              <p>
                <strong>Para instalaciones en Docker</strong>, el directorio de
                la cola de WAL debe ser un volumen compartido entre el
                contenedor de PostgreSQL y el host. El agente lee los archivos
                WAL desde la ruta del host, mientras que PostgreSQL escribe en
                la ruta del contenedor mediante <code>archive_command</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={dockerVolumeExample} />
                </div>
              </div>

              <p>
                Asegúrese de que el directorio dentro del contenedor pertenezca
                al usuario <code>postgres</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerWalDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={dockerWalDirPermissions} />
                </div>
              </div>

              <h3 id="step-6-start">Paso 6 — Iniciar el agente</h3>

              <p>
                Reemplace los marcadores entre{" "}
                <code>&lt;ANGLE_BRACKETS&gt;</code> por sus valores reales.
              </p>

              <p>
                <strong>PostgreSQL instalado en el sistema</strong>{" "}
                (pg_basebackup disponible en el PATH):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={startCommandHost} />
                </div>
              </div>

              <p>
                <strong>PostgreSQL en una carpeta concreta</strong> (p. ej.{" "}
                <code>/usr/lib/postgresql/17/bin</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandFolder}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={startCommandFolder} />
                </div>
              </div>

              <p>
                <strong>Docker</strong> (use el puerto de PostgreSQL{" "}
                <strong>dentro del contenedor</strong>, normalmente 5432, no el
                puerto mapeado al host):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={startCommandDocker} />
                </div>
              </div>

              <h3 id="after-installation">Después de la instalación</h3>

              <ul>
                <li>
                  El agente sigue ejecutándose en segundo plano tras{" "}
                  <code>start</code>
                </li>
                <li>
                  Comprobar el estado: <code>./databasus-agent status</code>
                </li>
                <li>
                  Ver los registros: <code>databasus.log</code> en el directorio
                  de trabajo
                </li>
                <li>
                  Detener el agente: <code>./databasus-agent stop</code>
                </li>
              </ul>

              {/* Restore */}
              <h2 id="restore">Restaurar desde una copia del agente</h2>

              <p>
                Restaure una copia física o incremental en un directorio de
                destino. Para la recuperación a un punto en el tiempo, añada la
                opción <code>--target-time</code> para restaurar a un momento
                concreto.
              </p>

              <h3 id="restore-step-1">Paso 1 — Descargar el agente</h3>

              <p>
                Descargue el binario del agente en el servidor donde quiere
                restaurar (el mismo comando que en el paso 1 de la instalación).
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={downloadCommand} />
                </div>
              </div>

              <h3 id="restore-step-2">Paso 2 — Detener PostgreSQL</h3>

              <p>
                PostgreSQL debe estar detenido antes de restaurar. El directorio
                de destino debe estar vacío.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; stop</code>
                </pre>
              </div>

              <p>Para Docker:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker stop &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>

              <h3 id="restore-step-3">Paso 3 — Ejecutar la restauración</h3>

              <p>
                Reemplace <code>&lt;YOUR_AGENT_TOKEN&gt;</code> por su token de
                agente y <code>&lt;PGDATA_DIR&gt;</code> por la ruta a un
                directorio de datos de PostgreSQL vacío.
              </p>

              <p>
                <strong>Instalación en el host:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={restoreCommand} />
                </div>
              </div>

              <p>
                <strong>Instalación en Docker</strong> (
                <code>&lt;HOST_PGDATA_PATH&gt;</code> es la ruta en el host que
                se montará como el volumen pgdata del contenedor):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={restoreCommandDocker} />
                </div>
              </div>

              <p>
                Monte <code>&lt;HOST_PGDATA_PATH&gt;</code> en la ruta PGDATA
                del contenedor al (re)crear el contenedor de postgres. La ruta
                depende de la versión mayor: PostgreSQL 18+ usa{" "}
                <code>/var/lib/postgresql/&lt;major&gt;/docker</code>;
                PostgreSQL 17 y anteriores usan{" "}
                <code>/var/lib/postgresql/data</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeMountExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={dockerVolumeMountExample} />
                </div>
              </div>

              <p>
                Para la <strong>recuperación a un punto en el tiempo</strong>{" "}
                (PITR), añada <code>--target-time</code> con una marca de tiempo
                RFC 3339 (p. ej. <code>2025-01-15T14:30:00Z</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandPitr}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={restoreCommandPitr} />
                </div>
              </div>

              <h3 id="restore-step-4">
                Paso 4 — Gestionar archive_command
              </h3>

              <p>
                La copia restaurada incluye la configuración original de{" "}
                <code>archive_command</code>. PostgreSQL fallará al archivar los
                WAL tras la recuperación, a menos que haga una de estas dos
                cosas:
              </p>

              <ul>
                <li>
                  <strong>Volver a conectar el agente</strong>: monte el
                  directorio de la cola de WAL e inicie el agente de Databasus
                  en la instancia restaurada, igual que en la configuración
                  original.
                </li>
                <li>
                  <strong>Desactivar el archivado</strong>: si todavía no
                  necesita copias continuas, comente o restablezca los ajustes
                  de archivado en <code>postgresql.auto.conf</code>:
                </li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{archiveCommandCleanup}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={archiveCommandCleanup} />
                </div>
              </div>

              <h3 id="restore-step-5">Paso 5 — Iniciar PostgreSQL</h3>

              <p>
                Inicie PostgreSQL para comenzar la recuperación de WAL.
                Reproducirá los segmentos WAL automáticamente.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; start</code>
                </pre>
              </div>

              <p>Para Docker:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker start &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>

              <h3 id="restore-step-6">Paso 6 — Limpiar</h3>

              <p>
                Una vez completada la recuperación, elimine el directorio de
                restauración de WAL:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>rm -rf &lt;PGDATA_DIR&gt;/databasus-wal-restore/</code>
                </pre>
              </div>

              {/* How it works */}
              <h2 id="how-it-works">Cómo funciona</h2>

              <p>
                El agente de Databasus es un binario ligero escrito en Go que
                ejecuta dos procesos concurrentes:
              </p>

              <ul>
                <li>
                  <strong>Streaming de WAL</strong>: recoge los archivos de
                  segmentos WAL del directorio de la cola aproximadamente cada
                  10 segundos y los sube a Databasus
                </li>
                <li>
                  <strong>Copias base periódicas</strong>: ejecuta{" "}
                  <code>pg_basebackup</code> según el horario configurado para
                  crear copias físicas completas del clúster de la base de
                  datos
                </li>
              </ul>

              <p>
                Durante la restauración, el agente descarga la copia base y
                todos los segmentos WAL pertinentes, y luego configura{" "}
                <code>recovery.signal</code> y <code>restore_command</code> en{" "}
                <code>postgresql.auto.conf</code>. Al arrancar, PostgreSQL
                reproduce los segmentos WAL hasta alcanzar el punto de
                recuperación objetivo.
              </p>

              <p>
                El agente siempre inicia la conexión hacia Databasus
                (saliente). El servidor de la base de datos no necesita aceptar
                conexiones entrantes desde Databasus, lo que lo hace adecuado
                para redes privadas y entornos con cortafuegos.
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
