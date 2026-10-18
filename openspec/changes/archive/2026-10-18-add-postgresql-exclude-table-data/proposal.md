## Why

A PostgreSQL logical backup can only drop a table completely: the "Exclude tables" setting removes its structure and its rows, so the table is missing after a restore. People with large log, history or archive tables want the opposite trade. They do not need those rows in the backup, but the application expects the tables to exist, with their indexes and constraints, once the database is restored. `pg_dump` supports this with `--exclude-table-data`, and Databasus does not expose it.

Adding a second list next to "Exclude tables" only helps if the two are easy to tell apart. With the current label, "Exclude tables" and a new "Exclude table data" read almost the same. The existing field gets a label that says the whole table goes, and both fields get tooltips that say what the restored database will contain.

## What Changes

- A PostgreSQL logical database gets a second list in its backup settings, "Exclude table data". A table matched by it is backed up with its structure, indexes and constraints but without its rows, so a restore recreates it empty. Entries accept the same names and patterns as the existing list: `table`, `schema.table` and glob patterns such as `logs_*`.
- The existing setting keeps its behavior and its API field, `excludeTables`. In the PostgreSQL form and on the database page its label changes from "Exclude tables" to "Exclude whole tables", and its tooltip says that the table will be missing after a restore.
- The new field sits directly below "Exclude whole tables" in Advanced settings. Its tooltip says the table is restored empty, that a table in both lists is excluded entirely, that the partitions of a partitioned table need their own entries or a pattern, and that the restore fails to recreate a foreign key when other backed-up tables reference rows of an emptied table. The Advanced settings section opens on its own when the new list is not empty.
- The privilege check that runs when a database is saved or its connection is tested keeps requiring read access to a table whose data is excluded, because `pg_dump` still locks every table whose structure it dumps. Only whole-table exclusion lifts that requirement. The error that reports missing privileges says so, instead of suggesting "exclude tables" in general.
- The estimated database size leaves out tables whose data is excluded, as it already does for whole-excluded tables.
- The size estimate and the privilege check match table names the way `pg_dump` does, for both lists. A name without a schema part matches only tables visible on the role's default search path, usually `public`. Until now they treated it as matching every schema, so an unreadable same-named table in another schema passed the check and then failed inside `pg_dump`, and its size was left out of the estimate. What the backup contains does not change. Both tooltips explain the rule and point to `*.name` for every schema.
- Copying a database carries the new list over, as it does the existing one.
- The new label, placeholder and tooltip, the renamed label and the updated tooltip are added to all six interface dictionaries.

Nothing here is BREAKING. Existing databases get an empty "Exclude table data" list and back up exactly as before, and the API only gains an optional `excludeTableData` field.

## Capabilities

### New Capabilities

- `postgresql-logical-backup-scope`: which tables a PostgreSQL logical backup leaves out, either completely or by their rows only, what a restore of such a backup contains, which privileges the backup still requires and how the estimated size accounts for left-out tables.

### Modified Capabilities

None. `logical-backup-outcome` covers whether a logical backup is reported as succeeded or failed, which this change does not affect.

## Out of scope

- Data-only exclusion for MySQL, MariaDB and MongoDB. `mysqldump` and `mongodump` have no flag for it, so each would need its own design. Their "Exclude tables" and "Exclude collections" labels stay as they are, because there is no second field to tell them apart from.
- Physical PostgreSQL backups. They copy the whole cluster and have no table exclusion of either kind.
- A per-entry mode switch in a single list instead of two lists.
- Detecting at save time that a backed-up table has a foreign key into a table whose data is excluded. The tooltip warns about it instead.
- Rejecting a pattern that appears in both lists. `pg_dump` excludes such a table entirely, and the tooltip says so.
- Special handling of sequences owned by an emptied table. They follow `pg_dump`'s own behavior for `--exclude-table-data`.
- Regrouping Advanced settings into sub-sections such as "Backup scope".
- Restore-time selection of which tables to restore.

## Impact

- **Module rules this change answers to:** the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
- **Database:** one migration adds `exclude_table_data TEXT NOT NULL DEFAULT ''` to `postgresql_logical_databases`.
- **Backend:** the PostgreSQL logical database model, its dump filter, the privilege check, the size estimate and their tests in `backend/internal/features/databases/databases/postgresql/logical/`; the `pg_dump` argument builder in `backend/internal/features/backups/backups/usecases/logical/postgresql/`; the database copy test in `backend/internal/features/databases/`; the backup and restore tests in `backend/internal/features/tests/logical/postgresql/`.
- **API:** create, update, read and copy of a PostgreSQL logical database carry an optional `excludeTableData` list.
- **Frontend:** the PostgreSQL logical database type, its edit form and read-only view, and the six dictionaries under `frontend/src/shared/i18n/locales/`.
