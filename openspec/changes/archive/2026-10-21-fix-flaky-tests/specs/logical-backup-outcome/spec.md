## ADDED Requirements

### Requirement: The fail message includes the dump tool's error output

When the dump tool of a logical PostgreSQL, MySQL, MariaDB or MongoDB backup exits with an error and the storage upload did not fail, the backup SHALL be recorded as failed, and its fail message SHALL contain the error output the tool wrote before it exited. This SHALL hold however quickly the tool exits after writing that output, including when it exits before it produced any backup data.

#### Scenario: Dump tool fails before producing data

- **WHEN** the dump tool writes an error such as `permission denied for sequence orders_id_seq` and exits with a non-zero status before producing any backup data
- **THEN** the backup is recorded as failed, and its fail message contains `permission denied for sequence orders_id_seq`

## MODIFIED Requirements

### Requirement: The fail message names the storage error

When a logical backup fails because its storage upload failed, the fail message SHALL contain the error the storage returned, so the user can see which storage operation failed and why. It SHALL NOT be reduced to a generic message such as "backup cancelled". This SHALL hold when the storage fails the upload at once, before it has read any backup data, and the fail message SHALL NOT then report that starting the dump tool was cancelled.

#### Scenario: Fail message after a storage timeout

- **WHEN** a logical backup fails because the upload to its S3 storage timed out while connecting to the endpoint
- **THEN** the fail message shows that saving to storage failed and includes the storage's own description of the timeout, including the endpoint it tried to reach

#### Scenario: Fail message is the same for every engine

- **WHEN** the same storage timeout interrupts a PostgreSQL, a MySQL, a MariaDB and a MongoDB backup
- **THEN** each backup is recorded as failed with a fail message that includes the storage error

#### Scenario: Storage fails before reading any data

- **WHEN** the storage fails the upload of a logical backup at once, before it has read any backup data
- **THEN** the backup is recorded as failed, its fail message shows that saving to storage failed and includes the storage error, and it does not say that starting the dump tool was cancelled
