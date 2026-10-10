## Why

A Telegram notifier always sends to `https://api.telegram.org`. People who run their own Telegram Bot API server, or who reach Telegram through a reverse proxy because the official address is blocked or filtered in their network, cannot use the Telegram notifier at all. The per-notifier proxy setting does not help them: it routes traffic through a forward proxy but still targets the official host.

The issue that raised this asks for a process-wide environment variable. A per-notifier setting fits better. It is edited in the same form as the bot token and the proxy, it takes effect without restarting the container, and two notifiers can point at different servers.

## What Changes

- A Telegram notifier gets an optional Bot API URL. When it is empty, the notifier sends to the official Telegram API as today. When it is set, the notifier sends `sendMessage` to that server instead, keeping the standard `/bot<token>/sendMessage` path under it.
- The URL is validated on save and on test send, in the backend and in the form: `http` or `https` only, a host is required, a path prefix is allowed, and a username or password, a query string or a fragment is rejected. A trailing slash is accepted and does not produce a double slash in the request.
- The URL is not a secret. It is stored as entered, returned by the API and shown on the notifier's page. Because credentials are rejected, nothing secret can end up in it.
- The Telegram form gets a collapsible "Advanced settings" section, like the email notifier's. The new Bot API URL field and the existing proxy switch and proxy URL move into it. The section opens on its own when the notifier already has a Bot API URL or a proxy.
- The advanced-settings toggle that the database forms already use moves to the shared UI layer and serves the database, email and Telegram forms, instead of a third copy.
- The new label and tooltip are added to all six interface dictionaries.

Nothing here is BREAKING. Existing notifiers get an empty Bot API URL and keep sending to the official API, and the API only gains an optional field.

## Capabilities

### New Capabilities

- `telegram-notifier-endpoint`: where a Telegram notifier sends its messages, which Bot API URLs it accepts, and how the URL is stored and shown.

### Modified Capabilities

None. The `outbound-http-proxy` requirements keep holding as written: a notifier with its own proxy still uses it, and a notifier without one still follows the proxy environment variables, now for whichever host its Bot API URL names.

## Out of scope

- A process-wide `TELEGRAM_API_URL` environment variable, as the issue proposed. The per-notifier field replaces it.
- Credentials in the Bot API URL, such as basic auth for a reverse proxy. Supporting them would mean encrypting the URL and hiding it like the proxy URL.
- A warning or a restriction for plain `http` URLs. A self-hosted server on the same Docker network is usually reached over `http`, and the user chose not to add a warning.
- Blocking private or internal addresses. The webhook notifier already lets a user send requests to any address, so this field adds no new kind of exposure.
- Guiding the user through moving a bot from the official API to a self-hosted server (the `logOut` call), beyond a mention in the tooltip.
- Other Telegram methods or features, and custom API URLs for other notifiers.
- The website's advanced configuration page, which documents environment variables and gains none here.

## Impact

- **Module rules this change answers to:** the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
- **Database:** one migration adds a nullable `bot_api_url` column to `telegram_notifiers`.
- **Backend:** the Telegram notifier model in `backend/internal/features/notifiers/models/telegram/` (field, validation, update, sending), its unit tests and the notifier controller tests.
- **API:** create, update, read and test-send of a Telegram notifier carry an optional `botApiUrl`.
- **Frontend:** the Telegram notifier type, its validation and tests, the edit form and the read-only view, plus the six dictionaries under `frontend/src/shared/i18n/locales/`. The advanced-settings toggle and field row move to `frontend/src/shared/ui/`, so the five database forms and the email form switch to the shared components; the email form's header gets the database forms' slightly smaller bottom spacing.
