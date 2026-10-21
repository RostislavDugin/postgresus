## ADDED Requirements

### Requirement: The embedded PostgreSQL instance accepts only authenticated loopback TCP

The embedded PostgreSQL server SHALL run under the same non-root operating-system account as the Databasus application. The server SHALL create no Unix socket, so no process in the container can open a socket connection to it. Loopback TCP connections SHALL require SCRAM authentication with the generated credential from the first connection the server accepts, and replication connections to the embedded cluster SHALL be rejected.

The embedded PostgreSQL port SHALL remain unpublished by the image.

#### Scenario: A process looks for a local socket

- **WHEN** any process in a running container attempts a Unix-socket connection to the embedded PostgreSQL server
- **THEN** the connection fails because the server exposes no socket

#### Scenario: Loopback TCP connection without the generated password

- **WHEN** any process connects to the embedded PostgreSQL server over loopback TCP without the current generated password
- **THEN** SCRAM authentication rejects the connection

#### Scenario: Application uses the generated password

- **WHEN** Databasus connects to the embedded PostgreSQL server over loopback with the generated password
- **THEN** PostgreSQL accepts the connection

#### Scenario: Replication connection to the embedded cluster

- **WHEN** a client requests a replication connection to the embedded PostgreSQL server
- **THEN** PostgreSQL rejects it

### Requirement: Startup administration of the embedded cluster opens no connection path

The container SHALL set the generated password on the internal role before the embedded PostgreSQL server accepts any connection. That step SHALL run while the cluster has no network listener and no Unix socket. No startup step SHALL grant a password-less or identity-mapped login to the embedded cluster.

#### Scenario: First connection after startup administration

- **WHEN** the embedded PostgreSQL server accepts its first connection after a container start
- **THEN** the internal role already carries the newly generated password
- **AND** the connection must authenticate with that password

#### Scenario: Password from the previous run after a restart

- **WHEN** the container restarts and a process logs in over loopback with the password published by the previous run
- **THEN** SCRAM authentication rejects the connection
- **AND** a loopback login with the newly published password succeeds

### Requirement: The generated password stays out of process arguments and server logs

Startup SHALL NOT pass the generated password to any process through its command-line arguments. When the statement that sets the password fails, the container SHALL exit with an error before the Databasus application starts, and the container log SHALL NOT contain that statement or the password.

#### Scenario: Process list during startup

- **WHEN** a process in the container reads the command-line arguments of the startup processes
- **THEN** none of them contains the generated password

#### Scenario: Password statement fails

- **WHEN** the statement that sets the generated password fails
- **THEN** the container exits before starting the Databasus application
- **AND** the container log contains neither the statement nor the generated password

## REMOVED Requirements

### Requirement: The embedded PostgreSQL instance stays private under the shared runtime account

**Reason**: The embedded server no longer creates a Unix socket, so the temporary peer mapping and the post-bootstrap socket rejection this requirement described no longer exist. "The embedded PostgreSQL instance accepts only authenticated loopback TCP" and "Startup administration of the embedded cluster opens no connection path" replace it.

**Migration**: No operator action is needed. Manual administration uses the connection string published inside a running container, or PostgreSQL single-user mode inside a stopped one.
