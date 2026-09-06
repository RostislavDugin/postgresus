# Internal PostgreSQL Protection Specification

## Purpose

Prevents workspace-controlled PostgreSQL connections and backups from reading the Databasus metadata database or sending another workspace's saved credentials to a caller-controlled server.

## Requirements

### Requirement: Logical backups cannot target the embedded metadata database

The system SHALL reject a logical PostgreSQL source when its database name is `databasus` and any effective host points back to the Databasus container. Local targets include filesystem and abstract Unix sockets, loopback and unspecified IP addresses, supported container-local aliases, libc-compatible numeric IPv4 forms, and an empty entry in a comma-separated host list.

The system SHALL inspect every host in a multi-host value. A remote entry SHALL NOT make a later local entry acceptable.

#### Scenario: Filesystem Unix socket

- **WHEN** a user configures a logical PostgreSQL source with database `databasus` and a filesystem Unix-socket directory as its host
- **THEN** the system rejects the source before opening the connection

#### Scenario: Abstract Unix socket

- **WHEN** a user configures database `databasus` with an abstract Unix-socket host
- **THEN** the system rejects the source before opening the connection

#### Scenario: Loopback address in a host list

- **WHEN** a PostgreSQL host list contains a remote hostname followed by a loopback address and the database is `databasus`
- **THEN** the system rejects the whole source

#### Scenario: Numeric loopback spelling

- **WHEN** a user specifies a loopback address in an abbreviated, integer, octal, or hexadecimal form accepted by the operating-system resolver
- **THEN** the system treats it as local and rejects database `databasus`

#### Scenario: Local application database

- **WHEN** a user configures a local PostgreSQL source whose database name is not `databasus`
- **THEN** the embedded-target rule allows the source

#### Scenario: Remote database with the same name

- **WHEN** a PostgreSQL source is remote and its database name is `databasus`
- **THEN** the embedded-target rule allows the source

### Requirement: Physical backups cannot target the embedded PostgreSQL cluster

The system SHALL reject a physical PostgreSQL source when an effective local host uses the embedded cluster's port. This rule SHALL apply without relying on a database name because a physical backup addresses the cluster rather than one logical database.

#### Scenario: Embedded cluster through a Unix socket

- **WHEN** a user configures a physical backup against a local Unix socket and the embedded PostgreSQL port
- **THEN** the system rejects the source before any physical backup connection is opened

#### Scenario: Different local PostgreSQL instance

- **WHEN** a physical source uses a local address on a port other than the embedded PostgreSQL port
- **THEN** the embedded-target rule allows the source

### Requirement: Remote SSH targets keep their remote meaning

When a database is reached through an SSH bastion on another host, a loopback database address SHALL be interpreted relative to that remote host. A local SSH bastion SHALL NOT relax the embedded-target rules.

#### Scenario: Database bound to loopback on a remote bastion

- **WHEN** SSH tunneling is enabled through a non-local bastion and the PostgreSQL host is a loopback address
- **THEN** the embedded-target rule allows the source

#### Scenario: Local bastion points back to the container

- **WHEN** SSH tunneling names a local bastion and the PostgreSQL target matches the embedded instance
- **THEN** the system rejects the source

### Requirement: Persisted configurations are checked at connection time

The system SHALL apply embedded-target protection immediately before a PostgreSQL connection is opened, including connections created from rows saved before this policy existed. Model validation alone SHALL NOT be the only enforcement point.

#### Scenario: Existing logical row contains a socket target

- **WHEN** a previously saved logical source names the embedded database through a local socket
- **THEN** a backup, connection test, or healthcheck refuses to open that connection

#### Scenario: Existing physical row contains the embedded cluster

- **WHEN** a previously saved physical source names the embedded PostgreSQL cluster
- **THEN** physical connection and backup paths refuse to open that connection

### Requirement: PostgreSQL connection values remain literal

Every user-controlled string used in a PostgreSQL conninfo value SHALL be interpreted as one value. Whitespace, quotes, backslashes, equal signs, and text resembling another conninfo parameter SHALL NOT create or replace connection parameters.

The database name passed to a logical dump SHALL follow the same rule even when the dump tool accepts a conninfo string in place of a plain database name.

#### Scenario: Database name contains a second database parameter

- **WHEN** a database name contains `dbname=databasus` after another value
- **THEN** PostgreSQL receives the complete input as one literal database name
- **AND** the text cannot redirect the connection to the embedded database

#### Scenario: Other credential fields contain conninfo syntax

- **WHEN** a host, username, password, TLS mode, or certificate path contains whitespace or conninfo metacharacters
- **THEN** the connection parser returns the original field value without creating another parameter

### Requirement: Direct connection tests require workspace authorization

The direct connection-test endpoint SHALL require an authenticated user who can manage databases in the relevant workspace. An ad hoc database SHALL identify that workspace. A request referencing a saved database SHALL be authorized against the workspace stored on that database before saved credentials are merged with request fields.

#### Scenario: Ad hoc request without a workspace

- **WHEN** an authenticated user submits an unsaved database without `workspaceId`
- **THEN** the endpoint returns HTTP 400
- **AND** no connection is attempted

#### Scenario: Ad hoc request for an unauthorized workspace

- **WHEN** an authenticated user submits an unsaved database for a workspace where they cannot manage databases
- **THEN** the endpoint returns HTTP 403
- **AND** no connection is attempted

#### Scenario: Saved database from another workspace

- **WHEN** a user references a saved database outside a workspace they can manage and supplies a different host
- **THEN** the endpoint returns HTTP 403 before applying the saved password
- **AND** the system makes no connection to the supplied host

#### Scenario: Authorized workspace member tests a saved database

- **WHEN** a workspace member who can manage databases tests a saved database in that workspace and leaves its password blank
- **THEN** the system may use the saved password and performs the connection test

### Requirement: Direct connection-test failures do not disclose protected records or internal errors

The endpoint SHALL return the same HTTP 403 response for an unknown saved database and a saved database the caller cannot manage. Storage failures while loading a saved target and failures while checking workspace permissions SHALL return distinct, sanitized HTTP 500 messages that do not include the underlying error.

#### Scenario: Unknown saved database ID

- **WHEN** a caller submits an ID that does not identify a saved database
- **THEN** the endpoint returns the same HTTP 403 body used for an unauthorized saved database

#### Scenario: Saved target lookup fails internally

- **WHEN** storage fails while loading the referenced saved database
- **THEN** the endpoint returns HTTP 500 with `failed to load database connection target`
- **AND** the storage error is absent from the response

#### Scenario: Permission lookup fails internally

- **WHEN** the workspace permission lookup fails for either an ad hoc or saved database
- **THEN** the endpoint returns HTTP 500 with `failed to verify database connection permissions`
- **AND** the underlying membership error is absent from the response

### Requirement: Healthchecks use a trusted internal connection path

Scheduled healthchecks SHALL test a database already loaded by the system without pretending to be an end user. The user-facing direct endpoint SHALL NOT expose that trusted path or bypass its workspace checks.

#### Scenario: Scheduled healthcheck

- **WHEN** the healthcheck service loads a saved database and probes it
- **THEN** the probe runs without an end-user principal
- **AND** embedded-target protection still runs before the connection opens

#### Scenario: User-facing direct test

- **WHEN** an HTTP caller requests a direct connection test
- **THEN** the system uses the workspace-authorized path rather than the trusted healthcheck path

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

### Requirement: Internal PostgreSQL credentials are generated at startup

When Databasus uses its embedded metadata database, the container SHALL generate a cryptographically random PostgreSQL password at startup, set it on the internal role, and supply it to the application without logging it. A container restart SHALL replace the previous password. An explicitly supplied external metadata-database connection string SHALL remain unchanged.

An operator-supplied connection string that points back to the embedded database with the previously published fixed password is incompatible with rotation and SHALL NOT be treated as a supported embedded-database configuration.

#### Scenario: Fresh container startup

- **WHEN** a container starts without an explicit metadata-database connection string
- **THEN** it configures the embedded role and the application with the same generated password
- **AND** the application becomes healthy without printing that password

#### Scenario: Container restart

- **WHEN** the same container restarts
- **THEN** the embedded role receives a different generated password
- **AND** the application reconnects with the new password

#### Scenario: Previously published fixed password

- **WHEN** a client attempts loopback TCP authentication with the old fixed image password
- **THEN** PostgreSQL rejects the connection

#### Scenario: External metadata database

- **WHEN** the operator explicitly supplies a connection string for an external metadata database
- **THEN** startup preserves that value rather than replacing it with the embedded database connection

#### Scenario: Explicit connection string points to the embedded database

- **WHEN** the operator explicitly supplies the previously published fixed connection string for the embedded database
- **THEN** the application cannot authenticate after startup rotates the internal role's password
- **AND** the operator must remove that explicit value so startup can supply the generated credential
