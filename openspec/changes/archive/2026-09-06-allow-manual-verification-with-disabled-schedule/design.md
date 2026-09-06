## Context

`EnqueueManualVerification` creates a `MANUAL` verification in `PENDING` after the existing authorization and backup checks (`backend/internal/features/verification/runs/service.go:39-106`). The verification scheduler calls `sweepCanceledByDisabledConfig` on every tick (`backend/internal/features/verification/runs/scheduler.go:42-68`). That sweep cancels every row returned by `FindNonTerminalForDisabledConfigs` (`backend/internal/features/verification/runs/scheduler.go:240-263`).

`FindNonTerminalForDisabledConfigs` currently filters by disabled configuration and non-terminal status, but not by verification trigger (`backend/internal/features/verification/runs/repository.go:276-290`). The model distinguishes `MANUAL`, `SCHEDULED`, and `AFTER_BACKUP` triggers (`backend/internal/features/verification/runs/enums.go:13-19`). This change is constrained by `AGENTS.md` and `backend/AGENTS.md`, including their naming, controller-backed testing, formatting, linting, and mandatory review rules.

## Goals / Non-Goals

**Goals:**

- Make the cancellation sweep select only automatic verification runs.
- Encode the selection and cancellation responsibilities in their function names.
- Preserve manual runs in both non-terminal states without changing their enqueue or execution path.
- Preserve cancellation of time-based and after-backup automatic runs.

**Non-Goals:**

- Redesign verification scheduling or cancellation.
- Change authorization, retries, notifications, agent assignment, or stale-run handling.
- Add a database migration, API field, configuration option, or frontend control.

## Decisions

### Select automatic triggers in the repository query

`FindNonTerminalForDisabledConfigs` will be renamed to `GetNonTerminalAutomaticVerificationsForDisabledSchedules` and include an explicit trigger predicate for `SCHEDULED` and `AFTER_BACKUP`. The database will return only rows that satisfy the automatic, non-terminal, and disabled-schedule conditions. The old name will be removed without an alias.

Rejected alternatives:

- Fetch all non-terminal rows and skip `MANUAL` inside the scheduler. This reads unrelated rows and splits the selection rule between the repository and scheduler.
- Select every trigger except `MANUAL`. A future user-initiated trigger would then be canceled by default. An allowlist makes new trigger semantics an explicit decision.
- Name the method `GetAutomaticVerificationsToCancelForDisabledSchedules`. A repository query should describe the rows it returns rather than decide what its caller will do with them.
- Keep the `Find` prefix. The repository method retrieves a defined collection, so the project `Get` convention applies.
- Add `IsAutomatic()` to `VerificationTrigger`. The SQL query still needs the explicit trigger values, so a separate predicate would duplicate the classification for this path instead of replacing it.
- Add a separate configuration flag for manual verification. The requested behavior is that manual runs are independent of automatic scheduling, so another setting would create a second source of truth.

### Name the scheduler action by the transition it performs

`sweepCanceledByDisabledConfig` will be renamed to `cancelAutomaticVerificationsForDisabledSchedules`. The action will continue to mark the selected rows `CANCELED` with the existing disabled-schedule reason. The repository query will select both automatic trigger types in one pass. The old name will be removed without an alias.

Rejected alternatives:

- Keep `sweepCanceledByDisabledConfig`. That name suggests it processes rows that are already canceled, and `Config` does not identify which setting controls the transition.
- Add one sweep for time-based runs and another for after-backup runs. Both have the same disabled-configuration rule and terminal transition, so separate paths would duplicate behavior and tests.
- Stop canceling existing automatic runs and only prevent new ones. That would change established behavior beyond the requested manual-verification exception.

### Exercise the real API setup and a single scheduler sweep in tests

Tests will create the database configuration and manual verification through existing API-backed helpers. Time-based runs will come from the scheduler path, after-backup runs from backup-completion handling, and running states from the agent claim path. Each test will invoke one cancellation sweep directly and inspect the API-visible result. Coverage will include pending and running manual runs plus the two automatic trigger types.

Rejected alternatives:

- Wait for the scheduler ticker. This would make the tests slower and dependent on timing while exercising the same sweep.
- Test only the SQL query in isolation. The backend guidelines prefer controller-backed coverage, and the observable status after a sweep is the relevant contract.

## Risks / Trade-offs

- A future automatic trigger could be omitted from the allowlist. Its introduction must update the cancellation selection and add a disabled-schedule regression case.
- A focused sweep test does not test ticker timing. Existing scheduler wiring already calls the sweep, while the new tests cover the incorrect selection that caused the defect.
- Filtering in the query ties cancellation semantics to persisted trigger values. Those values already define the distinction between manual and automatic runs, so no new coupling is introduced.

## Migration Plan

No schema or data migration is needed. Deploy the backend change normally. Rollback consists of reverting the query change; existing verification rows need no conversion.
