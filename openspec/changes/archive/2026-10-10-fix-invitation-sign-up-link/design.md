## Context

See proposal.md for the problem. The current state that shapes the fix:

- `buildInvitationEmailHTML` appends `/sign-up` to `DATABASUS_URL` when the variable is set (`backend/internal/features/workspaces/services/membership_service.go:366-380`). No other code builds a link from `DATABASUS_URL`. The other two readers only derive the SMTP greeting name from it (`backend/internal/features/email/di.go:27`, `backend/internal/features/notifiers/models/email_notifier/model.go:102`).
- The backend answers every unmatched path with `index.html` (`backend/cmd/main.go:560-572`). An unknown path therefore loads the web app rather than returning 404.
- The web app's route table has three entries: `/auth/callback`, `/storages/google-oauth` and `/` (`frontend/src/App.tsx:44-50`). Nothing matches any other path, and React Router renders nothing.
- At `/`, a visitor who is not signed in gets `AuthPageComponent`, which starts in sign-up mode (`frontend/src/pages/AuthPageComponent.tsx:21-23`). The root already is the sign-up page.
- Signing up with an invited address activates the existing invited account and keeps its workspace membership (`backend/internal/features/users/services/user_services.go:64-109`). The sign-up form needs no extra parameter for the invitation to take effect.
- The membership service receives the SMTP sender in its composite literal (`backend/internal/features/workspaces/services/di.go:26-35`) and offers no way to replace it. The user service has `SetEmailSender` for this (`backend/internal/features/users/services/user_services.go:54-56`), and its controller tests use it (`backend/internal/features/users/controllers/password_reset_test.go:22-23`). The workspaces package already has `MockEmailSender` (`backend/internal/features/workspaces/testing/mocks.go`), with no caller.

## Goals / Non-Goals

**Goals:**

- The invitation link opens the sign-up form for every value of `DATABASUS_URL` an operator could reasonably write, including one with a trailing slash.
- Links in invitation emails that were already sent start working without any action from the operator.
- The link the email contains is pinned by a test that goes through the HTTP API.

**Non-Goals:**

- Normalizing `DATABASUS_URL` at configuration load. With nothing appended to it, the email has no reason to normalize it.

## Decisions

### The link is `DATABASUS_URL` itself

The "Sign up" link becomes the configured `DATABASUS_URL`, unchanged.

Alternatives rejected:

- **Keep `/sign-up` and add a matching page.** A `/sign-up` page would render what `/` already renders. It adds an address the web app must keep serving and gives the user nothing. A dedicated page earns its place only when it does something the root does not, such as reading an invitation token. That work is out of scope.
- **Trim a trailing slash, then append a path.** This only matters while a path is appended. Linking to the root removes the reason to trim.
- **Change only the backend.** Emails already sent would keep pointing at `/sign-up` and keep showing a blank page.

### The router moves to the `app/` layer

`frontend/src/App.tsx` moves to `frontend/src/app/ui/App.tsx`, its relative imports are updated, and `frontend/src/main.tsx:6` imports it from the new place. The file holds the app-wide providers and the router, which `frontend/AGENTS.md` ("Where new code goes") places in `app/`, a layer that has segments and no slices. `frontend/src/app/` does not exist yet. Root `AGENTS.md` ("Refactoring within a feature") makes fixing a module-doc violation in touched code part of the same task. The move touches one file and one import and changes no behavior.

Alternatives rejected:

- **Leave the file at `frontend/src/App.tsx`.** The route table this change edits would stay outside the layer the frontend rules assign it to, and the rule would be broken again by the next router edit.
- **`frontend/src/app/App.tsx` without a segment.** `app/` takes segments only, and `ui/` is the segment for components.

### Unknown paths redirect to `/`

The route table gains a catch-all `*` route that renders `<Navigate to="/" replace />`.

Alternatives rejected:

- **Render the root's content at any path.** The address bar would keep showing `/sign-up` or a typo, and a later route added at that address would silently change what an old link shows.
- **Show a "page not found" screen.** The only known unknown path is the one in sent invitation emails. A not-found screen would leave those people one more click from the form, and it needs translated text in six dictionaries.
- **A redirect without `replace`.** The unknown address would stay in the history, and the back button would bounce the user to it and straight back to `/`.

### The test drives the HTTP API with a replaced email sender

`MembershipService` gets `SetEmailSender`, mirroring the user service. A controller test in `membership_controller_test.go` swaps in the existing `workspaces_testing.MockEmailSender`, sets `config.GetEnv().DatabasusURL`, adds a member with an unknown address through `POST /api/v1/workspaces/memberships/{id}/members`, and asserts the `href` in the captured body. The test restores the sender and the URL in `t.Cleanup`, because both live in package-level state shared by the other tests in the package.

Alternatives rejected:

- **A unit test of `buildInvitationEmailHTML`.** `backend/AGENTS.md` prefers controller tests and reserves unit tests for complex model logic with no API surface. Building an email string is neither.
- **A constructor parameter instead of a setter.** The service is a package-level composite literal built at init. A different construction path would be a wider change than the one the user service already set the pattern for.

### No automated frontend test for the catch-all

The frontend change is verified by `pnpm typecheck`, `pnpm lint` and `pnpm build`. They prove the route compiles, but no automated test pins the redirect itself.

Alternatives rejected:

- **A component test.** Vitest runs in the `node` environment (`frontend/vitest.config.ts:12`), and the project has no DOM environment or Testing Library. Adding both for one declarative route costs more than the route.
- **A `matchRoutes` test against the route table.** The routes are inline JSX inside `AppContent` and depend on its state. Testing them this way would mean extracting the table first, which is restructuring for the test's sake.

## Risks / Trade-offs

- [An operator sets `DATABASUS_URL` to an address that does not open the web app root] → The email now links to exactly what the operator configured, so the link is as correct as the setting. This was already true for the `/sign-up` link.
- [The catch-all hides a future route that is mistyped in a link] → The user lands on the main screen instead of a blank page. That is a better failure, but it makes a broken link quieter. Accepted.
- [No automated test pins the redirect] → A later edit of the route table could drop the catch-all unnoticed. The route is one declarative line next to the others, and the gap closes for free if the project later adds a DOM test environment.
- [The test mutates package-level state] → No test in the package runs in parallel, and the test restores both values in `t.Cleanup`.
