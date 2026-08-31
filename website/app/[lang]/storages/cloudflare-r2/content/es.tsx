import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "Cómo usar Databasus con Cloudflare R2 | Databasus",
  description:
    "Guía paso a paso para configurar Cloudflare R2 como almacenamiento de las copias de seguridad de PostgreSQL con Databasus. Aprenda a configurar un almacenamiento compatible con S3 usando R2.",
  keywords: [
    "Databasus",
    "Cloudflare R2",
    "copia de seguridad de PostgreSQL",
    "almacenamiento S3",
    "almacenamiento en la nube",
    "copia de seguridad de base de datos",
  ],
  openGraph: {
    title: "Cómo usar Databasus con Cloudflare R2 | Databasus",
    description:
      "Guía paso a paso para configurar Cloudflare R2 como almacenamiento de las copias de seguridad de PostgreSQL con Databasus. Aprenda a configurar un almacenamiento compatible con S3 usando R2.",
    type: "article",
    url: getLocalizedUrl("es", "storages/cloudflare-r2"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Cómo usar Databasus con Cloudflare R2 | Databasus",
    description:
      "Guía paso a paso para configurar Cloudflare R2 como almacenamiento de las copias de seguridad de PostgreSQL con Databasus. Aprenda a configurar un almacenamiento compatible con S3 usando R2.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "storages/cloudflare-r2"),
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
            name: "Cómo usar Databasus con Cloudflare R2",
            description:
              "Guía paso a paso para configurar Cloudflare R2 como almacenamiento de las copias de seguridad de PostgreSQL con Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Indique el nombre del bucket",
                text: "Introduzca el nombre de su bucket de R2 en la configuración del almacenamiento.",
              },
              {
                "@type": "HowToStep",
                name: "Defina la región",
                text: 'En el campo de región, escriba "auto".',
              },
              {
                "@type": "HowToStep",
                name: "Genere un access key ID y un secret access key",
                text: "En el panel de Cloudflare, vaya a R2 → API → Manage API Tokens. Cree el token y concédale los permisos que necesite.",
              },
              {
                "@type": "HowToStep",
                name: "Localice su account ID",
                text: "En cualquier página de R2 del panel verá su Account ID cerca de la parte superior.",
              },
              {
                "@type": "HowToStep",
                name: "Construya el endpoint de S3",
                text: "Sustituya <ACCOUNT_ID> por el valor de su panel en el formato: https://<ACCOUNT_ID>.r2.cloudflarestorage.com",
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
              <h1 id="cloudflare-r2">Almacenamiento en Cloudflare R2</h1>

              <p className="text-lg text-gray-400">
                Para usar Cloudflare R2 como almacenamiento compatible con S3
                para sus copias de seguridad de PostgreSQL, debe configurar las
                credenciales y el endpoint de su bucket de R2.
              </p>

              <h2 id="configuration-steps">Pasos de configuración</h2>

              <h3 id="fill-bucket-name">1. Indique el nombre del bucket</h3>

              <p>
                Introduzca el nombre de su bucket de R2 en la configuración del
                almacenamiento:
              </p>

              <Image
                src="/images/cloudflare-r2-storage/image-1.webp"
                alt="Indicar el nombre del bucket en Cloudflare R2"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="set-region">2. Defina la región</h3>

              <p>
                En el campo de región, escriba <code>&quot;auto&quot;</code>.
              </p>

              <h3 id="generate-access-key">
                3. Genere un Access Key ID y un Secret Access Key
              </h3>

              <p>
                En el panel de Cloudflare, vaya a{" "}
                <strong>R2 → API → Manage API Tokens</strong>. Cree un nuevo
                token y concédale los permisos que necesite (por ejemplo,{" "}
                <strong>&quot;Object Read &amp; Write&quot;</strong>).
              </p>

              <p>Cuando el token esté creado, verá:</p>

              <ul>
                <li>
                  <strong>Access Key ID</strong> (el ID del token)
                </li>
                <li>
                  <strong>Secret Access Key</strong> (el hash SHA-256 del valor
                  del token)
                </li>
              </ul>

              <p>Copie ambos valores en Databasus:</p>

              <Image
                src="/images/cloudflare-r2-storage/image-2.gif"
                alt="Generar el Access Key ID y el Secret Access Key"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="find-account-id">4. Localice su account ID</h3>

              <p>
                En cualquier página de R2 del panel verá su Account ID cerca de
                la parte superior (o en los ajustes de su cuenta):
              </p>

              <Image
                src="/images/cloudflare-r2-storage/image-3.webp"
                alt="Localizar el Account ID en el panel de Cloudflare"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="construct-endpoint">5. Construya el endpoint de S3</h3>

              <p>Use el siguiente formato para su endpoint de S3:</p>

              <pre>
                <code>https://&lt;ACCOUNT_ID&gt;.r2.cloudflarestorage.com</code>
              </pre>

              <p>
                Sustituya <code>&lt;ACCOUNT_ID&gt;</code> por el valor de su
                panel e introdúzcalo en Databasus.
              </p>

              <p>¡Eso es todo! Su configuración debería verse así:</p>

              <Image
                src="/images/cloudflare-r2-storage/image-4.png"
                alt="Configuración completada"
                width={500}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                Databasus ya está listo para usar Cloudflare R2 como
                almacenamiento de sus copias de seguridad de PostgreSQL.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/es/storages"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Volver a almacenamientos
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
