## 1. Storage command

- [x] 1.1 Add `--test-storage` to the main backend binary; save a unique probe with `LocalStorage.SaveFile`, delete it with `LocalStorage.DeleteFile`, and verify focused command tests pass
- [x] 1.2 Start Valkey before running the storage command, retain explicit Valkey client injection in cache utilities, rate limiters, and publish-subscribe setup, and verify storage failures still stop PostgreSQL and the Databasus application
- [x] 1.3 Detect test mode from the executable name instead of arbitrary command-line arguments, verify `--test-storage` uses production configuration without test database settings, and remove the separate filesystem-check binary and lifecycle-specific model changes

## 2. Container runtime identity and services

- [x] 2.1 Replace the two image accounts and four identity variables with one `databasus` account plus validated optional `PUID` and `PGID`; verify mounted pgdata discovery, automatic, explicit, partial, invalid, and collision cases
- [x] 2.2 Prepare mounted paths without treating failed `chown` or `chmod` as failure, then run real capability checks before PostgreSQL and the Databasus application; verify writable restricted mounts start and inaccessible mounts fail with the documented error
- [x] 2.3 Run PostgreSQL and Valkey under `databasus`, use bootstrap-only peer mapping to role `postgres`, reject runtime socket login, and keep loopback SCRAM; verify process IDs and authentication cases in the container suite
- [x] 2.4 Preserve existing secret, control-file, application-log, nested WAL-queue, PostgreSQL, and deprecated-directory behavior; verify the filesystem-related guard and upgrade fixtures pass
- [x] 2.5 Move runtime startup into `docker/start.sh`, organize it around named functions and a short `main`, and keep the `Dockerfile` focused on image construction and entrypoint declaration

## 3. Filesystem regression suite

- [x] 3.1 Replace only the temporary-to-backup probes with `databasus --test-storage`, retain focused PostgreSQL, socket, and control-file checks, and verify ext4 bind, named-volume, and split-filesystem cases
- [x] 3.2 Cover CIFS group `999` with mode `0770`, fixed file modes, and local temporary storage; verify publication succeeds without required metadata changes
- [x] 3.3 Cover writable root-squashed NFS plus read-only, wrong-owner, and unwritable data-root failures; verify allowed operations start and rejected operations emit the exact actionable error
- [x] 3.4 Add upgrade fixtures for `v3.54.0`, `v3.55.0`, `v3.55.1`, and `v3.56.0`; verify metadata, secret key, application log, backup marker, nested WAL queue, and PostgreSQL cluster survive
- [x] 3.5 Remove tracker-number inventory and two-account assumptions from the harness; verify ShellCheck and Actionlint pass
- [x] 3.6 Run ShellCheck against `docker/start.sh` and verify the copied script in the candidate image is byte-identical and executable

## 4. Documentation and specifications

- [x] 4.1 Update all six Advanced Config pages with the single-account model, `PUID`/`PGID`, automatic selection, failure message, migration note, and English documentation link; verify website lint and build pass
- [x] 4.2 Update Helm README and values wording for the same identity model; verify no removed identity variable remains in user documentation
- [x] 4.3 Synchronize the remaining delta specs into the main specs, leave the local-storage model contract unchanged, and verify `openspec validate --strict` passes

## 5. Final verification

- [x] 5.1 Run backend formatting, focused tests, full tests, and `make lint`; resolve every failure
- [x] 5.2 Build the candidate image and run `make test-filesystems`; resolve every filesystem or upgrade regression
- [x] 5.3 Run the mandatory post-implementation reviewer against the complete diff and module rules, resolve every `CHANGES REQUIRED` finding, and leave the working tree ready to commit
