## Why

The Docker image ships the developer `.env.example` as `/.env`, and the backend loads it for every variable the operator did not set. Since v3.34 every production instance has silently inherited the developer mail settings: a mailpit host, test credentials, a `databasus.com` sender and, since v3.60, the `none` security mode. An operator who sets `SMTP_HOST` and `SMTP_PORT` for a relay without authentication gets an authentication attempt with the test credentials, which the relay rejects. An operator who sets real credentials but leaves `SMTP_SECURITY` unset gets the `none` mode, which sends the password unencrypted instead of using STARTTLS. Nothing a developer puts in `.env.example` should reach a production instance.

## What Changes

- The Docker image no longer contains any environment file. Every setting an operator does not supply stays unset, and the backend falls back to its own defaults in code.
- The backend treats the environment file as optional. When the file is missing, it starts from the process environment alone and logs that no file was loaded. A missing required variable still stops startup with a message saying a required setting is missing.
- **BREAKING:** An image-based instance that relied on a value from the baked file now sees that variable unset. With no SMTP variables set, the backend no longer thinks a mail server exists, so it matches the settings screen. With SMTP partly configured, the unset variables fall back to their defined defaults: no authentication without both a username and a password, a security mode derived from the port, and the sender checks that already apply. No production default is lost, since every non-SMTP value in the file either matches the code default or is overridden by the image. The bake step and its `DATABASE_DSN` stripping are deleted, with no fallback file left in their place.
- The frontend build stage drops its lookup of `frontend/.env.production.example`, a file that no longer exists. The Docker build context excludes every `.env` and `.env.*` file at any depth, so neither the backend's nor the frontend's local environment (including Vite's `.env.local` and `.env.production` variants) can reach a locally built image.

## Capabilities

### New Capabilities

- `runtime-configuration`: where the application's settings come from in a deployed image and in development, and what happens when the environment file is missing.

### Modified Capabilities

- `outgoing-email`: the instance mail server reads only values the operator supplied. A variable the operator did not set counts as unset, so an unset `SMTP_SECURITY` falls back to the port-based default and an unset username or password means no authentication.

## Out of scope

- Reworking `.env.example` itself or the development workflow around it. Developers still copy it to `.env`.
- Changing any SMTP default, validation rule or delivery behavior already defined in `outgoing-email`.
- A CI step that inspects the built image. Go tests pin the loading and delivery behavior; a re-added copy step in the Dockerfile is guarded only by the Dockerfile comment and review.
- Replying on the GitHub issue or publishing a workaround for v3.60.0 users.
- Other env-driven subsystems (OAuth, Turnstile, logging) beyond confirming that their code defaults match what the baked file supplied.

## Impact

- `Dockerfile`: remove `COPY .env.example /.env`, the `DATABASE_DSN` `sed` line and the dead frontend `.env.production.example` block.
- `.dockerignore`: exclude `**/.env` and `**/.env.*`, replacing the root-only `.env.local` and `.env.development` entries.
- `backend/internal/config/config.go`: load the environment file only when it exists.
- `backend/internal/config/config_test.go`: cover startup without a file.
- `backend/internal/util/smtp_transport` tests: cover delivery without credentials over STARTTLS.
- Module rules: root `AGENTS.md` (Dockerfile, build context) and `backend/AGENTS.md` (config loading).
