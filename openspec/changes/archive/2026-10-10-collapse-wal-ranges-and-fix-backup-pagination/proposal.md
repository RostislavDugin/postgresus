## Why

The physical backup list cannot show anything past its first 1000 rows. The frontend never requests a second page. Each time the user scrolls to the bottom, it re-requests the whole list from the start with a limit raised by 50, and the backend silently caps that limit at 1000. Once the list reaches the cap, the spinner keeps appearing but no new rows arrive.

Databases that stream WAL hit the cap within hours. Every closed 16 MB WAL segment is its own row, and one reported database produces about 1500 rows a day. The FULL and INCREMENTAL backups the user is looking for get pushed past row 1000, so the type filter is the only way to reach them. The same list is also re-requested in full every second, so a user who has scrolled deep makes the server build and send up to 1000 rows per second.

## What Changes

- The physical backup list collapses consecutive WAL segments into one WAL range row. A range shows its first and last segment names, the segment count, the total size and the time span. A new range starts after a gap in the WAL sequence, after a timeline switch, or where a FULL or INCREMENTAL backup sits between segments in the list. A range that follows a gap is marked as such.
- The newest WAL range grows in place on each refresh. Its last segment, segment count, size and end time update without adding rows.
- **BREAKING** The physical backup list API pages with an opaque cursor instead of `limit`/`offset`. The request takes `cursor`, and the response returns `nextCursor` while more rows exist. The `offset` request parameter and the `total`, `limit` and `offset` response fields are deleted. A WAL row in the response now describes a range, and the per-segment `walFilename` field is deleted.
- **BREAKING** The physical backup list no longer returns individual WAL segments. The backup/restore cycle tests that read segments through this API read them from the WAL segment catalog instead.
- The physical, logical and restore check lists load pages by appending the next page to what is already loaded, instead of re-requesting everything from the start.
- On each refresh tick the three lists re-request only their first page and merge it into the loaded rows. The tick interval stays as it is today.
- The paging, refresh and merge logic moves out of the three components into one shared hook.

## Capabilities

### New Capabilities

- `backup-list-paging`: how the physical backup, logical backup and restore check lists load further pages and stay current while the user scrolls.
- `physical-backup-wal-ranges`: how the physical backup list collapses WAL segments into range rows, where a range ends, and how the newest range updates.

### Modified Capabilities

None.

## Out of scope

- Expanding a WAL range into its individual segments, in the UI or the API.
- A "live" or "streaming" badge on the newest WAL range.
- Hiding WAL by default or changing the default type filter.
- Changing the refresh interval of any list.
- Storing WAL ranges in a table. Ranges are computed per request (see design.md).
- Delete, download or restore actions on a WAL range. WAL rows have none today.
- Cursor paging for the logical backup and restore check APIs. They keep `limit`/`offset` and `total`.
- Changes to the 1000-row cap itself.

## Impact

- Backend: the physical backup list query and DTOs (`backend/internal/features/backups/backups/core/physical/service`, `backend/internal/features/backups/backups/services/physical_service.go`, `backend/internal/features/backups/backups/dto/physical`), its controller tests, and the WAL helpers in `backend/internal/features/tests/physical/postgresql/shared/helpers.go`.
- Frontend: `entity/backups/physical` (API client and models), the three list components (`features/backups/physical`, `features/backups/logical`, `features/verification/runs`), a new hook in `shared/hooks`, and new dictionary keys for the WAL range row in every locale.
- API: the physical backup list endpoint changes shape. The frontend is its only consumer outside the test suite.
- Rules this change answers to: `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`.
