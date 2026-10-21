## Purpose

Defines how the Databasus container stops its application and its embedded PostgreSQL server, so that a normal stop leaves the metadata database cleanly shut down and the next start needs no crash recovery.

## ADDED Requirements

### Requirement: A stop signal shuts the application down before the embedded database

When the container receives `SIGTERM` or `SIGINT` while the application runs, the container SHALL pass the signal to the application and wait for the application process to exit. The embedded PostgreSQL server SHALL receive its shutdown request only after that exit. The container SHALL then shut the server down cleanly, wait for that shutdown to finish and exit with the application's exit status.

#### Scenario: Operator stops the container

- **WHEN** an operator stops a healthy container with the default stop timeout
- **THEN** the container exits with status `0` before the timeout
- **AND** the stopped data directory reports the cluster state `shut down`, not `in production`

#### Scenario: Interrupt signal

- **WHEN** a healthy container receives `SIGINT`
- **THEN** the container exits before the stop timeout
- **AND** the stopped data directory reports the cluster state `shut down`

#### Scenario: Database shutdown follows the application's exit

- **WHEN** the container stops after a stop signal
- **THEN** the log line in which the container reports the application's exit comes before the line in which PostgreSQL reports its shutdown request

### Requirement: The container ends when either service ends

When the application exits without a stop signal, the container SHALL shut the embedded PostgreSQL server down cleanly and exit with the application's exit status. When the embedded PostgreSQL server exits while the application runs, the container SHALL stop the application and exit with a non-zero status.

#### Scenario: Application exits on its own

- **WHEN** the application process exits while the container has received no stop signal
- **THEN** the container exits with the application's exit status
- **AND** the stopped data directory reports the cluster state `shut down`

#### Scenario: Embedded database exits unexpectedly

- **WHEN** the embedded PostgreSQL server exits while the application runs
- **THEN** the application process stops
- **AND** the container exits with a non-zero status

### Requirement: A stop signal during startup ends startup

When the container receives `SIGTERM` or `SIGINT` before the application has started, it SHALL NOT start the application. It SHALL stop the PostgreSQL process that is running at that moment, wait for it to exit and exit with a non-zero status. The container SHALL act on the signal while a PostgreSQL startup step is still running, not only after that step has finished.

#### Scenario: Stop while the regular server starts

- **WHEN** the container receives a stop signal after the regular PostgreSQL server has started and before the application starts
- **THEN** the container exits with a non-zero status before the stop timeout
- **AND** the log has no line that the application starts
- **AND** the stopped data directory reports the cluster state `shut down`

#### Scenario: Stop during a long startup step

- **WHEN** the container receives a stop signal while a PostgreSQL startup administration step is still running
- **THEN** the container logs that it is stopping startup before that step has ended
- **AND** the container exits with a non-zero status, and the application never starts
