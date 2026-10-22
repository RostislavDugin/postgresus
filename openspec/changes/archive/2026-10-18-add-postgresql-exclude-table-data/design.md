## Context

See proposal.md - Why for the motivation. The requirements are in `specs/postgresql-logical-backup-scope/spec.md`.

The existing whole-table exclusion runs through four places, and the new list has to run through the same ones:

- **Stored settings.** `ExcludeTables` is a `[]string` kept as a comma-joined `exclude_tables` column, converted in `BeforeSave` and `AfterFind` (`backend/internal/features/databases/databases/postgresql/logical/model.go:53-54`, `model.go:68-79`). `Update` and `CopyForNewDatabase` carry it over (`model.go:187-188`, `model.go:211-212`). Normalization lives in `backend/internal/util/namelist`, which splits on commas, new lines and tabs, trims and deduplicates.
- **Dump arguments.** Each normalized entry becomes `--exclude-table=<entry>` (`backend/internal/features/backups/backups/usecases/logical/postgresql/create_backup_uc.go:366-368`). `pg_dump` runs from the client tools of the server's own major version, so every supported version, 12 to 18, accepts `--exclude-table-data`.
- **Privilege check.** When a database is created, updated or has its connection tested, `PostgresqlLogicalDatabase.TestConnection` runs `checkDumpReadPrivileges`, which receives a `DumpFilter` (`model.go:942-945`). `getUnreadableDumpedRelations` drops whole-excluded relations from the list of unreadable tables and sequences (`privileges.go:90-107`). The error text suggests "include schemas and exclude tables" (`privileges.go:153-156`).
- **Size estimate.** `GetRawDbSizeMb` passes the same `DumpFilter` (`model.go:153-156`), and `getDumpedRelationsSizeBytes` skips whole-excluded tables (`size.go:30-44`). `DumpFilter` and the pattern matcher are in `dumpscope.go:34-101`. The matcher does not fully mirror `pg_dump`. Its comment at `dumpscope.go:80` says an unqualified pattern hits the table name in every schema, and `isMatchingRelation` behaves that way (`dumpscope.go:71-77`). `pg_dump` resolves table patterns the way psql does: it runs `RESET search_path` and, for a pattern without a schema part, adds `pg_table_is_visible(c.oid)`, so `logs` matches only a `logs` table visible on the role's default search path. Today this already skews the existing list: an unreadable `archive.logs` passes the privilege check when `logs` is excluded, although `pg_dump` dumps and locks it, and its size is left out of the estimate.

In the frontend, the PostgreSQL form renders "Include schemas" and "Exclude tables" as tag inputs inside Advanced settings (`frontend/src/features/databases/ui/edit/EditPostgreSqlLogicalSpecificDataComponent.tsx:677-732`) and opens that section on its own when either is set (`EditPostgreSqlLogicalSpecificDataComponent.tsx:102-113`). The database page lists them in `ShowPostgreSqlLogicalSpecificDataComponent.tsx:87-99`. The label key `databases.fields.excludeTables` is shared with the MySQL and MariaDB forms and pages, while the PostgreSQL tooltip already has its own key, `databases.edit.postgresql.excludeTablesTooltip`.

Restore needs no change. An archive made with `--exclude-table-data` simply has no `TABLE DATA` entry for the table, and `pg_restore` creates the table empty.

## Goals / Non-Goals

**Goals:**

- One more list that flows through storage, dump arguments, the privilege check, the size estimate, copy and the UI by the same paths as the existing one.
- Labels and tooltips that answer "what will the restored database contain" for each list.

**Non-Goals:**

- No change to restore, to physical backups or to the other engines.
- No validation across the two lists.

## Decisions

### Two separate lists

The model gets `ExcludeTableData []string` with `json:"excludeTableData"`, stored in a new `exclude_table_data` column in the same comma-joined format, and converted, updated and copied next to `ExcludeTables`.

- *Rejected: one list where every entry carries a mode ("whole table" or "data only").* The mode is visible next to each entry, but `excludeTables` would change from a string list to a list of objects. That breaks the API and the stored column, and it needs a row-editor component that matches no other field in the form. Two modes do not justify it.
- *Rejected: one list with a single switch for the whole list.* It cannot express the common case of dropping `tmp_*` completely while keeping `logs` as an empty table.
- *Rejected: a JSON column or a child table for the new list.* The existing list, the included schemas and the MySQL, MariaDB and MongoDB lists all use the comma-joined text column through `namelist`. A different format for one list adds a second code path for the same kind of data.

### The API field keeps its name, only the label changes

`excludeTables` stays the API and Go name of the whole-table list. Only the PostgreSQL label becomes "Exclude whole tables".

- *Rejected: renaming the field to `excludeWholeTables`.* It changes no behavior and breaks every API client and the stored column for a naming gain the user never sees.

### Whole exclusion is skipped by the privilege check, data exclusion is not

`DumpFilter` gains `ExcludeTableDataPatterns`. The size estimate skips a relation matched by either list. The privilege check keeps skipping only whole-excluded relations. The reason is in `pg_dump` itself: it takes `LOCK TABLE ... IN ACCESS SHARE MODE` on every table whose definition it dumps, data or not, and a role without read access fails that lock. The missing-privileges error changes its advice to "narrow the backup with include schemas and whole-table exclusion" and adds that excluding only a table's data still requires SELECT on it.

- *Rejected: skipping data-excluded tables in the privilege check too.* The check would pass and the backup would then fail inside `pg_dump` on the lock, with a less clear message, which is the failure the check exists to prevent.
- *Rejected: checking for any privilege that permits an `ACCESS SHARE` lock (INSERT, UPDATE, DELETE or TRUNCATE also do) for data-excluded tables.* It would admit a few more roles at the cost of a second privilege query, and no one has asked for it.

The size query already counts `pg_total_relation_size`, which includes indexes, so skipping the relation removes its indexes from the estimate as the spec requires.

### Unqualified entries match the way `pg_dump` matches them

The backup keeps passing entries to `pg_dump` unchanged, so what gets excluded does not change for anyone. The Go matcher is fixed to agree with it, for both lists. The two queries built on `dumpedRelationScope` (the size query in `size.go` and `unreadableRelationsQuery` in `privileges.go`) also select `pg_catalog.pg_table_is_visible(c.oid)`, and they run after `RESET search_path` on the same connection, as `pg_dump` does. `relationName` carries that flag, and `dumpTablePattern.isMatchingRelation` requires it for a pattern without a schema part. The comment at `dumpscope.go:80` is corrected.

This also fixes the existing list. The privilege check now reports an unreadable same-named table in another schema, which `pg_dump` would have failed on anyway, and the size estimate now counts such a table, which the backup contains. A backup that used to complete keeps completing, because a table the check now reports would already have broken the lock in `pg_dump`.

- *Rejected: rewriting an unqualified entry to `*.<entry>` before passing it to `pg_dump`, so it really matches every schema as the old comment claimed.* It silently widens what existing databases exclude, so a table that is backed up today would disappear from the next backup. The current tooltip promises nothing about other schemas, and `*.name` already gives that behavior to anyone who wants it.
- *Rejected: leaving the matcher as it is and documenting the gap.* Then the new list would inherit the same disagreement between the size estimate, the privilege check and the actual dump.

### Plain `--exclude-table-data`, partitions matched by their own names

Each normalized entry of the new list becomes `--exclude-table-data=<entry>`, appended after the `--exclude-table` arguments. The same matcher in `dumpscope.go` serves both lists, so the size estimate agrees with what `pg_dump` leaves out.

- *Rejected: `--exclude-table-data-and-children` on PostgreSQL 16 and later.* Naming a partitioned table would then empty its partitions, but only on some server versions, and the size estimate would have to learn partition trees. The existing whole-table list has the same per-name behavior, so the two lists stay consistent, and the tooltip tells the user to match partitions with a pattern such as `events_*`.

### Overlap and foreign keys are handled by the tooltip

A table in both lists is excluded entirely, which is `pg_dump`'s own precedence. Saving such a database succeeds. A backed-up table with a foreign key into an emptied table makes `pg_restore` fail on that constraint. The restore runs without `--exit-on-error` (`backend/internal/features/restores/usecases/postgresql/preexistingpublicschema.go:11-13`), so it reports the ignored errors and fails.

- *Rejected: refusing to save when the lists share an entry.* Overlapping globs such as `log*` and `logs` cannot be detected reliably, so a check would catch only exact duplicates, and the outcome of an overlap is already safe and predictable.
- *Rejected: a save-time query that finds foreign keys from backed-up tables into emptied tables and warns.* It is the most helpful option, but it needs a new query, a warning channel the save endpoint does not have, and its own tests. The usual targets, log and history tables, are rarely referenced. It can be added if users hit the problem.

### Interface keys

The PostgreSQL form and page get their own label keys: `databases.fields.excludeWholeTables` ("Exclude whole tables") and `databases.fields.excludeTableData` ("Exclude table data"). The existing `databases.edit.postgresql.excludeTablesTooltip` gets the new wording, and `databases.edit.postgresql.excludeTableDataTooltip` and `databases.edit.postgresql.excludeTableDataPlaceholder` are added. `databases.fields.excludeTables`, `databases.edit.excludeTablesPlaceholder` and `databases.edit.excludeTablesTooltip` stay for MySQL and MariaDB, and the PostgreSQL form keeps using the shared placeholder "No tables excluded" for the whole-table list. All six dictionaries get the new keys; English is the source of the key set, as `frontend/AGENTS.md` requires.

- *Rejected: renaming the shared `databases.fields.excludeTables`.* MySQL and MariaDB would show "whole tables" with no second list to contrast it with.

### Form layout

The new row goes directly below the whole-table row and copies its markup: a 150px label column, a tag input with `NAME_LIST_TOKEN_SEPARATORS` and `normalizeNameList`, and an info tooltip. It is hidden in restore mode like the other backup settings. "Exclude whole tables" may wrap to two lines in the label column, which the form already tolerates.

- *Rejected: switching the PostgreSQL rows to the shared `AdvancedFieldRowComponent`.* That component stacks the label above the field on narrow screens, unlike the rows around it, so changing only these two rows would make the section inconsistent. Moving the whole form to it is a separate refactoring.

### Constraints from the module rules

- `backend/AGENTS.md`: the migration is generated with `goose create` and filled by hand. Behavior is tested through the HTTP API where it has one, with model-level tests for the size estimate and the privilege check, and the backup-to-restore test goes in `internal/features/tests/logical/postgresql/`. Test names follow `Test_WhatWeDo_WhichConditions_WhatWeExpect`.
- `frontend/AGENTS.md`: no user-facing literals, every new string goes into all six dictionaries, and English strings use a plain hyphen.
- The root `AGENTS.md`: no issue numbers in production code, and no backward-compatibility shim, which this change does not need.

## Risks / Trade-offs

- [A user empties a table that other backed-up tables reference, and the restore fails on the foreign key] → The tooltip warns about it, and the restore error names the constraint. A save-time check remains possible later.
- [A user names a partitioned table and expects its partitions to be emptied] → The tooltip says partitions need their own names or a pattern. The size estimate follows the same rule, so it does not hide the mistake by showing a smaller size.
- [A user wrote `logs` expecting it to cover every schema] → The tooltip says that a bare name reaches only the default search path and that `*.logs` covers every schema. The fixed size estimate and privilege check now show what the backup really contains instead of hiding the difference.
- [A user expects data exclusion to get past the privilege error] → The error says explicitly that data exclusion still needs SELECT and points to whole-table exclusion.
- [The longer label wraps in the 150px column] → Accepted. Shorter wording such as "Skip tables" would weaken the contrast with "Exclude table data".

## Migration Plan

One goose migration adds `exclude_table_data TEXT NOT NULL DEFAULT ''` to `postgresql_logical_databases`. Existing rows get an empty list, so their backups do not change. The down migration drops the column. A rollback to an older build ignores the column, and data-excluded tables are then backed up with their rows again.
