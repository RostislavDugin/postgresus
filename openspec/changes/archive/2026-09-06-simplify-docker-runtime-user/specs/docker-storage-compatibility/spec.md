## ADDED Requirements

### Requirement: Docker services share one configurable runtime identity

The Docker image SHALL run Databasus, embedded PostgreSQL, and Valkey under one non-root operating-system account named `databasus`. Operators MAY set its numeric identity with `PUID` and `PGID`. Each supplied value SHALL be a non-zero decimal Linux ID within the supported system range.

When an override is absent, startup SHALL select that ID independently from the existing PostgreSQL data owner, then the mounted backup or data-root owner, and finally `999`. Root ownership SHALL NOT be selected automatically.

#### Scenario: Fresh installation uses the fallback identity

- **WHEN** an operator starts the image without identity overrides or existing non-root ownership
- **THEN** all bundled services run as user ID `999` and group ID `999` under the account name `databasus`

#### Scenario: Existing data determines the identity

- **WHEN** an existing PostgreSQL data directory has a non-root numeric owner and `PUID` or `PGID` is absent
- **THEN** startup uses the corresponding owner ID before considering other mounts or the fallback

#### Scenario: Operator supplies identity overrides

- **WHEN** an operator supplies valid `PUID` and `PGID` values
- **THEN** all bundled services run with those numeric IDs

#### Scenario: Operator overrides one identity value

- **WHEN** an operator supplies either `PUID` or `PGID` and mounted data provides the other numeric ID
- **THEN** startup combines the explicit value with the automatically selected counterpart

#### Scenario: Operator supplies an invalid identity

- **WHEN** `PUID` or `PGID` is empty, non-numeric, zero, or outside the supported range
- **THEN** the container exits before starting its services
- **AND** the log identifies the invalid variable

### Requirement: Startup validates storage through required operations

Startup SHALL attempt to prepare the data root, PostgreSQL data, PostgreSQL socket, temporary, and backup paths with the selected identity and required modes. A failed ownership or mode change SHALL NOT stop startup when the selected account can still perform every required operation.

Startup SHALL start Valkey and wait for readiness before running the main Databasus binary with `--test-storage` under the selected account. The command SHALL save a unique file through `LocalStorage.SaveFile` and remove it through `LocalStorage.DeleteFile`. Startup SHALL also verify the operations required for PostgreSQL data, its socket, and Databasus control files. A failed check SHALL stop startup before PostgreSQL or the Databasus application runs.

#### Scenario: Filesystem denies metadata changes but permits operations

- **WHEN** a mounted filesystem rejects ownership or mode changes but the selected account can perform all required operations
- **THEN** the container starts successfully

#### Scenario: Selected identity creates missing service directories

- **WHEN** container root cannot create a required directory but the selected account can create it
- **THEN** startup creates the directory as the selected account and continues

#### Scenario: Backup publication is not permitted

- **WHEN** the selected account cannot publish a file from the temporary path to the backup path
- **THEN** the container exits before starting the application
- **AND** the English error names the affected path, required operation, effective user and group IDs, and `https://databasus.com/advanced-config/#docker-storage-permissions`

#### Scenario: Mounted path is read-only

- **WHEN** any required mounted path is read-only
- **THEN** the container exits before starting PostgreSQL or the Databasus application
- **AND** the log provides the same identity and documentation details

#### Scenario: Data root cannot create control files

- **WHEN** temporary, backup, PostgreSQL data, and socket paths are writable but the data root cannot create a control file
- **THEN** the container exits before starting PostgreSQL or the Databasus application
- **AND** the English error identifies the data root and required operation

### Requirement: Existing Docker data remains usable after an image update

The current image SHALL start with persistent data created by versions `v3.54.0`, `v3.55.0`, `v3.55.1`, and `v3.56.0` when the mounted filesystem permits the selected account's required operations. Startup SHALL preserve the metadata database, secret key, application log, existing backup files, resumable nested WAL queues, and the existing PostgreSQL cluster. It SHALL prepare an existing WAL queue recursively and verify directory creation plus non-mutating file opens under the selected account before PostgreSQL or the Databasus application starts.

#### Scenario: Existing installation is updated

- **WHEN** an operator replaces one of the covered earlier images with the current image without adding identity settings
- **THEN** the current container becomes healthy
- **AND** the metadata database, secret key, application log, existing backup, nested WAL queue, and PostgreSQL data remain available

## MODIFIED Requirements

### Requirement: One filesystem suite runs locally and in CI

The repository SHALL provide `make test-filesystems` as the single entry point for the Docker filesystem suite. It SHALL build or accept a local candidate image and run the same cases locally and in a dedicated GitHub Actions job. The job SHALL build and load its own candidate without publishing or transferring it.

Each positive case SHALL start the real container and invoke `databasus --test-storage`, which uses the public local-storage save and delete methods. Each negative case SHALL assert the startup failure and its actionable error. Every case SHALL clean its containers, volumes, networks, mounts, and temporary files.

#### Scenario: Developer runs the complete suite

- **WHEN** a developer runs `make test-filesystems` without an image reference
- **THEN** the target builds a local candidate and tests it

#### Scenario: CI tests its candidate

- **WHEN** the filesystem job builds and loads its candidate
- **THEN** the suite tests that local image without pushing or pulling a candidate image

#### Scenario: A case allocates Docker resources

- **WHEN** the case succeeds or fails after resource allocation
- **THEN** the harness removes every resource created by that case

### Requirement: The matrix covers supported mount behavior

The suite SHALL cover an ext4 bind-mounted data root, a Docker named volume, separate temporary, backup, and PostgreSQL mounts, automatic and explicit runtime IDs, rejected backup and data-root permission layouts, CIFS numeric ownership and forced file modes, writable NFS with root squash, and upgrades from the covered earlier images. It SHALL verify one non-root numeric identity for all bundled services, bootstrap and runtime PostgreSQL authentication, and restrictive modes for generated credential files.

#### Scenario: Local Docker storage layouts

- **WHEN** the candidate uses an ext4 bind mount, named volume, or separate mounts
- **THEN** the container becomes healthy and the storage command saves and deletes its probe

#### Scenario: Temporary and backup paths use different filesystems

- **WHEN** the temporary and backup paths have different device IDs
- **THEN** `LocalStorage.SaveFile` succeeds through its supported cross-filesystem path and `LocalStorage.DeleteFile` removes the probe

#### Scenario: CIFS maps numeric ownership

- **WHEN** a real CIFS fixture presents root ownership, numeric group ownership `999`, directory mode `0770`, and mount-enforced file modes
- **THEN** the container becomes healthy and completes the storage command
- **AND** startup skips the root user ID and writes through group `999`

#### Scenario: NFS data storage denies ownership changes to container root

- **WHEN** a root-squashed NFS data root has no backups directory, rejects ownership and mode changes, remains writable by the selected identity, and local mounts provide PostgreSQL and temporary storage
- **THEN** the selected identity creates the backups directory and the container becomes healthy
- **AND** the storage command succeeds without requiring ownership changes

#### Scenario: Mounted path denies required access

- **WHEN** a mounted path does not permit an operation required by the selected identity
- **THEN** startup fails with the documented actionable error

## REMOVED Requirements

### Requirement: Docker service IDs have image defaults

**Reason**: One runtime account and automatic mounted-data discovery replace four independent image defaults.

**Migration**: Remove the four old variables. Set `PUID` and `PGID` only when automatic selection does not fit the mounted filesystem.

### Requirement: Service accounts remain isolated

**Reason**: Bundled services now use one non-root operating-system account.

**Migration**: Set `PUID` and `PGID` only when automatic selection is unsuitable.
