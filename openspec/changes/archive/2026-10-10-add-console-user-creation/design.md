## Context

See proposal.md - Why for the motivation and specs/root-admin-account/spec.md for the behavior.

Console commands are flags on the server binary. `parseCommandLineOptions` (`backend/cmd/main.go:154-176`) reads them, and after migrations `main` calls one `...IfRequested` function per command (`main.go:109-113`). Each function returns at once unless its flag is set; otherwise it does its work and ends the process through `logger.ExitAfterFlush`. A command that writes to the audit log first calls `audit_logs.SetupDependencies()` (`main.go:216`, `:241`), because nothing else installs the writer on the console path (`backend/internal/features/audit_logs/di.go:41-43`).

Both existing ways to create an account go through `createAccountClaimingEmptyInstance` (`backend/internal/features/users/services/user_services.go:715-754`). It decides whether the instance is empty, applies the registration policy to an ordinary account and settles the race for the bootstrap record. The partial unique index admits one flagged account, so the loser of a concurrent first insert is rebuilt as an ordinary account (`:738-753`).

Registration validates its input through Gin binding tags: `email` on the address and `min=8` on the password (`backend/internal/features/users/dto/dto.go:12-13`). Gin evaluates these tags with `github.com/go-playground/validator/v10`, which `backend/go.mod:279` already requires directly. The console reset passes its password straight to `ChangeUserPasswordByEmail` (`main.go:257-269`, `user_services.go:310-321`) and checks neither.

A password change moves `password_creation_time` (`backend/internal/features/users/repositories/user_repository.go:72-79`), and token validation compares against it (`user_services.go:249`). Rotating a password from the console therefore ends the account's existing sessions.

Addresses are unique in the database (`backend/migrations/20250605090323_init.sql:14`) and are compared as stored, without case folding (`user_repository.go:31-41`).

The design is constrained by these `backend/AGENTS.md` rules: one responsibility per file, sentinel errors in `errors.go`, tests that mirror the source split, controller tests preferred and service tests reserved for logic without an API surface, and no passwords or unmasked email addresses in logs. No accepted ADR in `adr/` governs console commands or user provisioning.

## Goals / Non-Goals

**Goals:**

- The creation logic and every refusal live under `backend/internal/features/users/` and are tested there. `main.go` only parses flags, calls the service and prints the outcome.
- The new path and both existing ones share a single empty-instance claim, race handling included.
- One password minimum applies to every way a password is set.

**Non-Goals:**

- A general command framework or subcommand parser. The new command follows the flag style of the existing three.
- Changing how addresses are compared or normalized.

## Decisions

### The command is a flag set, `--create-user` with `--email`, `--password`, `--role` and `--name`

`--create-user` selects the command and `--email` names the account, the same flag the reset uses. The password goes in `--password` rather than `--new-password`, because the reset already triggers on a non-empty `--new-password` (`main.go:237`). `--role` is required and accepts `admin` or `member` in any letter case. `--name` is optional. Without it, the build callback names the account after the address's local part, the text before the `@`. Registration requires a name (`dto.go:14`), and the local part identifies the account in `--list-admins` better than the fixed `User` an invitation gets (`user_services.go:368`). When `--create-user` is combined with `--new-password`, the command refuses and points at `--password`, so a mistyped flag never silently drops the password.

`createUserIfRequested` runs before `resetPasswordIfRequested` in `main`, so the combination reaches that refusal instead of resetting a password.

- Rejected: a `create-user` subcommand like `healthcheck` (`main.go:73`). The other account commands are flags, and documentation and scripts would carry two styles.
- Rejected: reusing `--new-password` for the new account's password. One flag would mean two commands, depending on whether `--create-user` is present.
- Rejected: defaulting `--role` to member. On an empty instance that default fails, and a script that omits the role would create an account other than the one its author meant.

### The empty-instance claim takes the ordinary-account check as an argument

`createAccountClaimingEmptyInstance` gets a parameter: the check to run before an ordinary account is inserted. Registration and OAuth pass `checkExternalRegistrationAllowed` as today. The console passes a check that always allows, since the registration policy does not apply at host trust level. The race handling, the bootstrap flag and the retry stay in one place.

- Rejected: a separate insert path for the console. It would copy the race recovery, the subtlest code in the feature, and the two copies could drift.
- Rejected: a boolean `shouldApplyRegistrationPolicy`. At the call site it reads as an unexplained `false`, and the function would still have to know which policy it skips.
- Rejected: calling `SignUp`. It applies the registration policy, completes invited accounts instead of refusing them, and always creates a member when the instance is not empty.

### The member-as-first-account refusal is checked before the insert

The console path asks whether the instance is empty before calling the shared claim, and refuses a member account on an empty instance. The check races with a registration that lands in between. It can only refuse a member that could in fact have been created; it cannot let a member become the first account. The race is harmless because no code path deletes users, so an instance never becomes empty again. An administrator account takes the instance or, if a registration wins the race, becomes an ordinary administrator through the shared retry.

- Rejected: letting the build callback return an error. That widens the signature shared with the web paths for a refusal only the console has.
- Rejected: promoting the requested member to administrator. The command would do something other than what it was told, and the operator would learn about it only from the listing.

### Idempotency is decided by an exact state match before the insert

The command reads the account by address first. An account that is active, holds the requested role and has a password (`HasPassword`, `backend/internal/features/users/models/user.go:57`) is reported as already existing, with exit code 0, and nothing is written, not even an audit entry. Any other account at that address is a sentinel error and exit code 1. That includes an active account without a password, such as one created through Google or GitHub (`buildOAuthUser`, `user_services.go:774-792`). Reporting success for it would tell a deployment that the address signs in with the given password when it does not. A missing account goes to the shared claim.

- Rejected: always failing on an existing address. Every deployment script would have to parse `--list-admins` before calling the command.
- Rejected: updating an existing account's password or role. That turns creation into an upsert. A rerun with a stale secret would rotate the password, and the console would change existing accounts' roles, which the exception below deliberately leaves out.
- Rejected: accepting an account without a password and setting the given password on it. It would quietly attach a password to an identity its owner set up to sign in through an external provider.

### The console may create an administrator without the bootstrap administrator, but only as a new account

The main spec reserves granting the administrator role for the bootstrap administrator (`openspec/specs/root-admin-account/spec.md`, requirement "The bootstrap administrator keeps its powers over other administrators after changing address"). The delta spec exempts one case: a new account created from the host console with the administrator role. The console reset can already set a password on the bootstrap administrator's own account (`main.go:257-269`) and act as it. Creating another administrator therefore gives the host nothing it does not already have, and the audit entry records each such account. The exception stops at new accounts, which is why the idempotency check above refuses an existing member instead of promoting it.

- Rejected: limiting the console to the bootstrap case, the first account on an empty instance. Every role assignment would stay inside the rule. But an owner who provisions a second operator account, or recreates an administrator on an instance someone registered on first, would be back to clicking through the interface, which is the problem the proposal sets out to remove.
- Rejected: letting the console also grant the role to existing accounts. That is a role-management command, not provisioning, and it widens the exception without a use case.

### The service renders the console lines, so their content is testable

`CreateUserFromConsole` returns an outcome value that tells a created account from an existing one. A function in the same file renders that value into the line the operator sees. It names the address and role, marks a new bootstrap administrator, and for an existing account says that nothing changed and the password was not changed. `main.go` prints the rendered line and, on failure, logs the returned error. The service itself writes no log line, so the rendered lines and the error messages are everything the command can emit. Tests assert that none of them contains the password or the stored hash.

- Rejected: building the lines in `main.go`. Code under `cmd/` is never tested, and the no-secret requirement would rest on a code review.
- Rejected: a grep over `main.go` for `fmt.Print*` and `log.*` arguments. It proves nothing about the content of an error message built deeper down.

### Validation uses the validator Gin already uses

The address is checked with `validator.New().Var(email, "required,email")`, and the password with the same rune count as `min=8`. Both are the library and tags the sign-up binding evaluates, so the console accepts and refuses exactly what registration does. The minimum becomes a named constant in the users feature. The binding tags keep their literal `min=8`, because a struct tag cannot reference a constant.

- Rejected: `net/mail.ParseAddress`, which `backend/internal/util/smtp_transport/address.go:11` uses. It accepts forms like `Ops <ops@example.com>` that registration refuses.

### The password minimum is enforced in `ChangeUserPassword`

`ChangeUserPassword` refuses a password below the minimum before hashing. Every password change passes through it: the console reset via `ChangeUserPasswordByEmail`, the profile change (`backend/internal/features/users/controllers/user_controller.go:188`) and the emailed reset (`user_services.go:637`). The web paths already bind `min=8`, so only the console reset behaves differently. The address is not validated there, so the placeholder `admin` can still be reset.

- Rejected: checking only in `main.go`. Code under `cmd/` is never tested.
- Rejected: checking only in `ChangeUserPasswordByEmail`. The rule would hold for one caller, and the next new caller would skip it.

### The code lives in its own file and is tested at the service level

`CreateUserFromConsole` goes into `backend/internal/features/users/services/console_account.go`, as a `UserService` method, because it needs the user repository, the shared claim and the audit writer. Its refusals are sentinel errors in `backend/internal/features/users/errors/errors.go`. Tests live in `console_account_test.go` next to it, following `disable_two_factor_test.go` and `admin_listing_test.go`. The command has no HTTP surface, so the backend rule allows a service test. Sign-in in those tests goes through `SignIn`, the same function the sign-in endpoint calls.

- Rejected: putting the logic in `main.go` next to `resetPassword`. CI runs only `./internal/...` tests.
- Rejected: `UserManagementService`. It works on accounts that already exist, and it has no access to the claim.

### The audit entry names the address and the role, never the password

A created account gets the entry `Account created from the host console: <email> (role ADMIN)`, attributed to the new account. On an empty instance it also gets the existing `writeInstanceClaimedAuditLog` entry. Neither the output nor the log carries the password. The created and existing lines are printed with `fmt.Println`, as `--list-admins` does (`main.go:198-200`), because the logger masks addresses and the operator needs to read them.

## Risks / Trade-offs

- [The password sits in the command line and is visible in the container's process list while the command runs] → The exposure matches `--new-password`, and only someone already inside the container can see it. Reading the password from standard input is listed out of scope in the proposal.
- [A rerun with a different password reports success and does not change the password, so a deployment can believe a rotation happened] → The output for an existing account says the password was not changed, and the documentation points to `--new-password` for rotation.
- [Two console runs for the same address at the same moment both pass the existence check] → The unique index on `users.email` refuses the second insert, and that run exits with code 1 without corrupting anything. A deployment that retries succeeds on the existing account.
- [`Ops@example.com` and `ops@example.com` are different accounts] → This is how registration already behaves. The command does not change it.
- [The minimum length now refuses short console resets an operator's script may rely on] → The proposal marks this as BREAKING. The error says what the minimum is.

## Migration Plan

No schema or data migration. The change ships in the server binary. Rolling back the binary removes the command and the reset's minimum; accounts created by the command remain ordinary accounts and keep working.
