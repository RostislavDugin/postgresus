## MODIFIED Requirements

### Requirement: Docker services share one configurable runtime identity

The Docker image SHALL run Databasus and embedded PostgreSQL under one non-root operating-system account named `databasus`. Operators MAY set its numeric identity with `PUID` and `PGID`. Each supplied value SHALL be a non-zero decimal Linux ID within the supported system range.

When an override is absent, startup SHALL select that ID independently from the existing PostgreSQL data owner, then the mounted backup or data-root owner and finally `999`. Root ownership SHALL NOT be selected automatically.

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

- **WHEN** `PUID` or `PGID` is empty, non-numeric, zero or outside the supported range
- **THEN** the container exits before starting its services
- **AND** the log identifies the invalid variable

### Requirement: Startup validates storage through required operations

Startup SHALL attempt to prepare the data root, PostgreSQL data, PostgreSQL socket, temporary and backup paths with the selected identity and required modes. A failed ownership or mode change SHALL NOT stop startup when the selected account can still perform every required operation.

Startup SHALL run the main Databasus binary with `--test-storage` under the selected account before starting PostgreSQL or the Databasus application. The storage probe SHALL save a unique file through the configured local storage and remove that file through the same storage path. Startup SHALL also verify the operations required for PostgreSQL data, its socket and Databasus control files. A failed check SHALL stop startup before PostgreSQL or the Databasus application runs. Startup SHALL NOT start or wait for a separate cache service.

#### Scenario: Filesystem denies metadata changes but permits operations

- **WHEN** a mounted filesystem rejects ownership or mode changes but the selected account can perform all required operations
- **THEN** the container starts successfully

#### Scenario: Selected identity creates missing service directories

- **WHEN** container root cannot create a required directory but the selected account can create it
- **THEN** startup creates the directory as the selected account and continues

#### Scenario: Backup publication is not permitted

- **WHEN** the selected account cannot publish a file from the temporary path to the backup path
- **THEN** the container exits before starting the application
- **AND** the English error names the affected path, required operation, effective user and group IDs and `https://databasus.com/advanced-config/#docker-storage-permissions`

#### Scenario: Mounted path is read-only

- **WHEN** any required mounted path is read-only
- **THEN** the container exits before starting PostgreSQL or the Databasus application
- **AND** the log provides the same identity and documentation details

#### Scenario: Data root cannot create control files

- **WHEN** temporary, backup, PostgreSQL data and socket paths are writable but the data root cannot create a control file
- **THEN** the container exits before starting PostgreSQL or the Databasus application
- **AND** the English error identifies the data root and required operation

#### Scenario: Container starts its bundled services

- **WHEN** storage validation succeeds
- **THEN** startup proceeds directly to embedded PostgreSQL and the Databasus application without a cache-service readiness step
