## Context

The image runs Databasus and PostgreSQL under separate operating-system accounts and stores their data below `/databasus-data`. A mount can behave differently from the image filesystem because numeric ownership, root squash, CIFS mapping, and filesystem boundaries are host concerns.

The test boundary is the shipped container. The backend local-storage model does not need a new diagnostic command or a second file-publication abstraction to test that boundary.

## Goals / Non-Goals

**Goals:**

- Run the same real-container filesystem cases locally and in CI.
- Cover open and closed Docker filesystem and permission reports with executable cases or a precise exclusion.
- Keep service identity defaults in the image and allow explicit overrides.
- Fix only defects reproduced by the container matrix.

**Non-Goals:**

- Changing the backend local-storage model, connection checks, or backup publication.
- Adding a storage diagnostic command to the application.
- Moving the Docker entrypoint into a separate production script.
- Building ownership migration state or version-to-version upgrade fixtures.
- Requiring exact ZFS, Proxmox LXC, or user-namespace hosts in GitHub Actions.

## Decisions

### 1. Use one direct container harness

`make test-filesystems` runs `e2e/docker-storage/run.sh`. The harness accepts `DATABASUS_IMAGE`; otherwise the Make target builds `databasus-storage-test:local` from the root Dockerfile.

Each positive case starts the real image, waits for health, and uses `docker exec --user` to create, write, read, and remove a unique file under the account and path being tested. The harness does not call backend storage code. A trap removes containers, volumes, networks, mounts, and temporary directories.

Cases stay sequential because CIFS and NFS fixtures share host mount support and cleanup must remain deterministic.

### 2. Keep the matrix about Docker filesystems and permissions

The portable matrix covers:

- the default data-root bind mount;
- a Docker named volume;
- separate mounts for application files, temporary files, backups, and PostgreSQL data;
- default IDs and free custom IDs;
- readable, writable, read-only, and inaccessible paths;
- a real CIFS share with numeric group mapping;
- a real root-squashed NFS export mounted as application backup storage.

The issue inventory includes open and closed reports. A report belongs in the matrix when it can be reproduced at the shipped container boundary. Reports about remote database grants, remote-storage protocol features, or application behavior outside the container mount boundary get an explicit exclusion.

### 3. Keep production changes small

The Dockerfile remains the production entrypoint source. It declares the four default IDs in image metadata and reads the same four variables when an operator overrides them.

A failing case may justify a narrow entrypoint change, such as avoiding an unnecessary recursive ownership rewrite or accepting a pre-owned root-squashed mount. It does not justify a general ownership state machine, new backend commands, or a new local-storage abstraction.

Default startup must not remap accounts because the image accounts already use the declared IDs. Custom IDs may remap the two service accounts before services start. The Databasus account must not gain access to PostgreSQL private paths.

Filesystem setup keeps the existing `/postgresus-data` and `WAL_V1` startup guards. It also repairs ownership only for the known root-level application files instead of traversing mounted trees.

### 4. Run the candidate locally in CI

The `test-filesystems` job builds and loads one amd64 candidate with `push: false`. It runs independently of backend tests and image smoke jobs. Release and development image publication depend on this job.

The job installs CIFS and NFS clients, uses digest-pinned fixture images, and never publishes the candidate or transfers it between jobs.

### 5. Keep identity documentation in Advanced Configuration

The Dockerfile owns the defaults. `.env.example`, Compose, Helm values, the Helm README, root README files, and installation pages do not repeat them.

The English and five localized Advanced Configuration pages list the four variables, defaults, accepted numeric range, and override use. They describe the current configuration without compatibility history.

## Risks / Trade-offs

- GitHub-hosted runners cannot reproduce every ZFS, Proxmox, or namespace layout. Portable cases cover the numeric ownership and mount behavior behind those reports.
- Network filesystem fixtures are slower than bind mounts and named volumes. They remain in one sequential job with bounded timeouts and cleanup.
- Direct shell operations do not test backend backup publication. Issues #401 and #431 concern application-level transfer policy rather than container permissions, so this change does not alter or duplicate that code.
