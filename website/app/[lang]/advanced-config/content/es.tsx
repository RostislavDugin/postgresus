import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Configuración avanzada - Documentación de Databasus",
  description:
    "Variables de entorno opcionales para autoalojar Databasus: inicio de sesión con Google y GitHub, correo SMTP, captcha de Cloudflare Turnstile, telemetría, exportación de registros por OpenTelemetry y un script de analítica propio. No hacen falta en una instalación por defecto.",
  keywords: [
    "variables de entorno de Databasus",
    "configuración avanzada de Databasus",
    "configuración autoalojada",
    "GitHub OAuth",
    "Google OAuth",
    "configuración de correo SMTP",
    "Cloudflare Turnstile",
    "variables de entorno de Docker",
    "registros OpenTelemetry",
  ],
  openGraph: {
    title: "Configuración avanzada - Documentación de Databasus",
    description:
      "Variables de entorno opcionales para autoalojar Databasus: inicio de sesión con Google y GitHub, correo SMTP, captcha de Cloudflare Turnstile, telemetría, exportación de registros por OpenTelemetry y un script de analítica propio. No hacen falta en una instalación por defecto.",
    type: "article",
    url: getLocalizedUrl("es", "advanced-config"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Configuración avanzada - Documentación de Databasus",
    description:
      "Variables de entorno opcionales para autoalojar Databasus: inicio de sesión con Google y GitHub, correo SMTP, captcha de Cloudflare Turnstile, telemetría, exportación de registros por OpenTelemetry y un script de analítica propio. No hacen falta en una instalación por defecto.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "advanced-config"),
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
            headline: "Configuración avanzada - Documentación de Databasus",
            description:
              "Variables de entorno opcionales para autoalojar Databasus: inicio de sesión con Google y GitHub, correo SMTP, captcha de Cloudflare Turnstile, telemetría, exportación de registros por OpenTelemetry y un script de analítica propio. No hacen falta en una instalación por defecto.",
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
              <h1 id="advanced-config">Configuración avanzada</h1>

              <p className="text-lg text-gray-400">
                Databasus funciona con valores por defecto razonables desde el
                primer momento: una instalación estándar de un solo contenedor
                no necesita configuración alguna. Todas las variables de esta
                página son <strong>opcionales</strong> y no hacen falta en el
                99% de los despliegues en producción.
              </p>

              <h2 id="oauth">OAuth</h2>

              <p>
                Por defecto Databasus usa inicio de sesión con correo y
                contraseña. Además, puede permitir que la gente inicie sesión
                con su cuenta de Google o GitHub. El botón de un proveedor
                aparece en cuanto se define su client ID, pero el inicio de
                sesión solo se completa cuando están presentes{" "}
                <strong>tanto</strong> el client ID como el client secret.
              </p>

              <p>
                Al registrar la aplicación OAuth, establezca su URL de
                redirección (callback) en{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>. Debido a
                esa redirección, el inicio de sesión OAuth necesita que su
                instancia se sirva por HTTPS en un dominio público; vea la nota
                más abajo.
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
                      HTTPS es obligatorio para el inicio de sesión y el correo.
                    </strong>{" "}
                    Tanto el inicio de sesión OAuth como el correo necesitan que
                    su instancia sea accesible por HTTPS en un dominio público:
                    los proveedores OAuth redirigen el navegador de vuelta a{" "}
                    <code>https://&lt;your-domain&gt;/auth/callback</code>, y
                    los enlaces dentro de los correos deben abrirse para quien
                    los recibe. Una instancia solo en localhost o en HTTP plano
                    no puede usar estas funciones. La forma más simple de
                    obtener HTTPS es la configuración del{" "}
                    <a
                      href="/es/installation/#caddy-reverse-proxy"
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
                Cree un cliente OAuth en la{" "}
                <a
                  href="https://console.cloud.google.com/apis/credentials"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Google Cloud Console
                </a>{" "}
                (APIs &amp; Services → Credentials → Create credentials → OAuth
                client ID, tipo de aplicación <em>Web application</em>) y añada{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code> como URI
                de redirección autorizada.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID de su cliente OAuth de Google. Al definirlo se
                      muestra el botón &quot;Sign in with Google&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret de su cliente OAuth de Google. Requerido
                      junto con el ID para que el inicio de sesión funcione.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h3 id="oauth-github">GitHub</h3>

              <p>
                Cree una aplicación OAuth en{" "}
                <a
                  href="https://github.com/settings/developers"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  GitHub Developer settings
                </a>{" "}
                (Settings → Developer settings → OAuth Apps → New OAuth App) y
                establezca la authorization callback URL en{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID de su aplicación OAuth de GitHub. Al definirlo
                      se muestra el botón &quot;Sign in with GitHub&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret de su aplicación OAuth de GitHub. Requerido
                      junto con el ID para que el inicio de sesión funcione.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="email-smtp">Correo electrónico (SMTP)</h2>

              <p>
                Conecte un servidor SMTP para que Databasus pueda enviar correo
                transaccional, como enlaces de restablecimiento de contraseña e
                invitaciones a espacios de trabajo. El correo se considera
                configurado{" "}
                <strong>
                  solo cuando están definidos tanto <code>SMTP_HOST</code> como{" "}
                  <code>DATABASUS_URL</code>
                </strong>
                ; hasta entonces, las funciones de correo permanecen ocultas en
                la interfaz.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>SMTP_HOST</code>
                    </td>
                    <td data-label="Description">
                      Nombre de host del servidor SMTP (p. ej.{" "}
                      <code>smtp.gmail.com</code>). Activa el correo junto con{" "}
                      <code>DATABASUS_URL</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PORT</code>
                    </td>
                    <td data-label="Description">
                      Puerto del servidor SMTP (p. ej. <code>587</code>). Debe
                      ser un entero positivo cuando <code>SMTP_HOST</code> está
                      definido.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_USER</code>
                    </td>
                    <td data-label="Description">
                      Nombre de usuario para la autenticación SMTP.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PASSWORD</code>
                    </td>
                    <td data-label="Description">
                      Contraseña para la autenticación SMTP. Para Gmail, use una
                      App Password, no la contraseña de su cuenta.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_FROM</code>
                    </td>
                    <td data-label="Description">
                      La dirección &quot;From&quot; del correo saliente.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_INSECURE_SKIP_VERIFY</code>
                    </td>
                    <td data-label="Description">
                      Póngalo en <code>true</code> para omitir la verificación
                      del certificado TLS al conectar con el servidor SMTP. Por
                      defecto es <code>false</code>. Úselo solo con servidores
                      con certificado autofirmado en una red de confianza:
                      desactiva la protección contra ataques de intermediario
                      (man-in-the-middle).
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>DATABASUS_URL</code>
                    </td>
                    <td data-label="Description">
                      URL base pública de su instancia (p. ej.{" "}
                      <code>https://backup.example.com</code>). Se usa para
                      construir los enlaces dentro de los correos. Requerida
                      junto con <code>SMTP_HOST</code>.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="signup-captcha">
                Captcha de registro (Cloudflare Turnstile)
              </h2>

              <p>
                Si su instancia es accesible desde internet, puede poner un
                desafío de{" "}
                <a
                  href="https://www.cloudflare.com/products/turnstile/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Cloudflare Turnstile
                </a>{" "}
                en los formularios de registro e inicio de sesión para dejar
                fuera a los bots. Ambas claves salen del panel de Turnstile, y
                el desafío se activa solo cuando las dos están definidas.
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
                    Para bloquear por completo los registros externos, en lugar
                    de solo ponerles un desafío, no necesita captcha alguno:
                    abra <strong>Databasus settings → Allow sign up</strong> en
                    la interfaz y desactívelo. Eso cierra el formulario de
                    registro por completo.
                  </p>
                </div>
              </div>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SITE_KEY</code>
                    </td>
                    <td data-label="Description">
                      Clave pública de sitio de Turnstile, usada para renderizar
                      el widget en el navegador.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SECRET_KEY</code>
                    </td>
                    <td data-label="Description">
                      Clave secreta de Turnstile, usada por el backend para
                      validar las respuestas al desafío.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="docker-storage-permissions">Permisos del almacenamiento Docker</h2>

              <p>
                La mayoría de las instalaciones debe conservar los ID de la imagen.
                Cámbielos solo si un bind mount o un recurso CIFS o NFS exige un
                propietario numérico concreto. Se aceptan enteros decimales de{" "}
                <code>1</code> a <code>4294967294</code>; los valores vacíos se
                rechazan.
              </p>

              <table>
                <thead><tr><th>Variable</th><th>Valor predeterminado</th><th>Cuenta</th></tr></thead>
                <tbody>
                  <tr><td><code>DATABASUS_PUID</code></td><td><code>65532</code></td><td>Usuario de Databasus</td></tr>
                  <tr><td><code>DATABASUS_PGID</code></td><td><code>65532</code></td><td>Grupo principal de Databasus</td></tr>
                  <tr><td><code>POSTGRES_PUID</code></td><td><code>999</code></td><td>Usuario de PostgreSQL</td></tr>
                  <tr><td><code>POSTGRES_PGID</code></td><td><code>999</code></td><td>Grupo principal de PostgreSQL</td></tr>
                </tbody>
              </table>

              <p>
                Mantenga los mounts de la aplicación separados de pgdata. El
                entrypoint empieza como root dentro de su namespace, prepara
                ambas cuentas y reduce privilegios con <code>gosu</code>. Use
                Docker rootless o un user namespace para aislar el host; no se
                admiten valores arbitrarios de Docker <code>user:</code>.
              </p>

              <h2 id="telemetry">Telemetría</h2>

              <p>
                Databasus envía por defecto telemetría de uso anónima y no
                identificable. No contiene datos personales y nos ayuda a
                entender cómo se usa el proyecto. Puede leer exactamente qué se
                recopila en la{" "}
                <a
                  href="/privacy"
                  className="text-blue-400 hover:text-blue-300"
                >
                  política de privacidad
                </a>
                , y puede desactivarla por completo.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Por defecto</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>IS_DISABLE_ANONYMOUS_TELEMETRY</code>
                    </td>
                    <td data-label="Default">
                      <code>false</code>
                    </td>
                    <td data-label="Description">
                      Póngalo en <code>true</code> para desactivar la telemetría
                      de uso anónima.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="logging">Registros</h2>

              <p>
                Databasus escribe sus registros en stdout y los duplica como
                JSON en <code>databasus.log</code> en el volumen de datos.
                Defina <code>OPEN_TELEMETRY_URL</code> y también los exportará
                por OpenTelemetry a un backend como VictoriaLogs, Graylog,
                SigNoz, Grafana Loki, Datadog o Honeycomb, o a un OpenTelemetry
                Collector, que es a su vez un receptor OTLP.
              </p>

              <ul>
                <li>
                  <strong>El transporte</strong> sigue el esquema.{" "}
                  <code>http://</code> y <code>https://</code> envían OTLP/HTTP
                  y usan la URL literal, ruta incluida; <code>grpc://</code> y{" "}
                  <code>grpcs://</code> envían OTLP/gRPC y usan solo el host y
                  el puerto.
                </li>
                <li>
                  <strong>La autenticación</strong> va en{" "}
                  <code>OPEN_TELEMETRY_HEADERS</code> o en la URL como{" "}
                  <code>user:password@host</code>.
                </li>
                <li>
                  <strong>Los secretos</strong> (contraseñas, tokens,
                  credenciales) dentro de las URL se ocultan antes de que un
                  registro salga del proceso.
                </li>
                <li>
                  <strong>Las entradas de auditoría</strong> se envían con los
                  registros de la aplicación etiquetadas como{" "}
                  <code>log_type=audit</code> e ignoran <code>LOG_LEVEL</code>,
                  así que subir el nivel nunca elimina el rastro de auditoría.
                </li>
              </ul>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Por defecto</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_URL</code>
                    </td>
                    <td data-label="Default">—</td>
                    <td data-label="Description">
                      URL completa del endpoint OTLP, ruta incluida. Déjela sin
                      definir para mantener los registros en el contenedor. Una
                      query string, un host ausente o un esquema desconocido
                      detiene el contenedor en el arranque, en lugar de quedarse
                      sin exportar en silencio.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </td>
                    <td data-label="Default">—</td>
                    <td data-label="Description">
                      Pares <code>key=value</code> separados por comas que se
                      envían con cada exportación, normalmente una clave de API.
                      Los valores usan codificación porcentual, como en el
                      formato estándar de{" "}
                      <code>OTEL_EXPORTER_OTLP_HEADERS</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_LEVEL</code>
                    </td>
                    <td data-label="Default">
                      <code>info</code>
                    </td>
                    <td data-label="Description">
                      Uno de <code>debug</code>, <code>info</code>,{" "}
                      <code>warn</code> o <code>error</code>. Un valor no
                      reconocido vuelve a <code>info</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_FILE_IS_ENABLED</code>
                    </td>
                    <td data-label="Default">
                      <code>true</code>
                    </td>
                    <td data-label="Description">
                      Escribe <code>databasus.log</code> junto al resto de los
                      datos, con rotación a los 5 MB y conservando 3 archivos
                      antiguos. Póngalo en <code>false</code> si su plataforma
                      ya recolecta stdout.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Valores para los backends habituales, cada uno con la cabecera
                que lo autentica. Reemplace hosts, regiones y claves por los
                suyos:
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
                      <code>Authorization=Basic%20dXNlcjpwYXNzd29yZA==</code> —
                      las credenciales que espera su <code>vmauth</code> o
                      reverse proxy, ya que VictoriaLogs no tiene autenticación
                      propia en la ruta de ingesta.
                    </td>
                  </tr>
                  <tr>
                    <td>OpenTelemetry Collector</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://otel-collector:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> —
                      corresponde a una extensión <code>bearertokenauth</code> o{" "}
                      <code>basicauth</code> en el receptor. Un Collector
                      accesible solo dentro de su red normalmente no necesita
                      ninguna.
                    </td>
                  </tr>
                  <tr>
                    <td>Graylog 6.2+</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://graylog:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> — el token
                      definido en la entrada OpenTelemetry (gRPC). La entrada
                      también acepta mTLS en su lugar.
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
                      <code>Authorization=Basic%20&lt;base64&gt;</code> — base64
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
                      Ninguna: el Agent guarda la clave de API y reenvía los
                      registros en su nombre.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Los valores de las cabeceras usan codificación porcentual, así
                que el espacio después de <code>Basic</code> o{" "}
                <code>Bearer</code> se escribe como <code>%20</code> y una coma
                dentro de un valor como <code>%2C</code>. La autenticación
                básica también puede ir directamente en la URL como{" "}
                <code>https://user:password@host/path</code>: Databasus la
                convierte en la misma cabecera y la mantiene fuera de los
                registros. Por <code>http://</code> y <code>grpc://</code> las
                claves y contraseñas viajan en claro, así que use{" "}
                <code>https://</code> o <code>grpcs://</code> fuera de una red
                de confianza.
              </p>

              <h2 id="analytics-script">Script de analítica</h2>

              <p>
                Databasus puede inyectar en la aplicación su propio fragmento de
                analítica o seguimiento: Google Analytics, Plausible, Umami y
                similares. Cuando <code>ANALYTICS_SCRIPT</code> está definida,
                su valor se inserta en el <code>&lt;head&gt;</code> de la página
                al arrancar.
              </p>

              <p>
                <strong>Advertencia de seguridad:</strong> el valor se inyecta
                tal cual como HTML y JavaScript sin procesar, y se ejecuta con
                acceso completo a la interfaz de Databasus en el navegador de
                cada visitante. Establézcalo únicamente con un fragmento que
                controle y en el que confíe plenamente.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>ANALYTICS_SCRIPT</code>
                    </td>
                    <td data-label="Description">
                      Marcado <code>&lt;script&gt;</code> personalizado que se
                      inyecta antes de la etiqueta de cierre{" "}
                      <code>&lt;/head&gt;</code>. Déjela sin definir para no
                      añadir analítica.
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
