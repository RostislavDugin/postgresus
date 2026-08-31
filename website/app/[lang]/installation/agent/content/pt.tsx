import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Instalação do agente - Documentação do Databasus",
  description:
    "Instale o agente do Databasus para backups físicos, backups incrementais, arquivamento de WAL e Point-in-Time Recovery (PITR) de bases de dados PostgreSQL.",
  keywords: [
    "agente Databasus",
    "backup físico PostgreSQL",
    "arquivamento de WAL",
    "PITR",
    "Point-in-Time Recovery",
    "pg_basebackup",
    "backup incremental",
    "recuperação de desastres",
    "agente PostgreSQL",
    "agente de backup de base de dados",
  ],
  openGraph: {
    title: "Instalação do agente - Documentação do Databasus",
    description:
      "Instale o agente do Databasus para backups físicos, backups incrementais, arquivamento de WAL e Point-in-Time Recovery (PITR) de bases de dados PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("pt", "installation/agent"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Instalação do agente - Documentação do Databasus",
    description:
      "Instale o agente do Databasus para backups físicos, backups incrementais, arquivamento de WAL e Point-in-Time Recovery (PITR) de bases de dados PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "installation/agent"),
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
            headline: "Instalação do agente - Documentação do Databasus",
            description:
              "Instale o agente do Databasus para backups físicos, backups incrementais, arquivamento de WAL e Point-in-Time Recovery (PITR) de bases de dados PostgreSQL.",
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

      <DocsNavbarComponent lang="pt" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="pt" />

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
                    <strong>Os backups via agente estão descontinuados.</strong>{" "}
                    O Databasus agora executa backups físicos e PITR
                    remotamente, com os backups nativos do PostgreSQL 17, sem
                    nenhum agente instalado no servidor da base de dados.{" "}
                    <a
                      href="/pt/faq/#why-no-agent"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      Leia o porquê e como os backups PITR funcionam agora
                    </a>
                    .
                  </p>
                </div>
              </div>

              <p className="text-lg text-gray-400">
                O agente do Databasus permite backups físicos, backups
                incrementais, arquivamento de WAL e Point-in-Time Recovery
                (PITR) para bases de dados PostgreSQL.
              </p>

              {/* When to use */}
              <h2 id="when-to-use">Quando usar o agente</h2>

              <p>
                Para a maioria das bases de dados,{" "}
                <strong>os backups remotos são a opção mais simples</strong>. O
                Databasus se conecta diretamente à base de dados pela rede,
                executa backups lógicos com o pg_dump e não exige nenhum
                software adicional no servidor da base de dados. Os backups
                remotos funcionam tanto com bases gerenciadas na nuvem (RDS,
                Cloud SQL, Supabase) quanto com instâncias auto-hospedadas.
              </p>

              <p>
                O agente foi pensado para cenários em que os backups remotos não
                bastam:
              </p>

              <ul>
                <li>
                  <strong>Recuperação de desastres com PITR</strong>: restaure
                  para qualquer segundo entre backups, com perda de dados quase
                  nula
                </li>
                <li>
                  <strong>Backups físicos</strong>: cópia no nível dos dados de
                  todo o cluster, com backup e restauração mais rápidos para
                  volumes grandes
                </li>
                <li>
                  <strong>Bases de dados não expostas publicamente</strong>: o
                  agente se conecta ao Databasus por conexões de saída, então a
                  base de dados nunca precisa de um endpoint público
                </li>
                <li>
                  <strong>Backups incrementais</strong>: arquivamento contínuo
                  de segmentos WAL combinado com backups de base periódicos
                </li>
              </ul>

              {/* In-app guided setup */}
              <h2 id="in-app-setup">Instalação guiada na interface</h2>

              <p>
                O Databasus mostra instruções interativas de instalação e
                restauração diretamente na interface. Ao abrir as configurações
                do agente de uma base de dados, todos os comandos já vêm
                preenchidos com os seus valores: arquitetura, ID da base de
                dados, token do agente, host do Databasus e tipo de implantação
                do PostgreSQL. Basta copiar cada comando e executá-lo no seu
                servidor.
              </p>

              <p>
                A documentação abaixo cobre os mesmos passos, como referência e
                para quem prefere seguir um guia fora da interface.
              </p>

              {/* Requirements */}
              <h2 id="requirements">Requisitos</h2>

              <ul>
                <li>
                  <strong>PostgreSQL 15 ou mais recente</strong>
                </li>
                <li>
                  <strong>Linux</strong> (amd64 ou arm64)
                </li>
                <li>
                  <strong>Acesso de rede</strong> do agente à sua instância do
                  Databasus (apenas de saída: a base de dados não precisa ser
                  alcançável a partir do Databasus)
                </li>
              </ul>

              {/* Installation */}
              <h2 id="installation">Instalação</h2>

              <h3 id="step-1-download">Passo 1 — Baixar o agente</h3>

              <p>
                Baixe o binário do agente no servidor onde o PostgreSQL está
                rodando. Substitua <code>&lt;DATABASUS_HOST&gt;</code> pela URL
                da sua instância do Databasus e <code>&lt;ARCH&gt;</code> por{" "}
                <code>amd64</code> ou <code>arm64</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={downloadCommand} />
                </div>
              </div>

              <h3 id="step-2-postgresql-conf">
                Passo 2 — Configurar o postgresql.conf
              </h3>

              <p>
                Adicione ou atualize estas configurações no seu{" "}
                <code>postgresql.conf</code> e depois{" "}
                <strong>reinicie o PostgreSQL</strong>.
              </p>

              <p>
                <strong>Para instalações no host</strong> (substitua{" "}
                <code>&lt;WAL_QUEUE_DIR&gt;</code> pelo caminho real, por
                exemplo <code>/opt/databasus/wal-queue</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConf}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={postgresqlConf} />
                </div>
              </div>

              <p>
                <strong>Para instalações em Docker</strong>, o caminho do{" "}
                <code>archive_command</code> (<code>/wal-queue</code>) é o
                caminho <strong>dentro do container</strong>. Ele precisa
                corresponder ao destino do volume montado — veja o Passo 5.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConfDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={postgresqlConfDocker} />
                </div>
              </div>

              <h3 id="step-3-pg-hba">Passo 3 — Configurar o pg_hba.conf</h3>

              <p>
                Adicione esta linha ao <code>pg_hba.conf</code>. Ela é
                necessária para o <code>pg_basebackup</code> fazer backups
                completos — não para replicação por streaming. Ajuste o endereço
                e o método de autenticação conforme necessário e depois
                recarregue o PostgreSQL.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pgHbaEntry}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={pgHbaEntry} />
                </div>
              </div>

              <h3 id="step-4-replication">
                Passo 4 — Conceder o privilégio de replicação
              </h3>

              <p>
                Este é um requisito do PostgreSQL para executar o{" "}
                <code>pg_basebackup</code> — não cria nenhuma réplica.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{grantReplication}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={grantReplication} />
                </div>
              </div>

              <h3 id="step-5-wal-queue">
                Passo 5 — Criar o diretório de fila de WAL
              </h3>

              <p>
                O PostgreSQL coloca aqui os segmentos WAL arquivados para o
                agente enviar.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createWalDir}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={createWalDir} />
                </div>
              </div>

              <p>
                Garanta que o PostgreSQL pode escrever no diretório e que o
                agente pode lê-lo:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{walDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={walDirPermissions} />
                </div>
              </div>

              <p>
                <strong>Para instalações em Docker</strong>, o diretório da fila
                de WAL deve ser um volume compartilhado entre o container do
                PostgreSQL e o host. O agente lê os segmentos WAL pelo caminho
                do host, enquanto o PostgreSQL escreve no caminho do container
                via <code>archive_command</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={dockerVolumeExample} />
                </div>
              </div>

              <p>
                Garanta que o diretório dentro do container é propriedade da
                conta <code>postgres</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerWalDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={dockerWalDirPermissions} />
                </div>
              </div>

              <h3 id="step-6-start">Passo 6 — Iniciar o agente</h3>

              <p>
                Substitua os marcadores em <code>&lt;ANGLE_BRACKETS&gt;</code>{" "}
                pelos seus valores reais.
              </p>

              <p>
                <strong>PostgreSQL instalado no sistema</strong> (pg_basebackup
                disponível no PATH):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={startCommandHost} />
                </div>
              </div>

              <p>
                <strong>PostgreSQL numa pasta específica</strong> (por exemplo{" "}
                <code>/usr/lib/postgresql/17/bin</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandFolder}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={startCommandFolder} />
                </div>
              </div>

              <p>
                <strong>Docker</strong> (use a porta do PostgreSQL{" "}
                <strong>dentro do container</strong>, normalmente 5432, e não a
                porta mapeada no host):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={startCommandDocker} />
                </div>
              </div>

              <h3 id="after-installation">Depois da instalação</h3>

              <ul>
                <li>
                  O agente fica rodando em segundo plano depois do{" "}
                  <code>start</code>
                </li>
                <li>
                  Verificar o status: <code>./databasus-agent status</code>
                </li>
                <li>
                  Ver os logs: <code>databasus.log</code> no diretório de
                  trabalho
                </li>
                <li>
                  Parar o agente: <code>./databasus-agent stop</code>
                </li>
              </ul>

              {/* Restore */}
              <h2 id="restore">Restaurar a partir de um backup do agente</h2>

              <p>
                Restaure um backup físico ou incremental para um diretório de
                destino. Para Point-in-Time Recovery, adicione a opção{" "}
                <code>--target-time</code> para restaurar para um momento
                específico.
              </p>

              <h3 id="restore-step-1">Passo 1 — Baixar o agente</h3>

              <p>
                Baixe o binário do agente no servidor onde você quer restaurar
                (mesmo comando do Passo 1 da instalação).
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={downloadCommand} />
                </div>
              </div>

              <h3 id="restore-step-2">Passo 2 — Parar o PostgreSQL</h3>

              <p>
                O PostgreSQL deve estar parado antes da restauração. O diretório
                de destino deve estar vazio.
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

              <h3 id="restore-step-3">Passo 3 — Executar a restauração</h3>

              <p>
                Substitua <code>&lt;YOUR_AGENT_TOKEN&gt;</code> pelo token do
                seu agente e <code>&lt;PGDATA_DIR&gt;</code> pelo caminho de um
                diretório de dados PostgreSQL vazio.
              </p>

              <p>
                <strong>Instalação no host:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={restoreCommand} />
                </div>
              </div>

              <p>
                <strong>Instalação em Docker</strong> (
                <code>&lt;HOST_PGDATA_PATH&gt;</code> é o caminho no host que
                será montado como o volume pgdata do container):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={restoreCommandDocker} />
                </div>
              </div>

              <p>
                Monte <code>&lt;HOST_PGDATA_PATH&gt;</code> no caminho PGDATA do
                container ao (re)criar o container do postgres. O caminho
                depende da versão principal: o PostgreSQL 18+ usa{" "}
                <code>/var/lib/postgresql/&lt;major&gt;/docker</code>; o
                PostgreSQL 17 e anteriores usam{" "}
                <code>/var/lib/postgresql/data</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeMountExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={dockerVolumeMountExample} />
                </div>
              </div>

              <p>
                Para <strong>Point-in-Time Recovery</strong> (PITR), adicione{" "}
                <code>--target-time</code> com uma data/hora em RFC 3339 (por
                exemplo <code>2025-01-15T14:30:00Z</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandPitr}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={restoreCommandPitr} />
                </div>
              </div>

              <h3 id="restore-step-4">Passo 4 — Ajustar o archive_command</h3>

              <p>
                O backup restaurado inclui a configuração original do{" "}
                <code>archive_command</code>. O PostgreSQL falhará ao arquivar
                segmentos WAL depois da recuperação, a menos que você faça uma
                de duas coisas:
              </p>

              <ul>
                <li>
                  <strong>Reconectar o agente</strong>: monte o diretório da
                  fila de WAL e inicie o agente do Databasus na instância
                  restaurada, igual à configuração original.
                </li>
                <li>
                  <strong>Desativar o arquivamento</strong>: se você ainda não
                  precisa de backups contínuos, comente ou remova as
                  configurações de arquivamento no{" "}
                  <code>postgresql.auto.conf</code>:
                </li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{archiveCommandCleanup}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={archiveCommandCleanup} />
                </div>
              </div>

              <h3 id="restore-step-5">Passo 5 — Iniciar o PostgreSQL</h3>

              <p>
                Inicie o PostgreSQL para começar a recuperação por WAL. Ele
                reproduzirá automaticamente os segmentos WAL.
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

              <h3 id="restore-step-6">Passo 6 — Limpeza</h3>

              <p>
                Quando a recuperação terminar, remova o diretório de restauração
                de WAL:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>rm -rf &lt;PGDATA_DIR&gt;/databasus-wal-restore/</code>
                </pre>
              </div>

              {/* How it works */}
              <h2 id="how-it-works">Como funciona</h2>

              <p>
                O agente do Databasus é um binário Go leve que executa dois
                processos em paralelo:
              </p>

              <ul>
                <li>
                  <strong>Streaming de WAL</strong>: coleta os segmentos WAL do
                  diretório de fila a cada 10 segundos aproximadamente e os
                  envia para o Databasus
                </li>
                <li>
                  <strong>Backups de base periódicos</strong>: executa o{" "}
                  <code>pg_basebackup</code> no agendamento configurado para
                  criar backups físicos completos do cluster
                </li>
              </ul>

              <p>
                Durante a restauração, o agente baixa o backup de base e todos
                os segmentos WAL relevantes e depois configura o{" "}
                <code>recovery.signal</code> e o <code>restore_command</code> no{" "}
                <code>postgresql.auto.conf</code>. Quando o PostgreSQL inicia,
                ele reproduz os segmentos WAL até atingir o ponto de recuperação
                alvo.
              </p>

              <p>
                O agente sempre inicia a conexão com o Databasus (de saída). O
                servidor da base de dados não precisa aceitar conexões de
                entrada vindas do Databasus, o que o torna adequado para redes
                privadas e ambientes com firewall.
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
