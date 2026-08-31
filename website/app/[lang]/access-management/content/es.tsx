import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Gestión de acceso - Documentación de Databasus",
  description:
    "Aprenda a gestionar el acceso, los roles y los permisos en Databasus. Controle quién puede registrarse, crear espacios de trabajo y administrar bases de datos con roles a nivel de espacio de trabajo y de sistema.",
  keywords: [
    "gestión de acceso Databasus",
    "roles de usuario",
    "permisos de espacio de trabajo",
    "registros de auditoría",
    "seguridad de copias de seguridad de PostgreSQL",
    "colaboración en equipo",
    "control de acceso",
    "gestión de espacios de trabajo",
  ],
  openGraph: {
    title: "Gestión de acceso - Documentación de Databasus",
    description:
      "Aprenda a gestionar el acceso, los roles y los permisos en Databasus. Controle quién puede registrarse, crear espacios de trabajo y administrar bases de datos con roles a nivel de espacio de trabajo y de sistema.",
    type: "article",
    url: getLocalizedUrl("es", "access-management"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Gestión de acceso - Documentación de Databasus",
    description:
      "Aprenda a gestionar el acceso, los roles y los permisos en Databasus. Controle quién puede registrarse, crear espacios de trabajo y administrar bases de datos con roles a nivel de espacio de trabajo y de sistema.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "access-management"),
    languages: getLanguageAlternates("access-management"),
  },
  robots: "index, follow",
};

export default function AccessManagementPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Gestión de acceso - Documentación de Databasus",
            description:
              "Aprenda a gestionar el acceso, los roles y los permisos en Databasus. Controle quién puede registrarse, crear espacios de trabajo y administrar bases de datos con roles a nivel de espacio de trabajo y de sistema.",
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
              <h1 id="settings">Ajustes</h1>

              <p>
                Databasus sirve tanto para usuarios individuales como para
                equipos. Esta sección está dedicada a la gestión de acceso para
                equipos.{" "}
                <strong>
                  Si es el único usuario de su instancia de Databasus
                </strong>
                , puede saltarse esta sección.
              </p>

              <p>
                Databasus no tiene muchos ajustes. En realidad, solo permite
                controlar:
              </p>

              <ul>
                <li>Quién puede registrarse en su instancia de Databasus</li>
                <li>Quién puede crear espacios de trabajo</li>
                <li>
                  Quién puede administrar bases de datos, notificadores y
                  almacenamientos dentro de los espacios de trabajo
                </li>
              </ul>

              <h2 id="workspaces">Espacios de trabajo</h2>

              <p>
                Un espacio de trabajo es el lugar donde{" "}
                <strong>
                  se agrupan bases de datos, notificadores y almacenamientos
                </strong>
                . Puede añadir miembros a un espacio de trabajo y crear varios.
              </p>

              <p>
                La gestión de acceso se controla por espacio de trabajo. Por
                ejemplo:
              </p>

              <ul>
                <li>
                  tiene un equipo de DevOps responsable de 10 bases de datos del
                  proyecto (varios usuarios dentro de un espacio de trabajo);
                </li>
                <li>
                  tiene 3 proyectos distintos con bases de datos y
                  almacenamientos distintos (varios espacios de trabajo con
                  usuarios diferentes);
                </li>
                <li>
                  tiene 5 bases de datos independientes a las que acceden
                  usuarios distintos (el usuario A accede a la DB1, el usuario B
                  a la DB2, el usuario C a la DB3, etc.).
                </li>
              </ul>

              <img
                src="/images/access-management/users.png"
                alt="Espacios de trabajo"
                width={550}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                Si lo permite en los{" "}
                <a href="#global-settings">ajustes globales</a>, los usuarios
                podrán registrarse en su Databasus y crear sus propios espacios
                de trabajo.
              </p>

              <p>
                <strong>
                  Los usuarios nunca ven espacios de trabajo ajenos hasta que se
                  les invita a unirse.
                </strong>
              </p>

              <h2 id="audit-logs">Registros de auditoría</h2>

              <p>
                Los registros de auditoría son mensajes sobre las acciones
                realizadas por los usuarios. Sirven para rastrear los cambios y
                acciones de los usuarios y para detectar cualquier actividad
                sospechosa.
              </p>

              <p>Por ejemplo:</p>

              <ul>
                <li>un usuario creó una nueva base de datos</li>
                <li>un usuario eliminó una base de datos</li>
                <li>un usuario inició un nuevo respaldo</li>
                <li>un usuario descargó un respaldo</li>
                <li>un usuario creó un nuevo notificador</li>
                <li>un usuario creó un espacio de trabajo</li>
                <li>un usuario eliminó un espacio de trabajo</li>
                <li>etc.</li>
              </ul>

              <p>Puede ver los registros de auditoría con filtros:</p>

              <ul>
                <li>por espacio de trabajo;</li>
                <li>por usuario (a través de varios espacios de trabajo);</li>
              </ul>

              <img
                src="/images/access-management/audit-logs.png"
                alt="Registros de auditoría"
                width={1000}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="user-roles">Roles de usuario</h2>

              <p>
                Todos los usuarios de Databasus tienen roles{" "}
                <u>dentro del sistema</u>:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Función</th>
                    <th>Admin</th>
                    <th>Member</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Administrar todos los ajustes y usuarios</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">❌</td>
                  </tr>
                  <tr>
                    <td>Crear espacios de trabajo</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">✅ (si los ajustes lo permiten)</td>
                  </tr>
                </tbody>
              </table>

              <p>
                Normalmente hay un solo usuario <code>admin</code> en el
                sistema: el que se crea al iniciar Databasus por primera vez.
              </p>

              <p>
                <u>Dentro de un espacio de trabajo</u> también hay roles:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Función</th>
                    <th>Viewer</th>
                    <th>Member</th>
                    <th>Admin</th>
                    <th>Owner</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Ver bases de datos, notificadores, almacenamientos</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Iniciar y descargar respaldos</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>
                      Administrar bases de datos, notificadores, almacenamientos
                    </td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Administrar usuarios</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Administrar administradores</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">❌</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                </tbody>
              </table>

              <p>
                Tenga en cuenta: los <strong>datos sensibles</strong>{" "}
                (contraseñas, tokens, etc.) de las bases de datos,
                almacenamientos y notificadores{" "}
                <strong>siempre quedan ocultos para cualquier usuario</strong>.
                Nadie puede ver los secretos después de su creación.
              </p>

              <h2 id="global-settings">Ajustes globales</h2>

              <p>En los ajustes globales hay 3 propiedades:</p>

              <ol>
                <li>
                  <strong>Allow external registrations</strong> - de forma
                  predeterminada, cualquier usuario puede registrarse en su
                  Databasus (aunque no tendrá acceso a ningún espacio de trabajo
                  hasta que se le invite o cree los suyos propios).
                  <br />
                  <br />
                  Si quiere que solo los usuarios invitados puedan registrarse,
                  desactive esta opción. En ese caso, el formulario de registro
                  permanecerá cerrado hasta que invite al usuario a alguno de
                  los espacios de trabajo.
                  <br />
                  <br />
                  Para invitar usuarios al espacio de trabajo, haga clic en
                  &quot;Add user&quot; e introduzca un correo electrónico.
                  Después, el usuario con ese correo podrá completar el
                  registro.
                </li>
                <li>
                  <strong>Allow member invitations</strong> - este ajuste es
                  necesario cuando los registros externos están desactivados.
                  <br />
                  <br />
                  Imagine que ya tiene usuarios y sabe que son de confianza (por
                  ejemplo, su equipo). Quiere permitirles invitar a otros
                  usuarios a unirse a Databasus. En ese caso, active esta opción
                  y podrán invitar a otras personas a sus espacios de trabajo.
                  <br />
                  <br />
                  Si está desactivada, solo los administradores pueden invitar
                  usuarios.
                </li>
                <li>
                  <strong>Allow member workspace creation</strong> - de forma
                  predeterminada, todos los miembros pueden crear sus propios
                  espacios de trabajo. Si quiere que solo los administradores
                  puedan crearlos, desactive esta opción.
                </li>
              </ol>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
