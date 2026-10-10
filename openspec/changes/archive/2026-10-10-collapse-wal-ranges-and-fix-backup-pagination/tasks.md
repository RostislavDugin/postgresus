## 1. Backend: grouped list with a cursor

- [x] 1.1 Replace `ListBackups` in `core/physical/service/service.go` with the CTE chain from design D1 (gap flag over committed WAL, filter, time-ordered grouping, range aggregation, keyset page of `limit + 1`). Extend `PhysicalBackupListRow` with the range fields and the sort time, and delete `CountBackups`. Verify with `go build ./...` from `backend/`.
- [x] 1.2 Change the physical list DTOs and service per design D2: the request takes `cursor` instead of `offset`; the response carries `backups`, `totalUsageMb` and `nextCursor`; `PhysicalBackupListItem` drops `walFilename` and gains `fromWalFilename`, `toWalFilename`, `walSegmentsCount` and `hasGapBefore`. Add cursor encoding and decoding in the service, and update the controller's swagger annotations. Verify with `go build ./...` and `make swagger`.
- [x] 1.3 Rewrite the paging controller tests in `controllers/physical/controller_test.go` for the cursor: following `nextCursor` returns the next rows; a FULL created between two requests causes no repeat or skip; the last page has no cursor; a malformed cursor returns 400; a cursor from another database returns only the requested database's rows; a user without access is rejected with a cursor. Delete the offset and `total` tests. Verify with `go test ./internal/features/backups/backups/controllers/physical/... -count=1`.
- [x] 1.4 Add controller tests for WAL ranges, one per scenario in `specs/physical-backup-wal-ranges/spec.md`: one range for contiguous segments with correct count, size, from/to names and times; a split and `hasGapBefore` at a missing segment; no gap mark on the oldest range or on a new timeline's first range; a split at a timeline switch; a split around a FULL; one range across a FULL when filtering to WAL; a before-date filter that cuts a range; a status filter that hides all WAL; a range that keeps its id and grows after another contiguous segment is added; a range that counts as one row toward the limit. Use `physical_testing.CreateTestWalSegment` with explicit LSNs and `ReceivedAt`. Verify with `go test ./internal/features/backups/backups/controllers/physical/... -count=1`.
- [x] 1.5 Switch `committedWalSegmentsInOrder` and `streamPostFullSegments` in `features/tests/physical/postgresql/shared/helpers.go` to read committed segments through `GetWalSegmentRepository().FindByChainSpan` (design D8), and update their callers in `shared/bodies.go`. Delete `listBackupsByType` if it has no callers left. Verify with `go vet ./internal/features/tests/...`.
- [x] 1.6 Run `make lint` from `backend/` and fix every finding.

## 2. Backend: query cost check

- [x] 2.1 Insert 50 000 committed segments on one timeline plus a FULL every 10 000 segments for a test database, then run `EXPLAIN (ANALYZE, BUFFERS)` on the first-page query and on a cursor page deep in the list. Record the timings and plan shape under Risks in `design.md`. Compare the result with the 100 ms per-request budget and record in `design.md` whether the change ships the on-the-fly query or needs the stored range table from D1.

## 3. Frontend: shared paging

- [x] 3.1 Add `shared/lib/pagedList.ts` (design D4) with `mergeRefreshedFirstPage` and `appendNextPage`, and `pagedList.test.ts` covering deduplication on append and, for the merge: only the first page loaded; new rows at the top with a deep loaded tail; a changed row in place; the first page's last key missing, which falls back to the first page; two ranges merged under the older range's id; an empty first page; no duplicate keys in any result. Export it from `shared/lib/index.ts`. Verify with `pnpm test`.
- [x] 3.2 Add `shared/hooks/usePolledPagedList.ts` (design D4): loaded rows, next cursor, `isLoading`, `isLoadingMore`, `hasMore`, `reload()`, the scroll listener, the refresh interval with an in-flight guard, deduplication on append, and a reset generation that discards stale responses. Export it from `shared/hooks/index.ts`. Verify with `pnpm lint` and `pnpm build`.
- [x] 3.3 Move the paging rules out of the hook into `shared/lib/PagedListController.ts`, a class with no React state that the hook wraps, and cover it with `PagedListController.test.ts`: reload loads the first page; the next page is requested with the stored cursor and appended; nothing is requested after the last page; a refresh requests only the first page and keeps the scrolled tail; a second refresh or next-page request is skipped while one is in flight; a refresh or next page that answers after a reload is dropped; only errors of the current list are reported. Verify with `pnpm test`.

## 4. Frontend: physical backup list

- [x] 4.1 Update `entity/backups/physical`: `PhysicalBackupListItem` gets the range fields and loses `walFilename`; `GetPhysicalBackupsResponse` gets `nextCursor` and loses `total`, `limit` and `offset`; `getPhysicalBackups` takes a cursor instead of an offset. Verify with `pnpm build`, which reports every consumer still using the old shape.
- [x] 4.2 Move `PhysicalBackupsComponent` onto `usePolledPagedList` with a 1 s refresh. Delete `currentLimit`, `loadMoreBackups`, `lastRequestTimeRef` and the component's own scroll and interval effects. Trigger, delete, filter change and database change call `reload()`; cancel refreshes only the first page (design D5). Remove the client `sorter` and `defaultSortOrder` (design D6). The "all loaded" line shows the number of loaded rows. Verify with `pnpm lint` and `pnpm build`.
- [x] 4.3 Render WAL range rows per design D7 and add the `backups.physical.list.walRange` keys to `en.ts` and every other locale. Verify with `pnpm lint`, `pnpm test` (the dictionary placeholder check) and `pnpm build`.
- [x] 4.4 Move the type cell into `PhysicalBackupTypeCellComponent` and cover its rendering with `PhysicalBackupTypeCellComponent.test.tsx`: a WAL range shows the segment count and its first and last segment names; a range after a gap shows the gap tag and one without does not; a single-segment range shows one name; a backup shows only its type badge. Verify with `pnpm test`.

## 5. Frontend: logical backup and restore check lists

- [x] 5.1 Move `LogicalBackupsComponent` onto `usePolledPagedList` with an offset adapter (design D3) and keep its 1 s refresh. Delete its `currentLimit` paging, `loadMoreBackups` and its own scroll and interval effects. Verify with `pnpm lint` and `pnpm build`.
- [x] 5.2 Move `VerificationsComponent` onto `usePolledPagedList` the same way and keep `VERIFICATIONS_REFRESH_MS`. Leave the agent availability polling as it is. Verify with `pnpm lint` and `pnpm build`.

## 6. Verification

- [x] 6.1 Run `pnpm format`, `pnpm lint`, `pnpm test` and `pnpm build` from `frontend/`, and `make lint` from `backend/`. Fix every failure.
