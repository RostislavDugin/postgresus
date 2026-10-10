## Context

See proposal.md - Why for the user-visible problem. The current mechanics:

- The physical list component never sends an offset. Scrolling raises `limit` by 50 and re-requests from offset 0 (`frontend/src/features/backups/physical/ui/PhysicalBackupsComponent.tsx:199-230`). The backend caps `limit` at 1000 (`backend/internal/features/backups/backups/services/physical_service.go:615-629`), so the list stops growing at 1000 rows while `hasMore` stays true.
- The same component re-requests every loaded row once a second (`PhysicalBackupsComponent.tsx:302-309`). `LogicalBackupsComponent.tsx:317-322` and `VerificationsComponent.tsx:256-262` copy the pattern. Their backends have no cap (`logical_service.go:156`, `verification/runs/service.go:197`), so they keep working but re-send a list that grows with every scroll.
- The physical list is one SQL query over a UNION of the FULL, incremental and committed WAL tables, ordered by `created_at DESC, id DESC` and paged with `LIMIT/OFFSET` (`backend/internal/features/backups/backups/core/physical/service/service.go:474-536`). `CountBackups` runs a second full pass over the same UNION for `total` (`service.go:542-558`).
- A WAL row's `created_at` is the segment's `received_at`. That value is set when the segment is claimed, not when its upload commits (`backend/internal/features/backups/backups/usecases/physical/postgresql/wal_upload.go:105,151`). Concurrent uploads can therefore order segments by time differently from their LSN order.
- `physical_wal_segments` has a unique index on `(database_id, timeline_id, start_lsn)` and an index on `(database_id, received_at)` (`backend/migrations/20260523120000_add_physical_pg_17_backups.sql:289-294`).
- WAL rows already render no actions (`PhysicalBackupsComponent.tsx:331`).
- The backup/restore cycle tests read individual WAL segments through the list API with `type=WAL&limit=1000` (`backend/internal/features/tests/physical/postgresql/shared/helpers.go:452-686`). The same package already reads the WAL catalog directly with `FindByChainSpan` (`shared/sshtunnel.go:131-136`).
- The physical table sorts rows on the client by `createdAt` (`PhysicalBackupsComponent.tsx:437-438`).

Rules this design is constrained by:

- `AGENTS.md`: no unrequested backward compatibility, so the old `offset`/`total` contract is deleted; names state intent.
- `backend/AGENTS.md`: prefer controller tests over unit tests; `features/tests/` only for backup-restore cycles; `time.Now().UTC()`.
- `frontend/AGENTS.md`: FSD import direction and no business logic in `shared/`; one type per model file; no user-facing literals, every locale updated, counts phrased as `Segments: {{count}}`; component section order with hooks below plain functions.

## Goals / Non-Goals

**Goals:**

- Group WAL into ranges in the same SQL query that pages the list, so a page always holds `limit` display rows.
- Make cursor pages stable under inserts at the top.
- One client-side paging, refresh and merge mechanism shared by the three lists.

**Non-Goals:**

- Making the per-request cost independent of the number of WAL segments. The query still reads every committed segment of the database, as `CountBackups` does today.
- Changing the logical backup or restore check APIs.

## Decisions

### D1. Group WAL ranges on the fly in SQL

The list query becomes a chain of CTEs:

```
committed_wal   committed segments, plus a per-segment gap flag:
                lag(end_lsn) OVER (PARTITION BY timeline_id ORDER BY start_lsn)
                is_gap_start = prev_end_lsn IS NOT NULL AND prev_end_lsn <> start_lsn
merged          FULL + INCREMENTAL + committed_wal rows, as today
filtered        merged + the existing outer WHERE (type, status, beforeDate)
grouped         in (created_at, id) order, a new group starts at every
                non-WAL row, and at a WAL row whose predecessor in that order
                is non-WAL, is on another timeline, or which is_gap_start
                group_id = running SUM over those starts
items           one row per group: non-WAL groups pass through; WAL groups
                aggregate into a range
paged           WHERE (sort_time, id) < (cursor) ORDER BY sort_time DESC, id DESC
                LIMIT limit + 1
```

The gap flag is computed over all committed segments before the filter. A before-date filter keeps only older segments, so this does not change any flag a visible segment carries. It also keeps the gap check in LSN order. The time-ordered pass decides only where FULL or INCREMENTAL rows cut a range, so out-of-order `received_at` values cannot split a range by mistake.

A WAL range row projects:

| field | value |
|---|---|
| `id` | id of the range's lowest-LSN segment |
| `startLsn` / `stopLsn` | `MIN(start_lsn)` / `MAX(end_lsn)` |
| `fromWalFilename` / `toWalFilename` | `wal_filename` of the lowest / highest `start_lsn` |
| `walSegmentsCount` | `COUNT(*)` |
| `sizeMb` | `SUM(compressed_size_mb)` |
| `createdAt` / `completedAt` | `MIN(received_at)` / `MAX(received_at)` |
| `hasGapBefore` | `is_gap_start` of the lowest-LSN segment |
| sort time | `MAX(received_at)` |

Using the lowest-LSN segment as `id` keeps a range's identity while it grows at its newest end, which the client merge relies on (D4). Retention deletes the oldest segments, so only the oldest range in the list can change identity, and it sits at the bottom.

Alternatives rejected:

- **A table of ranges maintained on WAL commit.** Reads would cost O(page), but every writer would have to keep it consistent: WAL commit extends or starts a range, retention trims ranges from the old end, orphan cleanup and test deletes split them in the middle. That is much more code on the write path. The on-the-fly query costs the same order as today's `CountBackups`. If the measurement in task 2 shows it too slow, this table is the next step.
- **Grouping in Go over a row stream.** Paging needs the group count before the page boundary is known, so Go would stream rows until `limit + 1` items are built. A long WAL range forces it to read every segment of that range anyway, and the logic splits between SQL and Go.
- **Grouping in the frontend.** A page of 50 rows can be 50 segments of one range, so the client would show one row and still need to page further. The paging bug would return in a new form.
- **Splitting ranges by LSN anchors (a backup's `start_lsn`) instead of list order.** The cut would sit at a recovery-meaningful point, but it would not match what the user sees. A failed FULL, or a filter that hides backups, would leave the visible order and the range boundaries disagreeing.

### D2. Keyset cursor instead of offset for the physical list

The request takes `cursor` and `limit`. The response returns `backups`, `totalUsageMb` and `nextCursor`, which is omitted on the last page. The server fetches `limit + 1` items to know whether a next page exists. The `offset` request parameter, the `total`, `limit` and `offset` response fields, and `CountBackups` are deleted.

The cursor is base64url-encoded JSON of the last item's sort time (RFC 3339 with nanoseconds) and id. It is opaque to the client. The service decodes it and binds both values as query parameters. A cursor that fails to decode returns an error, which the controller already maps to 400 (`backend/internal/features/backups/backups/controllers/physical/controller.go:91-95`). The cursor carries no database id. The database still comes from the path and passes the existing `GetDatabase` permission check before any query runs, so a foreign cursor only sets a position.

Alternatives rejected:

- **Offset over grouped items.** The newest range grows and new backups appear at the top. Offsets shift between requests and pages repeat or skip rows.
- **Explicit `afterTime`/`afterId` query parameters.** They are as safe, but they expose the sort key as API surface and clash in meaning with the existing `beforeDate` filter.
- **Keeping `total`.** Counting grouped items requires a second pass over the whole grouping. The UI needs only `hasMore`, and when everything is loaded the loaded row count is the total.

### D3. Logical backup and restore check lists keep offset paging

Both APIs keep `limit`, `offset` and `total`. The client asks for `offset = number of rows already loaded` and drops rows whose id it already holds. When a refresh has added `k` new rows at the top, the next page starts `k` rows early and repeats those rows. Deduplication removes them. Retention deletes the oldest rows, so it never shifts an offset above the deleted rows. A manual delete reloads the list (D5).

Alternative rejected: **cursor paging for these APIs too.** It would make the three lists uniform, but it would change two more APIs for a case that deduplication already covers.

### D4. One shared hook for paging, refresh and merge

`shared/lib/PagedListController.ts` owns the loaded rows, the next cursor, the loading flags, the in-flight guards and a reset generation counter. It holds no React state, so vitest covers the paging rules in the node environment the project tests run in, without a DOM library. `shared/hooks/usePolledPagedList.ts` wraps it and adds the scroll listener and the refresh interval. Each component passes:

- `fetchPage(cursor)`, returning `{ items, nextCursor? }`. The physical adapter forwards the server cursor. The offset adapters use the offset as the cursor and derive `nextCursor` from `total`.
- `getItemKey(item)`.
- `refreshIntervalMs`, `scrollContainerRef`, and the values whose change resets the list (database id, filters).

The hook returns the rows, `isLoading`, `isLoadingMore`, `hasMore` and `reload()`.

The merge of a refreshed first page (`mergeRefreshedFirstPage`) and the append of a next page with deduplication (`appendNextPage`) are pure functions in `shared/lib/pagedList.ts`, unit-tested with vitest:

```
loaded:  [a b c ... x | y z ...]        firstPage: [n a' b' ... x']
                    ^ last key of firstPage found at index i
result:  firstPage ++ loaded[i+1:] without keys already in firstPage
not found, or firstPage is the whole list -> result = firstPage, cursor = its cursor
```

When the first page's last key is found, the stored next cursor stays. When it is not found, the hook falls back to the first page alone and takes its cursor. This handles two rare cases: deletions that empty the loaded prefix, and two ranges merging because a late segment closed a gap. In the second case the merged range keeps the older range's id, so the key is normally found. After a reset, a response with a stale generation is discarded, which replaces the `lastRequestTimeRef` checks in the components.

Both modules are infrastructure with no domain knowledge, so they belong in `shared/`. The three consumers are slices on the same `features` layer and cannot import from each other.

Alternatives rejected:

- **Fixing three copies in place.** The merge logic is the part most likely to be wrong, and three copies would drift.
- **A react-query or SWR infinite query.** It would add a dependency and a caching model the rest of the frontend does not use.

### D5. What triggers a reload versus a refresh

A database switch, a filter change, a trigger and a delete call `reload()`, as they do today. A cancel only refreshes the first page, since the cancelled row stays in place. A delete reloads because a physical FULL delete removes its whole chain, deep below the first page.

### D6. Server order is authoritative in the physical table

The client-side `sorter` and `defaultSortOrder` on the Created column are removed. A range's `createdAt` is its first receive time, while the server places it by its last. A client sort over loaded rows would move ranges and could also reorder a partly loaded list.

### D7. How a range row renders

- Type cell: the WAL badge, then `Segments: {{count}}`, then `fromWalFilename → toWalFilename` in small monospace text. A range with `hasGapBefore` gets an amber "Gap before this range" tag.
- Size, Duration and actions cells: unchanged. Duration already renders `completedAt - createdAt`, the range's time span.
- Created cell: the first and the last receive time, with the relative time of the last one.

New dictionary keys go into every locale under `backups.physical.list.walRange`.

### D8. Cycle tests read segments from the catalog

`committedWalSegmentsInOrder` reads segments through `GetWalSegmentRepository().FindByChainSpan(databaseID, timelineID, 0, LSNMax)`, keeps the committed ones and returns them in LSN order, as `getCommittedWalSegmentCount` already does (`shared/sshtunnel.go:131-147`). Its callers switch from list items to the segment model.

Alternative rejected: **a segment-listing API for tests.** It would be a new public endpoint used only by tests, and expanding a range is out of scope.

## Risks / Trade-offs

- [Every list request reads all committed WAL of the database. A database with 1500 segments a day and 30 days of retention holds about 45 000 rows, read once a second per open tab.] → Task 2 measured the query on the PostgreSQL 17 test database with 50 000 contiguous segments on one timeline and 5 FULLs. The first page and a page deep in the list each took about 90 ms as plain queries, and about 125 ms under `EXPLAIN (ANALYZE, BUFFERS)`. The previous list query plus `CountBackups` took about 15 ms on the same data, so the estimate that grouping costs "the same order as `CountBackups`" was wrong by a factor of six. The time goes to CPU in the two window passes and the per-range `ARRAY_AGG ... ORDER BY` sorts. Raising `work_mem` to 32 MB removed the disk spills but did not change the time. The query stays under the 100 ms budget, with little margin, for a request each open tab sends once a second. The user left the call to the implementer, and the change ships this cost: the stored range table from D1 is a separate change, worth planning if databases with much larger WAL catalogs show up.
- [Breaking API change.] → The frontend and the cycle tests are the only consumers and change in the same commit. No shim stays.
- [The merge falls back to the first page in rare cases and drops the loaded tail.] → The user scrolls again. No row is lost or duplicated.
- [Out-of-order `received_at` puts a range's segments in a different time order from their LSN order.] → Range boundaries come from the LSN gap flag and from non-WAL rows, never from the time order of neighboring segments. From/to names and LSNs are taken by LSN, not by time.
- [Offset pages repeat `k` rows after a refresh adds `k` rows.] → Deduplication by key. The cost is a few extra rows per page request.

## Migration Plan

No schema migration. Backend and frontend ship together. Rollback is a revert of the commit.
