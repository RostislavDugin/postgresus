## Why

An instance gets its first account only through the registration form or a first Google or GitHub sign-in (`backend/internal/features/users/services/user_services.go:58`, `:965`). An owner who deploys Databasus from scripts therefore cannot provision an administrator without somebody opening the browser. They need an administrator that exists without a manual sign-up, so they can keep its password in a secrets vault and rotate it from the host. Everyone else still signs up as usual. The host console already lists administrators and resets passwords (`backend/cmd/main.go:184`, `:232`); creating an account is the missing third command.

## Governing docs

This change answers to [`AGENTS.md`](../../../AGENTS.md) at the repo root, [`backend/AGENTS.md`](../../../backend/AGENTS.md), [`website/AGENTS.md`](../../../website/AGENTS.md) and [`assets/readme/AGENTS.md`](../../../assets/readme/AGENTS.md). It touches no frontend and no verification agent code, so [`frontend/AGENTS.md`](../../../frontend/AGENTS.md) and [`agent/verification/AGENTS.md`](../../../agent/verification/AGENTS.md) do not apply.

## What Changes

- Add a console command that creates an account from inside the running instance: `docker exec -it databasus ./main --create-user --email="ops@example.com" --password="..." --role=ADMIN`, with an optional `--name`. The account is active and holds a password from the moment it is created, so it signs in immediately and the existing `--new-password` reset rotates its password like any other account's.
- Keep the first-account rule. The command run on an instance that holds no account creates its bootstrap administrator, with the same audit entry a first registration writes. Asked for `--role=MEMBER` there, it refuses, because a member as the first account would leave the instance with nobody able to appoint administrators.
- Skip the registration policy. The command works with open registration switched off, since whoever runs it already has a shell inside the instance. An owner can close registration and provision accounts from the console only.
- Let the console create an administrator on an instance that already has one, without the bootstrap administrator. Today only that account grants the role. The console already resets the bootstrap administrator's own password, so the exception adds no new power to the host. It covers new accounts only: the console never changes the role of an existing one.
- Make a repeated run harmless. When an active account with the same address and role and a password already exists, the command reports that nothing changed and that the password was not changed, and exits successfully. Any other existing account with that address makes the command fail: one with another role, a deactivated one, an invited one, or one without a password that signs in through Google or GitHub.
- Validate the input the way registration does: a syntactically valid email address and a password of at least eight characters.
- Apply the same eight-character minimum to every password change, the `--new-password` console reset included. Today the console reset accepts any non-empty password (`backend/cmd/main.go:232-269`), while every web path binds `min=8` (`backend/internal/features/users/dto/dto.go:13`, `:73`, `:140`). **BREAKING** for the console reset: a password shorter than eight characters is now refused. The address check is not added to the reset, so an upgraded instance can still reset the placeholder account `admin`.
- Record each account the command creates in the audit log, marked as created from the host console.
- Document the command on the `/password` website page and in its five translations, and in the password section of the root `README.md` and its five translations, beside `--list-admins` and the reset.

### Out of scope

- Configuring accounts through the Helm chart, environment variables or a config file. The command runs inside a running instance; wiring it into a deployment is the operator's job.
- Reading the password from standard input or an environment variable. The command takes it as an argument, the way `--new-password` does, and shares that command's exposure in the container's process list.
- Updating an existing account from this command: its role, name, status or password. Role changes stay in the web interface, and passwords stay with `--new-password`.
- Adding the created account to any workspace. A member created from the console joins workspaces the usual way.
- Creating a second bootstrap administrator, or moving that record to another account. An administrator created on an instance that already holds accounts is an ordinary administrator.
- Deleting accounts from the console.
- A console command for the invitation flow.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `root-admin-account`: adds creating an account from the host console, including how it claims an empty instance and why it is safe to run repeatedly; exempts that creation from the rule that only the bootstrap administrator grants the administrator role, for new accounts only; adds the password minimum to the console reset; extends the documentation requirement to cover account creation.

## Impact

- Backend: `backend/cmd/main.go` gains the flags and the call; the creation logic and its tests live under `backend/internal/features/users/`, because CI runs the tests of `./internal/...` only. The empty-instance claim in `createAccountClaimingEmptyInstance` (`user_services.go:715`) is shared with the new path rather than duplicated.
- No migration, no API change and no frontend change.
- Website: `website/app/(en)/password/page.tsx` and `website/app/[lang]/password/content/{ru,es,pt,zh,fr}.tsx`.
- README: `README.md` and `assets/readme/README.{ru,es,pt,zh,fr}.md`.
