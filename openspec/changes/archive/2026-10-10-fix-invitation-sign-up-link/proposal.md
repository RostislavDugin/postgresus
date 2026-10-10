## Why

A person invited to a workspace who has no Databasus account receives an email with a "Sign up" button. The button opens `<DATABASUS_URL>/sign-up`, which the web app has no page for, so the invited person sees a blank page and cannot create the account. The workaround is to open the instance root by hand, which the email never mentions.

## What Changes

- The "Sign up" button in the invitation email links to the configured `DATABASUS_URL` exactly as the operator wrote it, with no path appended. The root of the web app already shows the sign-up form to a visitor who is not signed in. Because nothing is appended, a trailing slash in `DATABASUS_URL` no longer matters.
- The web app sends any address it has no page for to its root instead of rendering a blank page. This repairs the `/sign-up` links in invitation emails that were already sent.
- The membership service gains a way for tests to replace its email sender, the same way the user service already allows. The test double for it already exists in the workspaces testing package and has no caller today.

Out of scope:

- An invitation token or any other secret in the invitation link.
- Invited accounts completing sign-up without proving they own the email address, and without the external registration setting being checked. This is a separate security question and needs its own change.
- A dedicated `/sign-up` page or any other new address in the web app.
- Filling in the invited email address on the sign-up form.
- The invitation email's text, layout and the branch used when `DATABASUS_URL` is not set.
- Escaping `DATABASUS_URL` inside the email markup.

## Capabilities

### New Capabilities

- `workspace-invitations`: what an invited person gets from the invitation email and where its link leads, plus how the web app answers an address it has no page for.

### Modified Capabilities

None.

## Impact

- Backend: `buildInvitationEmailHTML` and a test seam in `backend/internal/features/workspaces/services/membership_service.go`, and a new controller test in `backend/internal/features/workspaces/controllers/membership_controller_test.go`.
- Frontend: `frontend/src/App.tsx` moves to `frontend/src/app/ui/App.tsx`, where the frontend rules place the router, and gains the catch-all route; `frontend/src/main.tsx` imports it from there.
- No API, schema or configuration change. Nothing breaks for existing callers.
- Rules this change answers to: `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`.
