## MODIFIED Requirements

### Requirement: The embedded PostgreSQL instance stays private under the shared runtime account

The embedded PostgreSQL server SHALL run under the same non-root operating-system account as the Databasus application and Valkey. During bootstrap only, peer mapping SHALL allow that account to administer the PostgreSQL role named `postgres`. After bootstrap, Unix-socket database login SHALL be rejected. Loopback TCP connections SHALL require SCRAM authentication with the generated credential, and replication connections to the embedded cluster SHALL be rejected.

The embedded PostgreSQL and Valkey ports SHALL remain unpublished by the image.

#### Scenario: Application account opens the private socket

- **WHEN** the Databasus application account attempts a Unix-socket database login after bootstrap
- **THEN** PostgreSQL authentication rejects the connection

#### Scenario: Shared runtime account performs startup administration

- **WHEN** the shared `databasus` operating-system account connects as role `postgres` during startup
- **THEN** the temporary peer mapping permits the administrative connection

#### Scenario: Loopback TCP connection without the generated password

- **WHEN** any process connects to the embedded PostgreSQL server over loopback TCP without the current generated password
- **THEN** SCRAM authentication rejects the connection

#### Scenario: Application uses the generated password

- **WHEN** Databasus connects to the embedded PostgreSQL server over loopback with the generated password
- **THEN** PostgreSQL accepts the connection

#### Scenario: Replication connection to the embedded cluster

- **WHEN** a client requests a replication connection to the embedded PostgreSQL server
- **THEN** PostgreSQL rejects it

## REMOVED Requirements

### Requirement: PostgreSQL and application accounts cannot share a UID

**Reason**: All bundled processes now use the same non-root operating-system account.

**Migration**: Replace separate PostgreSQL and Databasus identity settings with optional `PUID` and `PGID` settings for the shared `databasus` account.

## RENAMED Requirements

- FROM: `### Requirement: The embedded PostgreSQL instance is isolated from the application account`
- TO: `### Requirement: The embedded PostgreSQL instance stays private under the shared runtime account`
