## Purpose

Defines what the scheduled healthcheck of a PostgreSQL physical source verifies, and guarantees that the healthcheck never competes with backups for replication capacity on that source.

## ADDED Requirements

### Requirement: The physical source healthcheck reports whether the source accepts the stored credentials

The scheduled healthcheck of a PostgreSQL physical source SHALL record the source as available when the source accepts an ordinary connection with the stored credentials and answers a trivial query within the probe's time limit. It SHALL record the source as unavailable when the connection or the query fails or does not finish in time.

The healthcheck SHALL NOT re-validate the source's replication configuration. WAL level, WAL sender and replication slot limits, WAL summarization, tablespaces, the WAL switch privilege and the cluster identity remain checked by the user-facing connection test and when a database is created or changed.

#### Scenario: Reachable source

- **WHEN** the healthcheck probes a physical source that accepts the stored credentials
- **THEN** the attempt is recorded as available

#### Scenario: Unreachable source

- **WHEN** the healthcheck probes a physical source that is stopped or cannot be reached
- **THEN** the attempt is recorded as unavailable

#### Scenario: Rejected credentials

- **WHEN** the healthcheck probes a physical source that rejects the stored password
- **THEN** the attempt is recorded as unavailable

#### Scenario: Source behind an SSH bastion

- **WHEN** the physical source is configured to be reached through an SSH bastion
- **THEN** the healthcheck reaches it through that bastion, as backups do

### Requirement: The physical source healthcheck never uses replication capacity

The scheduled healthcheck of a PostgreSQL physical source SHALL NOT open a replication connection to the source. A backup or a WAL stream running on the source SHALL NOT turn the healthcheck unavailable when the source has no WAL sender slot to spare, and the healthcheck SHALL NOT take a WAL sender slot that a backup or a WAL stream needs.

#### Scenario: Backup holds the last WAL sender slot

- **WHEN** a backup and a WAL stream occupy every WAL sender slot the source allows
- **AND** the healthcheck probes that source
- **THEN** the attempt is recorded as available

#### Scenario: Backup starts while the healthcheck runs

- **WHEN** the healthcheck is probing a source whose WAL sender slots are all taken except one
- **AND** a backup starts at the same moment
- **THEN** the backup obtains the remaining WAL sender slot

### Requirement: A failed physical source healthcheck leaves its cause in the log

When the healthcheck of a PostgreSQL physical source records an unavailable attempt, the reason the source gave SHALL appear in the application log entry for that attempt, so an operator can tell a refused password from a timeout or an unreachable host.

#### Scenario: Refused connection

- **WHEN** the source refuses the healthcheck connection
- **THEN** the log entry for the failed attempt contains the source's refusal message

### Requirement: The physical source healthcheck never writes the source password to the log

No log entry produced by the healthcheck of a PostgreSQL physical source SHALL contain the stored password of that source, whether the probe succeeds or fails.

#### Scenario: Wrong password rejected by the source

- **WHEN** the stored password is wrong and the source rejects the healthcheck connection
- **THEN** the log entry for the failed attempt does not contain that password
