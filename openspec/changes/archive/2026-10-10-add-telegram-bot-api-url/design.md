## Context

See proposal.md - Why for the motivation and specs/telegram-notifier-endpoint/spec.md for the required behavior.

The Telegram notifier builds its request address from a constant host: `backend/internal/features/notifiers/models/telegram/model.go:75` formats `https://api.telegram.org/bot%s/sendMessage`. Everything else the change needs already exists and only has to accept a second host:

- The notifier's HTTP client comes from `buildHTTPClient` in `backend/internal/features/notifiers/models/telegram/proxies.go:22`. Without a proxy it leaves `Transport` nil, so the default transport applies the proxy environment variables to whatever host the request names.
- Save, update and a direct test send for a saved notifier merge the incoming notifier, encrypt it and call `Validate` before anything is stored or sent (`backend/internal/features/notifiers/service.go:59-67`, `:97-101`, `:264-270`). A direct test send for a notifier that has not been saved yet only encrypts it and sends (`service.go:276-285`), so `Validate` alone does not guard that path.
- A transport error passes through `notifier_models.ErrorWithoutWebhookURLCredentials` (`model.go:100`) before it becomes the notifier's `LastSendError` (`backend/internal/features/notifiers/model.go:60-64`). `redactPathSecrets` in `backend/internal/features/notifiers/models/url_credentials.go:49` keeps only the first path segment and strips the token from a fused `bot<token>` segment, so a prefixed path such as `/telegram/bot<token>/sendMessage` is already redacted.
- A non-200 response is reported with its body verbatim (`model.go:107-111`). That body comes from whatever server the URL names.
- The email notifier's form already has a collapsible advanced section. Its row layout, `AdvancedFieldRowComponent`, is a private function at the bottom of `frontend/src/features/notifiers/ui/edit/notifiers/EditEmailNotifierComponent.tsx:284`, and the toggle header is inline markup at `:188-201`.
- The proxy columns were added by `backend/migrations/20260622120000_add_telegram_proxy.sql` as `is_proxy_enabled BOOLEAN NOT NULL DEFAULT FALSE` and a nullable `proxy_url TEXT` mapped to a Go `string`.

## Goals / Non-Goals

**Goals:**

- One optional field on the Telegram notifier decides the server; empty keeps today's behavior byte for byte.
- The same validation rules in the backend and in the form, so the form never offers a save the API rejects.
- No path through which the bot token reaches `LastSendError`, the logs or a test-send response, whichever server answers.

**Non-Goals:**

- Changing the request itself: method, form encoding, `parse_mode`, topic handling and timeout stay as they are.
- A shared "custom API base URL" abstraction for other notifiers.

## Decisions

### 1. A per-notifier field, not an environment variable

The URL is a column on `telegram_notifiers`, edited in the Telegram form.

- *Rejected: `TELEGRAM_API_URL` in the process environment*, as the issue proposed. It needs a container restart to change, cannot be seen or tested from the form, applies to every notifier in every workspace at once, and would be the only notifier setting that lives outside the notifier. The proxy, the closest existing setting, is already per notifier.

### 2. Empty means the official API; no separate switch

The field `BotAPIURL string` (`json:"botApiUrl"`, column `bot_api_url`) is empty by default, and an empty value sends to `https://api.telegram.org`.

- *Rejected: an `IsCustomBotAPIEnabled` switch plus a URL, like the proxy.* The proxy needs the switch because its URL is encrypted and hidden, so the form cannot tell "unchanged" from "cleared" by the field alone. The Bot API URL is returned in clear (decision 3), so the field itself is the state, and a switch would be a second source of truth.
- *Rejected: prefilling `https://api.telegram.org` into the field.* Every existing row would need a data migration, and a stored default would stop following the code if the default ever changed.

### 3. Stored and returned in clear, credentials rejected

The URL is not encrypted, `HideSensitiveData` leaves it alone, and `Validate` rejects a URL whose user info is set.

- *Rejected: encrypting it like the proxy URL.* That would allow basic auth for a reverse proxy, but the URL could no longer be shown, the form would need the proxy's "unchanged vs cleared" dance, and the user explicitly chose the simpler form. Rejecting user info is what makes storing in clear safe: the only secret on the request is the bot token, which stays encrypted in its own column.

### 4. Validation in a dedicated parser next to the proxy parser

A new `validateBotAPIURL` in `backend/internal/features/notifiers/models/telegram/bot_api_url.go` accepts `http`/`https`, requires a host, and rejects user info, a raw query and a fragment. `Validate` calls it when the field is set, and `Send` calls it again before building the request address, the way `buildHTTPClient` re-parses the proxy URL. The second call is what covers the direct test send of an unsaved notifier, which skips `Validate`: an invalid URL fails the send with the same error instead of reaching the network. The frontend mirrors the rules in `validateTelegramNotifier.ts` with `new URL()`.

- *Rejected: adding `Validate` to the unsaved direct test send.* It would change that path for every notifier type, which is outside this change.

- *Rejected: reusing `parseProxyURL`.* Its scheme list includes `socks5` and `socks5h`, and it allows user info, both wrong for an API endpoint.
- *Rejected: also blocking private and loopback addresses.* A self-hosted Bot API server is normally on the same Docker network or on `localhost`, which is the main use case. The webhook notifier already sends to any address, so a block here would add friction without closing anything.

### 5. Build the address by trimming the trailing slash at send time

`Send` joins `strings.TrimRight(baseURL, "/")` with `/bot<token>/sendMessage`. The value is stored exactly as entered (after the form trims whitespace).

- *Rejected: normalizing on save.* It would mutate user input in `Update` and `EncryptSensitiveData`, and `Send` would still need to be safe for rows written before or around the normalization. One place that builds the address is simpler.
- *Rejected: `url.JoinPath`.* It removes `./` and `../` elements from the joined path, so the request could go to a different path than the one the user entered and validation accepted. Concatenation sends exactly the validated prefix.

### 6. Strip the bot token from a non-OK response body

Before the response body goes into the returned error, every occurrence of the decrypted bot token is replaced with the `redacted` marker. The marker is the unexported `redactedURLSegment` in `backend/internal/features/notifiers/models/url_credentials.go:10`, which the `telegram_notifier` package cannot reach, so `notifier_models` gets a small exported `RedactSecretInText(text, secret string) string` beside it. Every redaction then uses the same marker, and the constant stays private.

The redaction happens inside `TelegramNotifier.Send`, which both the test sends and the scheduled sends call, so the test-send response and `LastSendError` carry the same redacted text.

- *Rejected: keeping the body verbatim.* The official API never echoes the request path, but an arbitrary reverse proxy can: an Express-style "Cannot POST /bot<token>/sendMessage" page, or a gateway that echoes the upstream URL, would put the token into `LastSendError` and the test-send response. The spec forbids that.
- *Rejected: dropping the body.* It holds Telegram's own explanation, such as "chat not found", which users need to fix their settings.

### 7. Nullable `TEXT` column, matching `proxy_url`

The migration adds `bot_api_url TEXT` with no default, through a stub created by `goose -dir backend/migrations create add_telegram_bot_api_url sql`.

- *Rejected: `TEXT NOT NULL DEFAULT ''`.* Either works with a Go `string`; the nullable form matches the sibling `proxy_url` column, so the table reads consistently.

### 8. Advanced-section components move to `shared/ui`

`frontend/src/features/databases/ui/edit/AdvancedSettingsToggleComponent.tsx` already renders the toggle header the email form repeats inline, but it hard-codes the `databases.edit.advancedSettings` key. It moves to `frontend/src/shared/ui/AdvancedSettingsToggleComponent.tsx` and takes the label as a prop. `AdvancedFieldRowComponent` moves out of the email form into `frontend/src/shared/ui/AdvancedFieldRowComponent.tsx`. Both are exported from `shared/ui/index.ts`. The five database forms, the email form and the Telegram form use them, and the inline header in the email form is deleted.

The Telegram toggle label comes from a new `notifiers.advancedSettings` key, which replaces `notifiers.email.advancedSettings`, so both notifier forms read one key. The Telegram form's section starts open when `botApiUrl` is set or `isProxyEnabled` is true. The proxy switch and proxy URL move into the section unchanged.

The shared header uses the database forms' spacing (`mb-1`), so the gap under the email form's header shrinks from `mb-3` to `mb-1`. That is the only visible change to existing forms.

- *Rejected: a notifier-only copy under `features/notifiers`.* It would be the third copy of the same header, beside the database one and the inline email one.
- *Rejected: importing the component from `features/databases`.* The frontend FSD rules forbid cross-imports between slices of the same layer; `shared/ui` is the layer both can import.
- *Rejected: also replacing the inline headers in the storage and logical backup forms.* They are outside the code this task touches; the shared component makes that a mechanical follow-up.

### 9. Testing through the controller with a local server

Controller tests in `backend/internal/features/notifiers/controller_test.go` use `httptest` servers as the Bot API server and as a proxy:

- A saved notifier whose Bot API URL is the server with a path prefix and a trailing slash, sent through `POST /notifiers/:id/test`, delivers `/<prefix>/bot<token>/sendMessage` with the notifier's `chat_id`.
- An unsaved notifier sent through `POST /notifiers/direct-test` reaches the server the same way. With an invalid URL or a URL with user info, the direct test returns an error and the server records no request.
- A notifier with both a Bot API URL and a proxy pointing at an `httptest` proxy makes the proxy receive the request for the Bot API URL.
- A server that answers 404 with a body echoing the request path yields an error that keeps the body with `redacted` in place of the token.
- A Bot API URL with a prefix pointing at a closed server yields an error that names the host and not the token.
- Saving an invalid or credential-bearing URL is rejected and the stored notifier is unchanged; reading a notifier returns `botApiUrl` in clear while the token and proxy URL stay empty.

Unit tests in `model_test.go` cover the rules of `Validate` as table tests beside the existing proxy `Validate` tests, and the address an empty Bot API URL produces (`https://api.telegram.org/bot<token>/sendMessage`), which no controller test can reach without calling Telegram. `url_credentials_test.go` covers `RedactSecretInText`. On the frontend, `validateTelegramNotifier.test.ts` covers the URL rules, and render tests with `renderToStaticMarkup`, the pattern `PhysicalBackupTypeCellComponent.test.tsx` uses, check that the Telegram form starts with the advanced section collapsed for a new notifier and expanded with a proxy or a Bot API URL, and that the read-only view shows the Bot API URL.

- *Rejected: unit-testing only the address builder.* The controller test covers merge, encryption, validation and sending in one pass, which is the contract the user relies on, and `backend/AGENTS.md` prefers controller tests.

## Risks / Trade-offs

- [The bot token goes to whatever server the URL names, over plain `http` if the user chooses it] → This is the feature: the user points their own bot at their own server. The tooltip says the URL receives the bot token, and the user decided against a further warning.
- [A user with edit rights on notifiers can make the backend send POST requests to an internal address] → The webhook notifier already allows the same, so no new exposure is added; recorded so a later SSRF policy covers both.
- [A self-hosted Bot API server rejects a bot still logged in to the official API] → The tooltip mentions that a bot must be moved to the self-hosted server first; Telegram's error reaches the user through the response body.
- [Moving the proxy into a collapsed section hides it from users who look for it] → The section opens on its own whenever a proxy is enabled, so existing users see it in the same place they configure it.

## Migration Plan

The migration only adds a nullable column, so it runs online and existing rows keep sending to the official API. Rolling back drops the column; a notifier that relied on a custom server then sends to the official API again, which matches the code being rolled back.
