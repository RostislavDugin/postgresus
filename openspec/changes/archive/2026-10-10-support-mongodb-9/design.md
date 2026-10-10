## Context

See proposal.md for why. The current MongoDB support has one client and three version tables:

- One bundle serves every server: `assets/tools/{x64,arm}/mongodb/bin/` holds `mongodump` and `mongorestore` 100.16.1, and `backend/internal/util/tools/mongodb.go:44` checks only that they start, because the bundle is not named after a server line.
- Detection reads `buildInfo`, takes `major.minor` (`backend/internal/features/databases/databases/mongodb/model.go:614`) and maps it through a `switch` on the major (`model.go:640`), refusing 4.0 and 4.1 and anything above 8.
- The restore downgrade guard looks both identities up in a map (`backend/internal/util/tools/mongodb.go:61-71`), called from `backend/internal/features/restores/service.go:306`. A missing entry orders as 0.
- `GetMongodbVersionEnum` (`mongodb.go:76`) is a third table, a `switch` that panics on an unknown major, and nothing outside its own file calls it.
- The MySQL guard already orders identities by their numbers through `compareReleaseLines` (`backend/internal/util/tools/mysql.go:119`, `backend/internal/util/tools/releaseline.go:38`) and returns an error for an identity it cannot read.
- The frontend declares `'4.2'`, `'4.4'`, `'5.0'` … `'8.0'` (`frontend/src/entity/databases/model/mongodb/MongodbVersion.ts:1-8`), while the backend stores and sends `"4"` … `"8"` (`mongodb.go:16-22`). The enum is only a field type; no screen renders the MongoDB version.
- The test matrices that boot real servers cover 5.0, 6.0, 7.0 and 8.2 (`model_test.go:31-35`, `backend/internal/features/tests/logical/mongodb/backup_restore_test.go:60-65`). MongoDB 4.x has no real-server test.

The approach is constrained by `assets/tools/AGENTS.md` (bundle provenance, both architectures from one source, verification in the runtime image, no version tables, identity constants on both sides, whole-matrix runs after a refresh), ADR-0005 (binaries are committed, not installed in the `Dockerfile`), `backend/AGENTS.md` (test naming, almost no comments) and `frontend/AGENTS.md` (one type per file in `model/`).

## Goals / Non-Goals

**Goals:**

- A MongoDB 9.x server is accepted, backed up and restored with a client whose vendor compatibility statement lists the 9 line.
- Adding the next MongoDB major takes one bound and one constant per side, with no table that silently misorders a forgotten entry.
- The shipped MongoDB binaries can be traced to a vendor archive and checksum.

**Non-Goals:**

- Holding the MongoDB bundle to a tools version at startup.
- Adding MongoDB 4.x servers to the real-server matrices.

## Decisions

### D1. Ship Database Tools 100.19.1

The vendor's compatibility statement for 100.19.1 lists server 9.0, 8.0 to 8.3, 7.0, 6.0, 5.0, 4.4 and 4.2, which covers every identity we accept. It is the newest release, and the `client-tool-bundles` spec asks for the newest patch of a maintained line. It also carries `golang.org/x/crypto` v0.57.0, past the fix the CVE-2026-56854 exception was waiting for.

Rejected: 100.18.0, the first release with 9.0 support. It is already superseded and gains nothing over 100.19.1. Staying on 100.16.1 and accepting 9 anyway is rejected because 100.16.1 predates 9.0 support, and the product never dumps a server whose major line the shipped client's compatibility statement does not list.

### D2. Take both architectures from the `ubuntu2204` build

The vendor builds the tools per distribution. `ubuntu2204` exists for both `x86_64` and `arm64`, needs glibc 2.34 at most (the runtime image has 2.36), and links only `libc` and `libgssapi_krb5.so.2`, which the image already carries (`assets/tools/README.md`, shared-library table). Using one build family for both architectures follows the rule at `assets/tools/AGENTS.md:107`.

Rejected: `debian12`, the build closest to the `debian:bookworm-slim` runtime. It has no `arm64` archive, so arm would come from a different family, which is exactly the mixing the rule forbids. Also rejected: `ubuntu2404`, which works equally well today but gains nothing, while an older build base keeps the library floor lower.

The archive checksums are taken from the vendor's `release.json` and the download is checked against them before the binaries are copied. The URL and checksum of each archive go into `assets/tools/README.md`, next to the MySQL and MariaDB source tables.

### D3. Map the detected version by its numbers, with one upper bound

`mapMongodbVersion` takes the whole version string `buildInfo` reports, instead of the major and minor that `detectMongodbVersion` cuts out with a regular expression today (`model.go:625-633`). It parses the leading `major.minor` as integers, refuses anything below 4.2 and anything above a single constant naming the newest supported major, 9, and otherwise returns the major as the identity. Moving the parsing into the function puts all three refusals behind one entry point that a table test covers without a server:

- an unreadable value: `could not parse MongoDB version: <value>`;
- below 4.2, including 3.6: `unsupported MongoDB version: 4.0 (minimum supported: 4.2)`;
- above the bound: `unsupported MongoDB version: 10.0 (supported: 4.2 through 9.x)`.

The upper bound is a major line, not the exact releases the vendor lists. The vendor names 9.0 for 100.19.1, and a 9.2 server is still accepted, because the `database-version-support` spec treats a newer release inside a covered line as a gap closed by refreshing the client.

Rejected: keeping the `(major, minor string)` signature. The unreadable-value refusal would then stay in `detectMongodbVersion`, reachable only through a live server. Also rejected: adding `case "9"` to the existing `switch`. That is the version table `assets/tools/AGENTS.md:74` calls a bug, and it keeps the error message as a hand-written list that drifts from the cases. Also rejected: no upper bound at all, accepting any future major. The bundled tools would then dump a server the vendor has not declared support for, which the `database-version-support` spec forbids.

### D4. Order MongoDB identities through `compareReleaseLines`

`IsMongodbBackupVersionHigherThanRestoreVersion` becomes `(bool, error)` and delegates to `compareReleaseLines`, like the MySQL guard. `restores/service.go:306` returns the error when an identity cannot be read.

Rejected: adding `9` to the map. A forgotten entry orders as 0, so a guard extended by hand fails open, and this change would leave that trap for the next major. Also rejected: a local `strconv.Atoi` comparison in `mongodb.go`. It would duplicate `releaseline.go` for no gain.

### D5. Delete `GetMongodbVersionEnum`

Nothing calls it, and keeping it means extending one more table that panics on an unknown major.

Rejected: extending it with `9` and keeping it for a possible caller. No caller exists, and the root `AGENTS.md` lists dead code among the things a change cleans up in the code it touches.

### D6. Frontend values match the backend identities

`MongodbVersion` becomes `MongodbVersion4 = '4'` through `MongodbVersion9 = '9'`, and the one test that uses a member is updated. This follows the `assets/tools/AGENTS.md` rule that the backend and frontend constants name the same value.

Rejected: adding only `MongodbVersion90 = '9.0'`. It would extend a type that matches nothing the backend sends. Also rejected: replacing the enum with `string`. That drops the build-time check the rule relies on.

### D7. Pin the test image to `mongo:9.0.2-noble`

The new entry follows the newest one, `mongo:8.2.3-noble`, which pins a patch, so a run does not change when Docker Hub moves a tag. The older entries (`mongo:5.0`, `mongo:6.0`, `mongo:7.0`) float and stay as they are. `just pull-testcontainers` collects images from the source, so the new one is pulled without further changes.

Rejected: `mongo:9.0`, which floats with each patch release.

### D8. Delete the expired Trivy exception

The exception at `.trivyignore.yaml:12-23` covered the 100.16.1 binaries and expired on 2026-10-05. The new binaries carry the fixed `x/crypto`, so the exception goes away with the binaries it described.

Rejected: renewing it with a new date. The finding it silenced no longer applies.

### D9. Leave accepted ADRs as they are

ADR-0005 and ADR-0013 list MongoDB versions as they stood when those decisions were made. Updating them would rewrite the record rather than document current behavior; the current list lives in the READMEs and in `assets/tools/README.md`.

## Risks / Trade-offs

- [The bundle refresh changes the client for every MongoDB server, not only 9.x] → Run the whole MongoDB backup/restore and model matrices on the new binaries, as `assets/tools/AGENTS.md:121` requires.
- [MongoDB 4.2 and 4.4 have no real-server test, so a regression there would only show in production] → The vendor's statement for 100.19.1 lists both. Adding 4.x to the matrices is left out of this change, and the gap is stated here so a reviewer can decide otherwise.
- [A binary that runs on a Fedora host can still fail in the runtime image] → Run `--version` for both architectures inside `debian:bookworm-slim` before committing, following the `assets/tools/AGENTS.md` refresh command.
- [A binary committed without the executable bit fails only at runtime] → Check the working tree, not the index, which keeps the old mode until the files are staged: `stat -c '%a'` shows `755` for all four files, and `git diff --summary` prints no mode change.
- [The upper bound must be raised by hand for MongoDB 10] → This is intended. Raising it is the moment someone checks that the bundled tools support the new major, which is the checklist in `assets/tools/AGENTS.md`.

## Migration Plan

No data migration. Stored identities `4` to `8` are unchanged, and the API shape stays the same. Rolling back means reverting the commit, which brings the old binaries and code back together. The old code does not know identity `9`: its connection test refuses a MongoDB 9 server again, and its restore guard orders a stored `9` as 0. A database added as MongoDB 9 before a rollback should therefore be removed or left without restores until support returns.
