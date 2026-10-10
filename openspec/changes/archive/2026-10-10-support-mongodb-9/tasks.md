## 1. MongoDB client bundle

The arm64 binaries cannot run on an amd64 host without emulation. Their version is read with `go version -m` and `strings` instead, and the product's startup check runs them on a real arm64 build.

- [x] 1.1 Download `https://fastdl.mongodb.org/tools/db/mongodb-database-tools-ubuntu2204-x86_64-100.19.1.tgz` and `https://fastdl.mongodb.org/tools/db/mongodb-database-tools-ubuntu2204-arm64-100.19.1.tgz` into a new empty directory outside the repository. Verify that `sha256sum` of each archive equals the `sha256` listed for it in `https://downloads.mongodb.org/tools/db/release.json` (`8312f251c360ddfbcdd430c64758bee58b7050e1041e8c9df2906761098acd49` for x86_64, `e5d37e679ff815c45a86c17bc17de200c3a215a6eafc82516e5ba271f563373e` for arm64)
- [x] 1.2 Copy `bin/mongodump` and `bin/mongorestore` from each archive over `assets/tools/x64/mongodb/bin/` and `assets/tools/arm/mongodb/bin/` without modifying them. Verify that `sha256sum` of each copied file equals the file inside its archive
- [x] 1.3 Verify the x64 pair starts in the runtime image and reports `100.19.1`: `docker run --rm --platform linux/amd64 -v "$PWD/assets/tools:/tools:ro" debian:bookworm-slim bash -c 'apt-get update -qq && apt-get install -y -qq libgssapi-krb5-2 >/dev/null && /tools/x64/mongodb/bin/mongodump --version && /tools/x64/mongodb/bin/mongorestore --version'`
- [x] 1.4 Verify the arm pair without running it: `go version -m` shows `golang.org/x/crypto` v0.57.0, `strings` finds `100.19.1`, `objdump -T` shows no `GLIBC_2.x` above 2.34, and `objdump -p` lists no `NEEDED` library beyond `libc.so.6`, `ld-linux-aarch64.so.1` and `libgssapi_krb5.so.2`
- [x] 1.5 Verify all four files are executable in the working tree and that git sees no mode change: `stat -c '%a %n' assets/tools/{x64,arm}/mongodb/bin/*` shows `755` for every file, and `git diff --summary -- assets/tools/x64/mongodb assets/tools/arm/mongodb` prints no `mode change` line
- [x] 1.6 In `assets/tools/README.md`, replace the 100.16.1 paragraph with a MongoDB source table that lists the tools release, the build family `ubuntu2204`, and the download URL and archive `sha256` for each architecture. Change the server range to 4.2 to 9.0, as the vendor states for 100.19.1. Verify by checking that the table's checksums match the ones verified in 1.1
- [x] 1.7 Delete the CVE-2026-56854 entry and the now-empty `vulnerabilities:` key from `.trivyignore.yaml`, keeping the file's final newline. Verify with `trivy rootfs --scanners vuln --severity CRITICAL --exit-code 1 assets/tools/x64/mongodb` and the same for `assets/tools/arm/mongodb`, if `trivy` is installed; otherwise verify with the `go version -m` output from 1.4 that `x/crypto` is at or above v0.55.0

## 2. Backend version identity and restore guard

- [x] 2.1 Add `MongodbVersion9` to `backend/internal/util/tools/mongodb.go`, delete `GetMongodbVersionEnum`, and update the comment on `GetMongodbExecutable` to the 4.2 to 9.x range. Verify with `go build ./...` in `backend/`
- [x] 2.2 Rewrite `mapMongodbVersion` in `backend/internal/features/databases/databases/mongodb/model.go` to take the whole version string from `buildInfo`, as design D3 describes, and make `detectMongodbVersion` pass that string straight through. The function parses the leading `major.minor` as integers, refuses anything below 4.2 or above one named upper-bound constant set to 9, and otherwise returns the major as the identity. Update the comment that names the tools release. Rewrite `Test_MapMongodbVersion_VersionMatrix_ReturnsExpected` to take a version string per case, and rename its `got`/`wantErr`/`wantVersion` fields to domain names as `backend/AGENTS.md` asks of touched tests. The table holds:
  - accepted: `4.2.0` and `4.4.29` → `MongodbVersion4`, `5.0.0` → `MongodbVersion5`, `6.0.0` → `MongodbVersion6`, `7.0.0` → `MongodbVersion7`, `8.0.0` and `8.10.1` → `MongodbVersion8`, `9.0.2` and `9.2.0` → `MongodbVersion9`. The existing `9.0 rejected` case flips to accepted;
  - refused below the minimum: `3.6.23`, `4.0.28` and `4.1.0`, each with an error containing the version found (`3.6`, `4.0`, `4.1`) and `minimum supported: 4.2`;
  - refused above the bound: `10.0.0`, with an error containing `10.0` and `supported: 4.2 through 9.x`;
  - unreadable: `abc` and an empty string, each with an error containing `could not parse MongoDB version` and the value received.

  Verify with `just test-package ./internal/features/databases/databases/mongodb/...` in `backend/`
- [x] 2.3 Make `IsMongodbBackupVersionHigherThanRestoreVersion` return `(bool, error)` and delegate to `compareReleaseLines`, like the MySQL guard. Update the caller in `backend/internal/features/restores/service.go` to return the error. Add `backend/internal/util/tools/mongodb_test.go` covering 9 onto 8 refused, 8 onto 9 allowed, 9 onto 9 allowed, 4 onto 9 allowed, and an unreadable identity returning an error, named after the MySQL tests in `mysql_test.go`. Verify with `just test-package ./internal/util/tools/... ./internal/features/restores/...` in `backend/`
- [x] 2.4 Update the fake client output in `backend/internal/util/tools/common_test.go` from `100.16.1` to `100.19.1`. Verify with `just test-package ./internal/util/tools/...` in `backend/`

## 3. Real-server test matrices

- [x] 3.1 Add `{"MongoDB 9.0", tools.MongodbVersion9, "mongo:9.0.2-noble"}` to `mongodbModelVersions` in `model_test.go` and to `mongodbVersions` in `backend/internal/features/tests/logical/mongodb/backup_restore_test.go`, then pull the image with `docker pull mongo:9.0.2-noble`. Verify with `just test-package ./internal/features/databases/databases/mongodb/...` in `backend/`
- [x] 3.2 Run the whole MongoDB backup and restore matrix on the new binaries, because the refreshed client serves every version and not only 9. Verify with `just test-package ./internal/features/tests/logical/mongodb/...` in `backend/`

## 4. Frontend identity values

- [x] 4.1 Change `frontend/src/entity/databases/model/mongodb/MongodbVersion.ts` to the identities the backend sends, `MongodbVersion4 = '4'` through `MongodbVersion9 = '9'`, and update the member used in `disableSrvWhenTunneled.test.ts`. Verify with `pnpm typecheck`, `pnpm lint` and `pnpm test` in `frontend/`

## 5. Documentation

- [x] 5.1 Add 9 to the MongoDB line in the root `README.md` and in all five copies under `assets/readme/`, following each locale's own list punctuation (the Chinese copy uses `、`), as `assets/readme/AGENTS.md` requires. Verify with `grep -n 'MongoDB' README.md assets/readme/README.*.md` showing 9 on line 118 of every file
- [x] 5.2 Add 9 to `.github/SECURITY.md` and to the MongoDB entry in `website/public/llms.txt`. Verify with `grep -n 'MongoDB' .github/SECURITY.md website/public/llms.txt`
- [x] 5.3 Add 9 to every version list on the English MongoDB page `website/app/(en)/mongodb-backup/page.tsx`: the feature list, both FAQ answers, the versions paragraph, and a `MongoDB 9` badge after `MongoDB 8`. Verify with `pnpm lint` in `website/`
- [x] 5.4 Mirror 5.3 in the five copies under `website/app/[lang]/mongodb-backup/content/`, following each locale's conventions in `website/AGENTS.md` and keeping heading anchors unchanged. Verify that `grep -nE '7 (и|y|e|et|and) 8|7 和 8' website/app/\(en\)/mongodb-backup/page.tsx website/app/\[lang\]/mongodb-backup/content/*.tsx` finds no list that still stops at 8, then run `pnpm lint` and `pnpm build` in `website/`

## 6. Close-out

- [x] 6.1 Update the Purpose line of `openspec/specs/database-version-support/spec.md` to name MySQL, MariaDB and MongoDB, since a delta spec cannot change it. Verify with `openspec validate support-mongodb-9 --strict`
- [x] 6.2 Run `just lint` at the repository root and `just format` for the touched modules, then confirm `git status` shows no formatter changes outside this change's files
- [x] 6.3 Confirm that `grep -rn '100\.16\.1' --exclude-dir=openspec --exclude-dir=node_modules .` and `grep -rnE 'MongodbVersion(42|44|[5-8]0)\b' frontend/src` find nothing, so no stale tools version or old frontend member remains
