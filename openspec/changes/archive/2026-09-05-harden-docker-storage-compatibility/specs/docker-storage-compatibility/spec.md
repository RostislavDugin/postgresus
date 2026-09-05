## Purpose

Defines the supported Docker account settings and the real-container filesystem tests that prevent storage permission changes from breaking common installation layouts.

## ADDED Requirements

### Requirement: Docker service IDs have image defaults

The Docker image SHALL declare `DATABASUS_PUID=65532`, `DATABASUS_PGID=65532`, `POSTGRES_PUID=999`, and `POSTGRES_PGID=999`. A container SHALL inherit these values without ordinary installation configuration. Operators SHALL be able to override each service account with free numeric IDs accepted by the image.

#### Scenario: Container uses image defaults

- **WHEN** an operator starts the image without identity overrides
- **THEN** the container environment contains the four image defaults
- **AND** the Databasus and PostgreSQL accounts use those IDs

#### Scenario: Operator supplies free custom IDs

- **WHEN** an operator supplies free numeric IDs through the four supported variables
- **THEN** the container starts with each service account using its configured user and primary group IDs

### Requirement: Service accounts remain isolated

The Databasus and PostgreSQL accounts SHALL have distinct user IDs and primary groups. PostgreSQL MAY receive access needed to traverse the shared data root. The Databasus account SHALL NOT gain access to PostgreSQL private data or its private socket.

#### Scenario: Databasus account reads PostgreSQL private paths

- **WHEN** the Databasus account attempts to read the embedded database tree or private socket
- **THEN** filesystem permissions reject the operation

### Requirement: Container preserves incompatible-data startup guards

The container SHALL refuse startup when it detects data at the deprecated Postgresus mount location or a metadata database containing a PostgreSQL configuration with backup type `WAL_V1`.

#### Scenario: Postgresus data directory is mounted

- **WHEN** the deprecated Postgresus mount location exists and contains data
- **THEN** the container exits before starting its services
- **AND** its log identifies the unsupported directory

#### Scenario: WAL_V1 configuration exists

- **WHEN** the metadata database contains a PostgreSQL configuration whose backup type is `WAL_V1`
- **THEN** the container exits before starting the Databasus application
- **AND** its log identifies the unsupported backup type

### Requirement: One filesystem suite runs locally and in CI

The repository SHALL provide `make test-filesystems` as the single entry point for the Docker filesystem suite. It SHALL build or accept a local candidate image and run the same cases locally and in a dedicated GitHub Actions job. The job SHALL build and load its own candidate without publishing or transferring it.

Each positive case SHALL start the real container and perform create, write, read, and remove operations directly under the responsible service account. Each negative case SHALL assert the container or filesystem operation failure. Every case SHALL clean its containers, volumes, networks, mounts, and temporary files.

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

The suite SHALL cover a bind-mounted data root, a Docker named volume, separate application, temporary, backup, and PostgreSQL mounts, default and custom service IDs, rejected permission layouts, CIFS numeric ownership, and NFS root squash. It SHALL test only the current candidate image and SHALL NOT include version-to-version upgrade fixtures.

The repository SHALL keep an inventory of open and closed Docker filesystem and permission reports. Each report that crosses the shipped container boundary SHALL map to an executable case. Each exclusion SHALL state why the report does not exercise that boundary.

#### Scenario: Local Docker storage layouts

- **WHEN** the candidate uses a bind mount, named volume, or separate mounts
- **THEN** the container becomes healthy and both service accounts can perform their required file operations

#### Scenario: CIFS maps numeric ownership

- **WHEN** a real CIFS fixture presents a mounted path with the configured Databasus identity
- **THEN** the container becomes healthy and the Databasus account can perform its required file operations

#### Scenario: NFS backup storage denies ownership changes to container root

- **WHEN** a root-squashed NFS fixture is already writable by the configured service identities
- **THEN** the container becomes healthy with its backup path on NFS
- **AND** both accounts can perform their required file operations on their assigned mounts

#### Scenario: Mounted path denies required access

- **WHEN** a mounted path does not permit an operation required by its service account
- **THEN** the container or the direct account-level probe fails
