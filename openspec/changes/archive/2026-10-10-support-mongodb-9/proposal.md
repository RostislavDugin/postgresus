## Why

Databasus refuses to add a MongoDB 9.x database: the connection test fails with `unsupported MongoDB version: 9.0 (supported: 4.2+, 5.x, 6.x, 7.x, 8.x)`. MongoDB 9.0 is released (the `mongo:9.0.2` image is current), so anyone running it cannot back it up. Accepting the version is not enough on its own. The bundled MongoDB Database Tools are 100.16.1, and server 9.0 support first appeared in 100.18.0, so the client has to move before the server can be accepted.

The same refresh closes a security exception that has already lapsed. `.trivyignore.yaml` silences CVE-2026-56854 for the 100.16.1 binaries, which carry `golang.org/x/crypto` v0.45.0, and that exception expired on 2026-10-05. Database Tools 100.19.1 carry `x/crypto` v0.57.0, past the fixed version, so the exception can be deleted instead of renewed.

## What Changes

- Accept a MongoDB server reporting major version 9 and record it under a new version identity `9`.
- Replace the bundled `mongodump` and `mongorestore` on both architectures with MongoDB Database Tools 100.19.1. Both architectures come from the same upstream build family, `ubuntu2204`, and the download URL and archive checksum of each are recorded in `assets/tools/README.md`. The current binaries match no published 100.16.1 archive we checked, so their origin is unknown today.
- Make the MongoDB restore downgrade guard compare version identities by their numbers, the way the MySQL and MariaDB guards already do. Today it looks the identity up in a table, and an identity missing from that table orders as 0, so forgetting to extend the table would let a MongoDB 9 backup restore onto an 8 server without a word.
- Delete `GetMongodbVersionEnum`, which nothing outside its own file calls. It is a second version table that would need a `9` entry for no caller.
- Correct the frontend MongoDB version values to the identities the backend actually sends (`4` to `9`). The frontend currently declares `4.2`, `4.4`, `5.0` and so on, none of which the backend ever produces. No screen renders the MongoDB version, so nothing visible changes.
- Delete the expired CVE-2026-56854 exception from `.trivyignore.yaml`.
- Add MongoDB 9.0 to the backend test matrices that boot a real server, and update the supported-version lists in the READMEs, `SECURITY.md`, the website's MongoDB page and `llms.txt`.

Nothing here is BREAKING. Stored MongoDB databases keep their recorded identities `4` to `8`, the API shape is unchanged, and the renamed frontend values were never matched against data.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `database-version-support`: gains MongoDB requirements. The spec covers MySQL and MariaDB only, so it gets the accepted MongoDB versions, the refusal of versions below 4.2 and above 9, and the restore downgrade guard for MongoDB.
- `client-tool-bundles`: the requirement that every accepted version identity has a designated client on every architecture is extended to MongoDB, whose single Database Tools bundle serves every accepted server version and is held to the vendor's compatibility statement rather than to release lines. A new security requirement makes the shipped MongoDB tools traceable: they are byte-identical to a vendor archive whose address and checksum the client documentation records, and both architectures come from the same release and build family.

## Out of scope

- A version check for the MongoDB bundle at startup. The bundle is not named after a server release line, so the startup check only confirms the tools start; adding a pinned Database Tools version check is a separate decision.
- A refresh script for the MongoDB bundle like the MariaDB one. Each architecture is one upstream archive, so a recorded URL and checksum give the same reproducibility.
- Showing the MongoDB version in the database view.
- Changing the shared MongoDB test container image (`mongo:8.2.3-noble`) or the SSL test, which stay on MongoDB 8.
- Editing accepted ADRs. ADR-0005 and ADR-0013 record the matrix as it was when they were decided.
- Any other engine's bundle or version list.

## Impact

- **Module rules this change answers to:** the root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, `website/AGENTS.md`, `assets/readme/AGENTS.md` and `assets/tools/AGENTS.md`.
- **Binaries:** `assets/tools/{x64,arm}/mongodb/bin/{mongodump,mongorestore}` change for every MongoDB server, not only 9.x, because one bundle serves them all. The whole MongoDB matrix has to pass on the new client.
- **Backend:** `backend/internal/util/tools/mongodb.go`, the version detection in `backend/internal/features/databases/databases/mongodb/model.go`, the restore guard call in `backend/internal/features/restores/service.go` and the MongoDB test matrices.
- **Frontend:** `frontend/src/entity/databases/model/mongodb/MongodbVersion.ts` and the one test that uses it.
- **CI:** the blocking Trivy image scan no longer depends on the expired exception.
- **Main spec Purpose:** `openspec/specs/database-version-support/spec.md` names only MySQL and MariaDB in its Purpose, which a delta cannot change, so its Purpose line is updated directly to include MongoDB.
- **Docs:** root `README.md` and the five copies under `assets/readme/`, `.github/SECURITY.md`, `assets/tools/README.md`, `website/public/llms.txt`, and the website MongoDB page with its five translations.
