import type { Metadata } from "next";
import {
  OG_LOCALES,
  getLanguageAlternates,
  getLocalizedUrl,
} from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "Cómo hacer copia de seguridad de bases de datos en localhost | Databasus",
  description:
    "Aprenda a hacer copias de seguridad de bases de datos PostgreSQL que se ejecutan en localhost con Databasus. Configure el modo de red host de Docker para respaldar bases de datos locales.",
  keywords: [
    "Databasus",
    "copia de seguridad en localhost",
    "copia de seguridad de PostgreSQL local",
    "respaldar base de datos local",
    "red host de Docker",
    "copia de seguridad de PostgreSQL",
    "copia de seguridad de base de datos",
    "base de datos en localhost",
  ],
  openGraph: {
    title:
      "Cómo hacer copia de seguridad de bases de datos en localhost | Databasus",
    description:
      "Aprenda a hacer copias de seguridad de bases de datos PostgreSQL que se ejecutan en localhost con Databasus. Configure el modo de red host de Docker para respaldar bases de datos locales.",
    type: "article",
    url: getLocalizedUrl("es", "faq/localhost"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title:
      "Cómo hacer copia de seguridad de bases de datos en localhost | Databasus",
    description:
      "Aprenda a hacer copias de seguridad de bases de datos PostgreSQL que se ejecutan en localhost con Databasus. Configure el modo de red host de Docker para respaldar bases de datos locales.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "faq/localhost"),
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
            name: "Cómo hacer copia de seguridad de bases de datos en localhost con Databasus",
            description:
              "Guía paso a paso para hacer copias de seguridad de bases de datos PostgreSQL que se ejecutan en localhost con Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Configure el modo de red host de Docker",
                text: "Actualice su configuración de Docker para usar el modo de red host y que el contenedor pueda acceder a los servicios en localhost.",
              },
              {
                "@type": "HowToStep",
                name: "Use Docker Compose o docker run",
                text: "Aplique el ajuste network_mode: host en Docker Compose o use la opción --network host con docker run.",
              },
              {
                "@type": "HowToStep",
                name: "Conéctese a la base de datos en localhost",
                text: "Use 127.0.0.1 o localhost como host de la base de datos en la configuración de respaldo de Databasus.",
              },
            ],
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
              <h1 id="localhost-backup">
                Cómo hacer copia de seguridad de bases de datos en localhost
              </h1>

              <p className="text-lg text-gray-400">
                Aprenda a configurar Databasus para respaldar bases de datos
                PostgreSQL que se ejecutan en su máquina anfitriona (localhost)
                cuando usa Docker.
              </p>

              <h2 id="the-problem">El problema</h2>

              <p>
                Si ejecuta Databasus en Docker y quiere respaldar bases de datos
                que corren en su máquina anfitriona (localhost), debe configurar
                Docker para usar el <strong>modo de red host</strong>.
              </p>

              <p>
                De forma predeterminada, los contenedores de Docker se ejecutan
                en una red aislada y no pueden acceder a los servicios en{" "}
                <code>localhost</code>. El modo de red host permite que el
                contenedor comparta el espacio de nombres de red del anfitrión.
              </p>

              <h2 id="docker-compose-solution">Solución para Docker Compose</h2>

              <p>
                Actualice su archivo <code>docker-compose.yml</code> para usar{" "}
                <code>network_mode: host</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={dockerComposeHost} />
                </div>
              </div>

              <h2 id="docker-run-solution">Solución para docker run</h2>

              <p>
                Use la opción <code>--network host</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRunHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={dockerRunHost} />
                </div>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Nota:</strong> con el
                  modo de red host puede conectarse a su base de datos en
                  localhost usando{" "}
                  <code className="bg-[#374151] text-gray-200">127.0.0.1</code>{" "}
                  o{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>{" "}
                  como host en la configuración de respaldo de Databasus.
                  También accederá a la interfaz de Databasus directamente en{" "}
                  <code className="bg-[#374151] text-gray-200">
                    http://localhost:4005
                  </code>{" "}
                  sin mapeo de puertos.
                </p>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">
                    ⚠️ Importante para usuarios de Windows y macOS:
                  </strong>{" "}
                  el modo de red{" "}
                  <code className="bg-[#374151] text-red-400">host</code> solo
                  funciona de forma nativa en Linux. En Windows y macOS, Docker
                  se ejecuta dentro de una VM de Linux, así que debe usar{" "}
                  <code className="bg-[#374151] text-gray-200">
                    host.docker.internal
                  </code>{" "}
                  en lugar de{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>{" "}
                  como dirección del host de la base de datos en la
                  configuración de respaldo.
                </p>
              </div>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/es/faq"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Volver a las preguntas frecuentes
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
