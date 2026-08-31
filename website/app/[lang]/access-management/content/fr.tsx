import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Gestion des accès - Documentation Databasus",
  description:
    "Apprenez à gérer les accès, les rôles et les permissions dans Databasus. Contrôlez qui peut s'inscrire, créer des espaces de travail et gérer les bases de données avec des rôles au niveau de l'espace de travail et du système.",
  keywords: [
    "gestion des accès Databasus",
    "rôles utilisateurs",
    "permissions d'espace de travail",
    "journaux d'audit",
    "sécurité des sauvegardes PostgreSQL",
    "collaboration en équipe",
    "contrôle d'accès",
    "gestion des espaces de travail",
  ],
  openGraph: {
    title: "Gestion des accès - Documentation Databasus",
    description:
      "Apprenez à gérer les accès, les rôles et les permissions dans Databasus. Contrôlez qui peut s'inscrire, créer des espaces de travail et gérer les bases de données avec des rôles au niveau de l'espace de travail et du système.",
    type: "article",
    url: getLocalizedUrl("fr", "access-management"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Gestion des accès - Documentation Databasus",
    description:
      "Apprenez à gérer les accès, les rôles et les permissions dans Databasus. Contrôlez qui peut s'inscrire, créer des espaces de travail et gérer les bases de données avec des rôles au niveau de l'espace de travail et du système.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "access-management"),
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
            headline: "Gestion des accès - Documentation Databasus",
            description:
              "Apprenez à gérer les accès, les rôles et les permissions dans Databasus. Contrôlez qui peut s'inscrire, créer des espaces de travail et gérer les bases de données avec des rôles au niveau de l'espace de travail et du système.",
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
              <h1 id="settings">Paramètres</h1>

              <p>
                Databasus convient aussi bien aux utilisateurs seuls qu&apos;aux
                équipes. Cette section est consacrée à la gestion des accès pour
                les équipes.{" "}
                <strong>
                  Si vous êtes le seul utilisateur de votre instance Databasus
                </strong>
                , vous pouvez ignorer cette section.
              </p>

              <p>
                Databasus n&apos;a pas beaucoup de paramètres. En réalité, il
                vous permet uniquement de contrôler :
              </p>

              <ul>
                <li>Qui peut s&apos;inscrire sur votre instance Databasus</li>
                <li>Qui peut créer des espaces de travail</li>
                <li>
                  Qui peut gérer les bases de données, les notificateurs et les
                  stockages au sein des espaces de travail
                </li>
              </ul>

              <h2 id="workspaces">Espaces de travail</h2>

              <p>
                Un espace de travail est l&apos;endroit où vous{" "}
                <strong>
                  regroupez bases de données, notificateurs et stockages
                </strong>
                . Vous pouvez y ajouter des membres (et créer plusieurs espaces
                de travail).
              </p>

              <p>
                Vous pouvez gérer les accès par espace de travail. Par exemple :
              </p>

              <ul>
                <li>
                  vous avez une équipe DevOps responsable des 10 bases du projet
                  (donc quelques utilisateurs dans un espace de travail) ;
                </li>
                <li>
                  vous avez 3 projets différents avec des bases et des stockages
                  différents (donc plusieurs espaces de travail avec des
                  utilisateurs différents) ;
                </li>
                <li>
                  vous avez 5 bases indépendantes accessibles chacune par des
                  utilisateurs différents (l&apos;utilisateur A a accès à la
                  base 1, l&apos;utilisateur B à la base 2, l&apos;utilisateur C
                  à la base 3, etc.).
                </li>
              </ul>

              <img
                src="/images/access-management/users.png"
                alt="Workspaces"
                width={550}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                Si vous autorisez les utilisateurs à s&apos;inscrire sur votre
                Databasus (voir les{" "}
                <a href="#global-settings">paramètres globaux</a>), ils pourront
                créer leurs propres espaces de travail.
              </p>

              <p>
                <strong>
                  Les utilisateurs ne voient jamais d&apos;autres espaces de
                  travail que les leurs tant qu&apos;ils n&apos;y sont pas
                  invités.
                </strong>
              </p>

              <h2 id="audit-logs">Journaux d&apos;audit</h2>

              <p>
                Les journaux d&apos;audit enregistrent les actions des
                utilisateurs. Ils permettent de suivre les modifications et de
                détecter toute activité suspecte.
              </p>

              <p>Par exemple :</p>

              <ul>
                <li>un utilisateur a créé une nouvelle base de données</li>
                <li>un utilisateur a supprimé une base de données</li>
                <li>un utilisateur a lancé une nouvelle sauvegarde</li>
                <li>un utilisateur a téléchargé une sauvegarde</li>
                <li>un utilisateur a créé un nouveau notificateur</li>
                <li>un utilisateur a créé un espace de travail</li>
                <li>un utilisateur a supprimé un espace de travail</li>
                <li>etc.</li>
              </ul>

              <p>
                Vous pouvez consulter les journaux d&apos;audit avec des filtres
                :
              </p>

              <ul>
                <li>par espace de travail ;</li>
                <li>
                  par utilisateur (à travers plusieurs espaces de travail) ;
                </li>
              </ul>

              <img
                src="/images/access-management/audit-logs.png"
                alt="Audit logs"
                width={1000}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="user-roles">Rôles utilisateurs</h2>

              <p>
                Tous les utilisateurs de Databasus ont des rôles{" "}
                <u>au niveau du système</u> :
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Fonctionnalité</th>
                    <th>Admin</th>
                    <th>Member</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Gérer tous les paramètres et utilisateurs</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">❌</td>
                  </tr>
                  <tr>
                    <td>Créer des espaces de travail</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">
                      ✅ (si autorisé par les paramètres)
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                En général, il n&apos;y a qu&apos;un seul utilisateur{" "}
                <code>admin</code> dans le système, celui que vous créez au
                premier lancement de Databasus.
              </p>

              <p>
                <u>Au sein d&apos;un espace de travail</u>, il y a également des
                rôles :
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Fonctionnalité</th>
                    <th>Viewer</th>
                    <th>Member</th>
                    <th>Admin</th>
                    <th>Owner</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Voir les bases de données, notificateurs, stockages</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Lancer et télécharger des sauvegardes</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>
                      Gérer les bases de données, notificateurs, stockages
                    </td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Gérer les utilisateurs</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Gérer les admins</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">❌</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                </tbody>
              </table>

              <p>
                Gardez à l&apos;esprit : les <strong>données sensibles</strong>{" "}
                (mots de passe, tokens, etc.) des bases, stockages et
                notificateurs{" "}
                <strong>
                  sont toujours masquées pour tous les utilisateurs
                </strong>
                . Personne ne peut voir les secrets après leur création.
              </p>

              <h2 id="global-settings">Paramètres globaux</h2>

              <p>Les paramètres globaux comportent 3 propriétés :</p>

              <ol>
                <li>
                  <strong>Allow external registrations</strong> - par défaut,
                  tous les utilisateurs peuvent s&apos;inscrire sur votre
                  Databasus (mais ils n&apos;ont accès à aucun espace de travail
                  tant qu&apos;ils ne sont pas invités ou qu&apos;ils ne créent
                  pas leurs propres espaces de travail).
                  <br />
                  <br />
                  Si vous voulez autoriser uniquement les utilisateurs invités à
                  s&apos;inscrire, vous pouvez désactiver cette option. Dans ce
                  cas, le formulaire d&apos;inscription sera fermé tant que vous
                  n&apos;aurez pas invité l&apos;utilisateur dans l&apos;un des
                  espaces de travail.
                  <br />
                  <br />
                  Pour inviter des utilisateurs dans un espace de travail,
                  cliquez sur &quot;Add user&quot; et saisissez un e-mail.
                  Ensuite, l&apos;utilisateur avec cet e-mail pourra finaliser
                  son inscription.
                </li>
                <li>
                  <strong>Allow member invitations</strong> - ce paramètre est
                  utile lorsque les inscriptions externes sont désactivées.
                  <br />
                  <br />
                  Imaginez que vous avez déjà des utilisateurs et que vous savez
                  qu&apos;ils sont fiables (par exemple, votre équipe). Vous
                  voulez leur permettre d&apos;inviter d&apos;autres
                  utilisateurs à rejoindre Databasus. Dans ce cas, vous pouvez
                  activer cette option et ils pourront inviter d&apos;autres
                  utilisateurs à rejoindre les espaces de travail via des
                  invitations.
                  <br />
                  <br />
                  Si elle est désactivée, seuls les admins peuvent inviter des
                  utilisateurs.
                </li>
                <li>
                  <strong>Allow member workspace creation</strong> - par défaut,
                  tous les membres peuvent créer leurs propres espaces de
                  travail. Si vous voulez réserver la création des espaces de
                  travail aux admins, vous pouvez désactiver cette option.
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
