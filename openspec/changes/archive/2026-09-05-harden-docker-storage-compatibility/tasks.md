## 1. Keep the implementation boundary small

- [x] 1.1 Keep the backend local-storage model and PostgreSQL credential code unchanged. Do not add application diagnostic commands or test-only binaries to the production image.
  Evidence: `git diff --exit-code -- backend` passes, and the image contains no new diagnostic binary.
- [x] 1.2 Keep the production entrypoint inline in the Dockerfile. Do not add external entrypoint scripts, ownership migration state, or version-to-version upgrade fixtures.
  Evidence: the entrypoint remains the `COPY <<EOF /app/start.sh` block in `Dockerfile`; the change adds one test script and no upgrade fixtures. The existing `/postgresus-data` and `WAL_V1` startup guards remain in place.

## 2. Filesystem harness

- [x] 2.1 Add `make build-docker-storage-test-image` and `make test-filesystems`. Make the harness accept `DATABASUS_IMAGE`, select one case through `CASE`, label every Docker resource uniquely, and clean all resources on success or failure.
  Evidence: `Makefile` exposes both targets; `e2e/docker-storage/run.sh` accepts both variables and labels resources with a per-run value. Docker container, volume, and network queries returned no labeled resources after the full run.
- [x] 2.2 Probe mounted paths directly with `docker exec --user databasus` and `docker exec --user postgres`. Verify unique-file create, write, read, and remove operations without calling backend storage code.
  Evidence: `probe_path` performs the four operations through `docker exec --user`; the complete matrix passes.

## 3. Image identities

- [x] 3.1 Declare `DATABASUS_PUID=65532`, `DATABASUS_PGID=65532`, `POSTGRES_PUID=999`, and `POSTGRES_PGID=999` in Docker image metadata. Verify the image environment, running-container environment, effective IDs, default no-remap behavior, and free explicit overrides.
  Evidence: the `default-ids` case also mounts failing commands over `groupmod` and `usermod`, so a redundant default remap stops the container. The `default-ids` and `custom-ids` cases verify image metadata, container environment, and effective account IDs. Both pass.
- [x] 3.2 Keep the four variables out of `.env.example`, default Compose configuration, Helm values, the Helm README, root README files, and installation pages. Document them only in all six Advanced Configuration pages.
  Evidence: repository search finds the four variable names only in `Dockerfile`, the filesystem harness, OpenSpec, and all six Advanced Configuration pages.

## 4. Filesystem and permission matrix

- [x] 4.1 Add healthy cases for a bind-mounted data root, a Docker named volume, and separate application, temporary, backup, and PostgreSQL mounts.
  Evidence: `bind-root`, `named-volume`, and `split-mounts` pass in the complete matrix.
- [x] 4.2 Add rejected cases for read-only and inaccessible paths. Assert container status and a relevant startup or operation failure.
  Evidence: `read-only` observes a nonzero container exit; `inaccessible-path` observes the Databasus account's expected permission denial.
- [x] 4.3 Add real CIFS and root-squashed NFS cases with digest-pinned fixtures and a prerequisite check for host client support.
  Evidence: `cifs` and `nfs-root-squash` pass with digest-pinned server images. `run_network_prerequisites` checks host helpers before either case.
- [x] 4.4 Audit open and closed Docker filesystem and permission issues. Map every container-boundary report to a case and record a precise reason for each exclusion.
  Evidence: the issue inventory in `e2e/docker-storage/run.sh` maps #45, #64, #141, #224, #274, #478, #738, #751, and #763. It excludes #96 and #199 because an arbitrary container user cannot start both services, and excludes #401 and #431 because they concern application transfer behavior rather than container permissions.
- [x] 4.5 Fix only Dockerfile or inline-entrypoint defects reproduced by these cases. Do not change backend local-storage behavior.
  Evidence: the Dockerfile declares the four IDs, skips default remapping, uses shallow best-effort ownership preparation for pre-owned mounts and the known root-level application files, and fixes process `TMPDIR` at `/tmp`. No backend file changed.

## 5. CI and final verification

- [x] 5.1 Add an independent `test-filesystems` GitHub Actions job that builds and loads its own candidate without publishing it. Pin every Action to a full commit SHA with a version comment, retain top-level `permissions: contents: read`, and require the job before release and development image publication. Verify the workflow with Actionlint and direct permission and dependency inspection.
  Evidence: Actionlint passes. The job uses full-SHA Actions, `push: false`, `load: true`, and its own local tag. Top-level permissions remain `contents: read`; both publication paths depend on the job.
- [x] 5.2 Run the targeted cases and the complete `DATABASUS_IMAGE=databasus-storage-test:local make test-filesystems` matrix. Run Dockerfile, shell, Compose, Helm, workflow, website, and OpenSpec validation, confirm zero labeled Docker resources remain, and obtain final reviewer `PASS`.
  Evidence: the rebuilt image passes the full matrix and the direct `/postgresus-data`, root-file ownership, and `WAL_V1` checks. The test runs leave no labeled resources. Bash syntax, ShellCheck, Docker build checks, Compose, Helm, Actionlint, website lint/build, OpenSpec validation, and the final reviewer all pass.
