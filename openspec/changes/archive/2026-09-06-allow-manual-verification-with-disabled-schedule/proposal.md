## Why

Manual verification is an on-demand action and must not depend on whether automatic verification is enabled. The current scheduler cancellation sweep treats every non-terminal verification for a disabled configuration as automatic, so it cancels manual runs shortly after they are enqueued.

## What Changes

- Keep manual verifications in `PENDING` or `RUNNING` when automatic verification is disabled or becomes disabled.
- Continue canceling non-terminal `SCHEDULED` and `AFTER_BACKUP` verifications when automatic verification is disabled.
- Express the cancellation boundary in the data selection itself: only automatic triggers enter the disabled-schedule cancellation set.
- Rename the repository selection and scheduler action so their names state the automatic-verification rule they enforce.
- Add regression coverage for manual and automatic verification behavior during the cancellation sweep.
- This change is not breaking. It does not remove or alter any API, configuration field, or stored data shape.

## Capabilities

### New Capabilities

- `backup-verification`: Defines how manual and automatic backup verifications behave when scheduled verification is disabled.

### Modified Capabilities

None.

## Impact

The implementation is limited to the backend verification scheduler, its data selection, and adjacent tests. It is governed by `AGENTS.md` and `backend/AGENTS.md`. No frontend, verification agent, database migration, API contract, external dependency, or deployment change is expected.

Out of scope:

- Changing who may enqueue or cancel a manual verification.
- Changing verification retries, agent assignment, or stale-run handling.
- Changing notification behavior or user-facing copy.
- Changing how automatic verification runs are created.
