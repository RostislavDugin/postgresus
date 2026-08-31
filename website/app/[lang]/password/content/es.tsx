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
  title: "Restablecer contraseña - Documentación de Databasus",
  description:
    "Aprenda a restablecer las contraseñas de usuario en Databasus con la herramienta de línea de comandos integrada. Recuperación de contraseña rápida y segura para su sistema de copias de seguridad de PostgreSQL.",
  keywords: [
    "restablecer contraseña Databasus",
    "restablecer contraseña de usuario",
    "contraseña de copia de seguridad de PostgreSQL",
    "recuperación de contraseña en Docker",
    "recuperación de contraseña",
    "autenticación de Databasus",
  ],
  openGraph: {
    title: "Restablecer contraseña - Documentación de Databasus",
    description:
      "Aprenda a restablecer las contraseñas de usuario en Databasus con la herramienta de línea de comandos integrada. Recuperación de contraseña rápida y segura para su sistema de copias de seguridad de PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("es", "password"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Restablecer contraseña - Documentación de Databasus",
    description:
      "Aprenda a restablecer las contraseñas de usuario en Databasus con la herramienta de línea de comandos integrada. Recuperación de contraseña rápida y segura para su sistema de copias de seguridad de PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "password"),
    languages: getLanguageAlternates("password"),
  },
  robots: "index, follow",
};

export default function PasswordResetPage() {
  const resetPasswordCommand = `docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="admin"`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Restablecer contraseña - Documentación de Databasus",
            description:
              "Aprenda a restablecer las contraseñas de usuario en Databasus con la herramienta de línea de comandos integrada.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cómo restablecer la contraseña de un usuario de Databasus",
            description:
              "Guía paso a paso para restablecer las contraseñas de usuario en Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Ejecute el comando de restablecimiento de contraseña",
                text: "Ejecute el comando docker exec con la nueva contraseña y el correo electrónico del usuario.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Use docker exec para ejecutar el comando de restablecimiento de contraseña dentro del contenedor de Databasus",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Verifique el cambio de contraseña",
                text: "Inicie sesión en Databasus con la nueva contraseña para confirmar que el cambio se aplicó correctamente.",
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
              <h1 id="reset-password">Restablecer la contraseña de usuario</h1>

              <h2 id="reset-password-command">
                Comando de restablecimiento de contraseña
              </h2>

              <p>
                Para restablecer la contraseña de un usuario, use el siguiente
                comando en el servidor donde se ejecuta Databasus:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={resetPasswordCommand} />
                </div>
              </div>

              <h2 id="parameters">Parámetros</h2>

              <p>El comando acepta los siguientes parámetros:</p>

              <ul>
                <li>
                  <strong>--new-password</strong>: la nueva contraseña.
                  Asegúrese de que sea segura y combine letras, números y
                  caracteres especiales.
                </li>
                <li>
                  <strong>--email</strong>: el correo electrónico del usuario
                  cuya contraseña quiere restablecer (por ejemplo,{" "}
                  <code>admin</code>, <code>user@example.com</code>).
                </li>
              </ul>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
