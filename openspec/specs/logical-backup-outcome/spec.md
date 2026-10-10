# Logical Backup Outcome Specification

## Purpose

Decides how a logical backup that did not complete is reported, so that a backup the user stopped on purpose stays quiet while every other unfinished backup is reported as a failure that reaches the user through notifications.

## Requirements

### Requirement: Only a cancellation request makes a backup cancelled

A logical backup of a PostgreSQL, MySQL, MariaDB or MongoDB database SHALL be recorded as cancelled only when a user requested cancellation of that backup before it ended. Every other backup that does not complete SHALL be recorded as failed. This SHALL hold whatever the underlying error says, including an error that describes a timeout or a cancellation inside a storage provider.

A cancelled backup SHALL NOT send a backup failure notification and SHALL NOT mark its database as failing. A failed backup SHALL get a fail message, SHALL mark its database as failing, and SHALL send the backup failure notification to the notifiers configured for it.

#### Scenario: Storage times out after the dump has finished

- **WHEN** the database dump finishes, and the upload of the backup to its storage then fails with a network timeout
- **THEN** the backup is recorded as failed, its database is marked as failing, and the backup failure notification is sent

#### Scenario: Storage times out while the dump is still running

- **WHEN** the upload of the backup fails with a network timeout while the database dump is still producing data
- **THEN** the backup is recorded as failed and the backup failure notification is sent

#### Scenario: Storage error that wraps a cancellation

- **WHEN** the storage provider fails the upload with an error that reports its own request as cancelled, and nobody requested cancellation of the backup
- **THEN** the backup is recorded as failed and the backup failure notification is sent

#### Scenario: User cancels a running backup

- **WHEN** a user cancels a logical backup while it is running
- **THEN** the backup is recorded as cancelled, no backup failure notification is sent, and the database is not marked as failing

#### Scenario: User cancels while the storage is failing

- **WHEN** a user requests cancellation of a running logical backup, and the backup then ends with a storage error instead of a cancellation error
- **THEN** the backup is recorded as cancelled, no backup failure notification is sent, and the database is not marked as failing

#### Scenario: Backup exceeds the time limit

- **WHEN** a logical backup runs longer than the backup time limit of 23 hours and is stopped for that reason
- **THEN** the backup is recorded as failed with the fail message `backup exceeded the time limit of 23 hours`, its database is marked as failing, and the backup failure notification is sent

#### Scenario: Application shuts down during a backup

- **WHEN** the application shuts down while a logical backup is running
- **THEN** the backup is recorded as failed with the fail message `backup interrupted by application shutdown`, not as cancelled

#### Scenario: Backup left in progress by a previous run

- **WHEN** the application starts and finds a logical backup still marked as in progress from its previous run
- **THEN** the backup is recorded as failed with the fail message `Backup failed due to application restart`, not as cancelled, even though cancellation is requested for it

### Requirement: The fail message names the storage error

When a logical backup fails because its storage upload failed, the fail message SHALL contain the error the storage returned, so the user can see which storage operation failed and why. It SHALL NOT be reduced to a generic message such as "backup cancelled".

#### Scenario: Fail message after a storage timeout

- **WHEN** a logical backup fails because the upload to its S3 storage timed out while connecting to the endpoint
- **THEN** the fail message shows that saving to storage failed and includes the storage's own description of the timeout, including the endpoint it tried to reach

#### Scenario: Fail message is the same for every engine

- **WHEN** the same storage timeout interrupts a PostgreSQL, a MySQL, a MariaDB and a MongoDB backup
- **THEN** each backup is recorded as failed with a fail message that includes the storage error
