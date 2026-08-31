import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "Cómo conectar Google Drive a Databasus | Databasus",
  description:
    "Guía paso a paso para configurar Google Drive como almacenamiento de las copias de seguridad de PostgreSQL con Databasus. Aprenda a crear el proyecto de Google Cloud y a configurar OAuth.",
  keywords: [
    "Databasus",
    "Google Drive",
    "copia de seguridad de PostgreSQL",
    "Google Cloud",
    "OAuth",
    "almacenamiento en la nube",
    "copia de seguridad de base de datos",
  ],
  openGraph: {
    title: "Cómo conectar Google Drive a Databasus | Databasus",
    description:
      "Guía paso a paso para configurar Google Drive como almacenamiento de las copias de seguridad de PostgreSQL con Databasus. Aprenda a crear el proyecto de Google Cloud y a configurar OAuth.",
    type: "article",
    url: getLocalizedUrl("es", "storages/google-drive"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Cómo conectar Google Drive a Databasus | Databasus",
    description:
      "Guía paso a paso para configurar Google Drive como almacenamiento de las copias de seguridad de PostgreSQL con Databasus. Aprenda a crear el proyecto de Google Cloud y a configurar OAuth.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "storages/google-drive"),
    languages: getLanguageAlternates("storages/google-drive"),
  },
  robots: "index, follow",
};

export default function GoogleDrivePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cómo conectar Google Drive a Databasus",
            description:
              "Guía paso a paso para configurar Google Drive como almacenamiento de las copias de seguridad de PostgreSQL con Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Cree un nuevo proyecto",
                text: "Vaya a la consola de Google Cloud y cree un nuevo proyecto.",
              },
              {
                "@type": "HowToStep",
                name: "Habilite la Google Drive API",
                text: "Vaya a la pestaña API & Services, luego a la biblioteca de APIs y habilite la Google Drive API.",
              },
              {
                "@type": "HowToStep",
                name: "Configure la pantalla de consentimiento",
                text: "Vaya a Credentials → Create credentials → Configure consent screen y complete los datos requeridos.",
              },
              {
                "@type": "HowToStep",
                name: "Cree el OAuth client ID",
                text: "Vaya a Credentials → Create credentials → OAuth client ID.",
              },
              {
                "@type": "HowToStep",
                name: "Configure los ajustes de la aplicación",
                text: "Elija el tipo de aplicación Web application y configure los orígenes autorizados y las URIs de redirección.",
              },
              {
                "@type": "HowToStep",
                name: "Añada el scope",
                text: 'Vaya a Data Access y añada el scope "/auth/drive.file".',
              },
              {
                "@type": "HowToStep",
                name: "Publique la app",
                text: "Vaya a Audience y publique la app.",
              },
              {
                "@type": "HowToStep",
                name: "Inicie sesión con su cuenta de Google",
                text: "Complete los datos de las credenciales e inicie sesión con su cuenta de Google.",
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
              <h1 id="google-drive">Almacenamiento en Google Drive</h1>

              <p className="text-lg text-gray-400">
                Para guardar sus respaldos en Google Drive, debe crear un
                proyecto de Google Cloud para acceder a la Google Drive API y
                luego iniciar sesión con su cuenta de Google.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Importante:</strong> para
                  conectar Google Drive, su instancia de Databasus debe
                  funcionar bajo HTTPS. Para más información sobre cómo
                  configurar HTTPS, consulte la{" "}
                  <a
                    href="/es/installation#caddy-reverse-proxy"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    guía de configuración de Caddy como reverse proxy →
                  </a>
                  <br />
                  En esta guía suponemos que su instancia de Databasus funciona
                  en <code>databasus.yourdomain.com</code>.
                </p>
              </div>

              <h2 id="create-google-cloud-project">
                Cree el proyecto de Google Cloud
              </h2>

              <h3 id="create-new-project">1. Cree un nuevo proyecto</h3>

              <p>
                Vaya a{" "}
                <a
                  href="https://console.cloud.google.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://console.cloud.google.com/
                </a>{" "}
                y elija <strong>&quot;new project&quot;</strong> (arriba a la
                izquierda).
              </p>

              <h3 id="enable-google-drive-api">
                2. Habilite la Google Drive API
              </h3>

              <p>
                Vaya a la pestaña{" "}
                <strong>&quot;API &amp; Services&quot;</strong> y luego a{" "}
                <strong>&quot;API library&quot;</strong>. Elija{" "}
                <strong>Google Drive API</strong> y habilítela:
              </p>

              <Image
                src="/images/google-drive-storage/image-1.webp"
                alt="Habilitar la Google Drive API"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-consent-screen">
                3. Configure la pantalla de consentimiento
              </h3>

              <p>
                Vaya a <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;Configure consent screen&quot;</strong> y complete
                los datos:
              </p>

              <Image
                src="/images/google-drive-storage/image-2.webp"
                alt="Configurar la pantalla de consentimiento"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-oauth-client-id">4. Cree el OAuth client ID</h3>

              <p>
                Vaya a <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;OAuth client ID&quot;</strong>:
              </p>

              <Image
                src="/images/google-drive-storage/image-3.webp"
                alt="Crear el OAuth client ID"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-application-settings">
                5. Configure los ajustes de la aplicación
              </h3>

              <p>Complete los datos siguientes:</p>

              <ul>
                <li>
                  <strong>Application type:</strong> Web application
                </li>
                <li>
                  <strong>Authorized JavaScript origins:</strong>{" "}
                  <code>https://databasus.yourdomain.com</code>
                </li>
                <li>
                  <strong>Authorized redirect URIs:</strong>{" "}
                  <code>
                    https://databasus.yourdomain.com/storages/google-oauth
                  </code>
                </li>
              </ul>

              <p>
                <strong>Nota:</strong> sustituya{" "}
                <code>databasus.yourdomain.com</code> por el dominio real donde
                está alojado Databasus (por ejemplo,{" "}
                <code>backup.example.com</code>).
              </p>

              <p>Luego copie las credenciales:</p>

              <Image
                src="/images/google-drive-storage/image-4.png"
                alt="Configurar los ajustes de la aplicación - parte 1"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <Image
                src="/images/google-drive-storage/image-5.png"
                alt="Configurar los ajustes de la aplicación - parte 2"
                width={450}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="add-scope">6. Añada el scope</h3>

              <p>
                Vaya a <strong>&quot;Data Access&quot;</strong> y añada el scope{" "}
                <code>&quot;/auth/drive.file&quot;</code>:
              </p>

              <Image
                src="/images/google-drive-storage/image-6.png"
                alt="Añadir el scope"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="publish-app">7. Publique la app</h3>

              <p>
                Vaya a <strong>&quot;Audience&quot;</strong> y publique la app:
              </p>

              <Image
                src="/images/google-drive-storage/image-7.png"
                alt="Publicar la app"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="sign-in-google-account">
                Inicie sesión con su cuenta de Google
              </h2>

              <h3 id="fill-credentials">1. Complete las credenciales</h3>

              <p>
                Introduzca en Databasus las credenciales de los pasos
                anteriores:
              </p>

              <Image
                src="/images/google-drive-storage/image-8.png"
                alt="Completar las credenciales"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="choose-account">2. Elija su cuenta</h3>

              <p>Elija su cuenta de Google para iniciar sesión.</p>

              <h3 id="handle-security-warning">
                3. Gestione la advertencia de seguridad
              </h3>

              <p>
                Si ve una advertencia, haga clic en{" "}
                <strong>&quot;Advanced&quot;</strong> (esquina inferior
                izquierda) y elija <strong>&quot;Proceed anyway&quot;</strong>.
              </p>

              <p>
                <strong>Nota:</strong> esta advertencia aparece porque Google
                aún no ha verificado su app. Como la aplicación es suya, puede
                continuar sin riesgo.
              </p>

              <p>
                ¡Eso es todo! Su Google Drive ya está conectado a Databasus y
                listo para guardar sus copias de seguridad de PostgreSQL.
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
