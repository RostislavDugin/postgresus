import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Cómo hacer copia de seguridad de Supabase con Databasus | Databasus",
  description:
    "Aprenda a hacer copias de seguridad de su base de datos PostgreSQL de Supabase con Databasus. Guía paso a paso para configurar el session pooler o una dirección IPv4 para los respaldos de Supabase.",
  keywords: [
    "Databasus",
    "copia de seguridad de Supabase",
    "copia de seguridad de PostgreSQL de Supabase",
    "respaldar base de datos de Supabase",
    "session pooler de Supabase",
    "IPv4 de Supabase",
    "copia de seguridad de PostgreSQL",
    "copia de seguridad de base de datos",
  ],
  openGraph: {
    title:
      "Cómo hacer copia de seguridad de Supabase con Databasus | Databasus",
    description:
      "Aprenda a hacer copias de seguridad de su base de datos PostgreSQL de Supabase con Databasus. Guía paso a paso para configurar el session pooler o una dirección IPv4 para los respaldos de Supabase.",
    type: "article",
    url: getLocalizedUrl("es", "faq/supabase"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title:
      "Cómo hacer copia de seguridad de Supabase con Databasus | Databasus",
    description:
      "Aprenda a hacer copias de seguridad de su base de datos PostgreSQL de Supabase con Databasus. Guía paso a paso para configurar el session pooler o una dirección IPv4 para los respaldos de Supabase.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "faq/supabase"),
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
            name: "Cómo hacer copia de seguridad de Supabase con Databasus",
            description:
              "Guía paso a paso para hacer copias de seguridad de su base de datos PostgreSQL de Supabase con Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Obtenga los datos de conexión de Supabase",
                text: "Vaya a los ajustes de su proyecto de Supabase y localice los datos de conexión de la base de datos.",
              },
              {
                "@type": "HowToStep",
                name: "Use el Session Pooler con IPv4",
                text: "Copie la cadena de conexión del Session Pooler y compruebe que 'Use IPv4 Address' está habilitado.",
              },
              {
                "@type": "HowToStep",
                name: "Configure Databasus",
                text: "Introduzca los datos de conexión de Supabase en Databasus para empezar a respaldar su base de datos.",
              },
              {
                "@type": "HowToStep",
                name: "Conozca las limitaciones de esquemas",
                text: "De forma predeterminada solo se respalda el esquema public, ya que los demás esquemas de Supabase están restringidos.",
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
              <h1 id="supabase-backup">
                Cómo hacer copia de seguridad de Supabase
              </h1>

              <p className="text-lg text-gray-400">
                Databasus admite copias de seguridad de bases de datos
                PostgreSQL de Supabase. El requisito principal es usar una
                dirección IPv4 para conectarse a su instancia de Supabase.
              </p>

              <h2 id="connection-options">Opciones de conexión</h2>

              <p>
                Hay dos formas de conectar Databasus a su base de datos de
                Supabase:
              </p>

              <ol>
                <li>
                  <strong>Usar el Session Pooler con IPv4</strong>{" "}
                  (recomendado): opción gratuita disponible en todos los
                  proyectos de Supabase
                </li>
                <li>
                  <strong>Comprar el complemento IPv4</strong>: opción de
                  conexión directa de Supabase
                </li>
              </ol>

              <h2 id="session-pooler">
                Opción 1: usar el Session Pooler (recomendado)
              </h2>

              <p>
                El Session Pooler proporciona una dirección IPv4 para la
                conexión a su base de datos de Supabase sin coste adicional. Así
                se configura:
              </p>

              <h3 id="step-1">1. Localice la conexión del pooler</h3>

              <p>
                Vaya a su proyecto de Supabase, entre en{" "}
                <strong>Project Settings</strong> → <strong>Database</strong>.
                Baje hasta la sección <strong>Connection string</strong> y
                seleccione el modo <strong>Session pooler</strong>.
              </p>

              <img
                src="/images/faq/supabase/image-1.png"
                alt="Seleccionar el modo Session pooler en Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h3 id="step-2">2. Copie los datos de conexión</h3>

              <p>
                Copie los datos de conexión y úselos en Databasus al añadir su
                base de datos. Vea la captura para distinguir cada dato de
                conexión.
              </p>

              <img
                src="/images/faq/supabase/image-2.png"
                alt="Habilitar la opción IPv4 Address en Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h2 id="ipv4-addon">Opción 2: comprar el complemento IPv4</h2>

              <p>
                Supabase ofrece un complemento IPv4 de pago que proporciona una
                dirección IPv4 dedicada para su base de datos. Esta opción le da
                una conexión directa sin pasar por el connection pooler.
              </p>

              <p>Para habilitar esta opción:</p>

              <ol>
                <li>Vaya al panel de su proyecto de Supabase</li>
                <li>
                  Entre en <strong>Project Settings</strong> →{" "}
                  <strong>Add-ons</strong>
                </li>
                <li>
                  Habilite el complemento <strong>IPv4</strong>
                </li>
                <li>
                  Use los datos de conexión directa a la base de datos en
                  Databasus
                </li>
              </ol>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Consejo:</strong> en la
                  mayoría de los casos, la opción gratuita del Session Pooler
                  con IPv4 funciona perfectamente para los respaldos. El
                  complemento IPv4 de pago solo hace falta si necesita una
                  conexión directa por otros motivos.
                </p>
              </div>

              <h2 id="default-schema">Limitación del esquema predeterminado</h2>

              <p>
                De forma predeterminada, Databasus respalda solo el esquema{" "}
                <code>public</code> cuando trabaja con bases de datos de
                Supabase. Esto se debe a que Supabase restringe el acceso a los
                demás esquemas (como <code>auth</code>, <code>storage</code> y{" "}
                <code>realtime</code>) por motivos de seguridad.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-blue-400">ℹ️ Nota:</strong> el esquema{" "}
                  <code>public</code> contiene los datos de su aplicación y sus
                  tablas personalizadas. Los esquemas gestionados por Supabase,
                  como <code>auth</code> y <code>storage</code>, están
                  protegidos y los administra el propio Supabase.
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
