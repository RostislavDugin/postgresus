## Context

See proposal.md for the motivation. The current state that shapes the approach:

- `Dockerfile:197` copies the developer `.env.example` to `/.env`, and `Dockerfile:198` strips only `DATABASE_DSN` from it. The comment at `Dockerfile:189-196` explains the copy as a source of defaults, but no production setting depends on it (see the table below).
- `backend/internal/config/config.go:145` calls `godotenv.Load` and exits the process when the file is missing. The image keeps a file only to satisfy this call.
- `godotenv.Load` does not overwrite a variable that already exists, even with an empty value. `unsetEmptyEnvVars` (`config.go:298`) then removes empty variables, so the code sees them as unset. That is why `SMTP_USER=` worked as a workaround while a missing `SMTP_USER` picked up the baked test login.
- `backend/internal/util/logger/settings.go:244` loads the same file and already ignores a missing one.
- `backend/internal/features/email/di.go` derives the security mode from the port only when `SMTP_SECURITY` is empty. `smtp_transport.openSession` skips authentication when either the username or the password is empty. Both defaults are correct, and the baked file is what bypassed them.
- `docker/start.sh:147` decides the frontend's "mail configured" flag from the real environment, while the backend reads the merged one. With no SMTP variables set, the two disagree today.
- The frontend stage at `Dockerfile:14-18` looks for `frontend/.env.production.example`, which was deleted in 082f8f9c. `COPY frontend/ ./` carries any local `frontend/.env`, `frontend/.env.local`, `frontend/.env.production` or `frontend/.env.production.local` into the build, because the `.env.local` and `.env.development` entries in `.dockerignore` match only at the root. Vite reads those files in `pnpm build`, and `frontend/src/constants.ts:32-43` falls back to `import.meta.env.VITE_*` for the OAuth client IDs, the mail flag and the Turnstile key, so their values would be baked into the bundle.

What production reads from the baked file today, and what replaces it:

| Variable | Replacement after the change |
|---|---|
| `ENV_MODE` | `ENV ENV_MODE=production` in the Dockerfile already wins |
| `DATABASE_DSN` | already stripped; `start.sh` publishes the embedded one |
| `LOG_LEVEL`, `LOG_FILE_IS_ENABLED` | the code defaults (info, enabled) equal the baked values |
| `SHOW_DB_INSTALLATION_VERIFICATION_LOGS` | `config.go` sets `true` when empty, equal to the baked value |
| `SMTP_*` | unset; this is the fix |
| `TEST_*`, `GOOSE_*`, `DEV_DB_*`, `WAL_BACKUP_*`, `DATABASE_URL`, `VITE_*` | not read by the production binary; migrations take their DSN from the config (`backend/cmd/main.go:502-503`) |
| OAuth, Turnstile, `DATABASUS_URL`, `OPEN_TELEMETRY_*` | empty in the file, so nothing changes |

## Goals / Non-Goals

**Goals:**

- The image contains no environment file at all, so no future addition to `.env.example` can reach production.
- A developer's local `.env` files never enter a Docker build context.
- Startup without an environment file succeeds and says so in the log.

**Non-Goals:**

- Changing the precedence between the process environment, the published embedded DSN and the file.
- Making `logger/settings.go` and `config.go` share one loader. They load the same file for different reasons at different times, and merging them is a separate refactor.

## Decisions

### Remove the file from the image instead of shipping an empty or production-only one

The Dockerfile drops the `COPY` and the `sed` line, and the backend stops requiring the file.

Alternatives rejected:

- **An empty `/.env` (`RUN touch /.env`).** This is a one-line fix, but it keeps the slot open. The next person who needs a default would put it there, and we would be back to values that bypass the code defaults.
- **A separate `docker/.env.production` with production values.** Every value it could hold already exists as a code default or a Dockerfile `ENV`. A second source of defaults would drift from the first. The previous `backend/.env.production.example` did exactly that and carried a development database password.
- **Stripping `SMTP_*` with `sed`.** This fixes the symptom for one prefix and leaves every other development value in the image.

### Make the file optional in `config.go`; a missing required variable stays fatal

Before calling `godotenv.Load`, `config.go` checks whether the file exists. A missing file logs at info level that no environment file was loaded, and startup continues. A file that exists but cannot be read or parsed stays fatal, because silently ignoring a broken file would hide a misconfiguration. The existing required-field checks keep catching a development checkout that has no `.env`.

The existence check and the load go into a small helper that returns whether the file was loaded and any error. That makes it testable without running `loadEnvVariables`, which exits the process on failure.

Alternatives rejected:

- **Fatal in development, optional in production.** The mode comes from `ENV_MODE`, which the file itself may set, so the decision would depend on the thing being loaded. The missing-variable checks already give a developer a precise message.
- **Ignoring every load error, the way `logger/settings.go` does.** A malformed file would then disappear silently from the configuration.

### Exclude every `.env` file from the build context and delete the dead frontend block

`.dockerignore` gets `**/.env` and `**/.env.*`, which replace the root-only `.env.local` and `.env.development` entries. The root `.env.example` no longer needs to be in the context once the copy is gone, so no exception is needed for it. `Dockerfile:14-18` is deleted. The frontend build reads `VITE_APP_VERSION` from the `ENV` at `Dockerfile:8`, and runtime flags come from `runtime-config.js`, which `start.sh` generates. The frontend build needs no file.

Alternatives rejected:

- **Keep the block and only add the ignore rule.** The block would still refer to a file that does not exist.
- **Exclude only `**/.env`.** Vite also reads `.env.local`, `.env.production` and `.env.production.local`, so local values would still reach the bundle.

### Pin the behavior with Go tests, not an image test

Tests in `config_test.go` cover the helper: a missing file, a file that loads, a file that does not override a variable already set, even to an empty value, and a malformed file. A test in `features/email` covers a host and port with no credentials and no security mode, which must give `starttls`. A transport test against the existing fake SMTP server sends with empty credentials over STARTTLS and asserts that the delivery succeeds and no `AUTH` command is issued. The tests run the binary's real loading path, so they would catch a code-level regression.

Alternative rejected: a CI step that runs the built image and asserts that `/.env` does not exist. It would catch a re-added `COPY`, but it costs an image build per run, and the proposal scopes it out. A Dockerfile comment at the former copy site records why there is no environment file, which is the cheaper guard.

### Constraints from module rules

- Root `AGENTS.md`: keep Dockerfiles free of secrets. The baked file carried development passwords, so removing it also satisfies this rule.
- Root `AGENTS.md`: no unrequested backward compatibility. Nothing replaces the baked file, and the BREAKING note in the proposal states what operators lose.
- `backend/AGENTS.md`: never log secrets. The new log line names only the file path and whether it was loaded.
- No ADR in `adr/` covers configuration loading, so this change does not conflict with an accepted decision.

## Risks / Trade-offs

- [An operator unknowingly relied on a baked SMTP value, for example a working relay that accepted the test login] → The relay would have needed to accept `info@databasus.com`/`password`, which is implausible. The release notes name the change, and the logged SMTP failure points at the missing variable.
- [Someone re-adds a `COPY` of an env file to the Dockerfile] → The Dockerfile comment explains why there is none, and the review hooks see the diff. A CI image check stays out of scope by decision.
- [A developer runs the backend without `.env` and sees an unfamiliar error] → The info line saying no environment file was loaded comes right before the configuration error. A manual run from a directory without `.env` and with an empty environment logged `DATABASE_DSN is empty` and exited with status 1.
- [An operator mounts their own file at `/.env`] → It keeps working, because the backend still loads the file when it exists.

## Migration Plan

Nothing to migrate in data. Operators upgrading an image that relied on a baked value set that variable explicitly. Rolling back means reverting the commit; no state depends on it.
