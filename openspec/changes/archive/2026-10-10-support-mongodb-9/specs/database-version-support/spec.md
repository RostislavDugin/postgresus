## ADDED Requirements

### Requirement: Supported MongoDB server versions are accepted

The system SHALL accept a MongoDB server reporting version 4.2 or later up to and including the 9 line, and SHALL record a version identity that is the server's major version alone, so that every release inside one major line shares one identity.

#### Scenario: Add a MongoDB 9 server

- **WHEN** an authorized user tests the connection to a reachable MongoDB server reporting version 9.0 with sufficient backup privileges
- **THEN** the connection test succeeds and the database records a version identity of 9

#### Scenario: A later minor inside the MongoDB 9 line

- **WHEN** an authorized user tests the connection to a MongoDB server reporting a 9.x version later than 9.0, such as 9.2
- **THEN** the connection test succeeds and the database records a version identity of 9

#### Scenario: Add a MongoDB server on a previously supported line

- **WHEN** an authorized user tests the connection to a reachable MongoDB server reporting version 4.2, 4.4, 5.x, 6.x, 7.x or 8.x with sufficient backup privileges
- **THEN** the connection test succeeds and the database records the server's major version as its identity

#### Scenario: A database recorded before this change

- **WHEN** the product starts with MongoDB databases recorded under identities 4 to 8, and a backup and a restore run for one of them
- **THEN** the database keeps its recorded identity, and the backup and the restore succeed with the shipped MongoDB client

### Requirement: A MongoDB server outside the supported range is refused with a reason

The system SHALL refuse a MongoDB server older than 4.2 or newer than the 9 line, and SHALL name the version it found and the range it supports, so that no backup is attempted against a server whose major line the shipped client's vendor compatibility statement does not list. Inside an accepted major line the server may be newer than the releases that statement names, such as 9.2 against a client listed for 9.0: that gap is closed by refreshing the client, not by refusing the database.

#### Scenario: MongoDB server below 4.2

- **WHEN** an authorized user tests the connection to a MongoDB server reporting version 4.0 or 3.6
- **THEN** the connection test fails, reports the version it found, names 4.2 as the minimum supported version, and no database is added

#### Scenario: MongoDB server above the newest supported line

- **WHEN** an authorized user tests the connection to a MongoDB server reporting version 10.0
- **THEN** the connection test fails, reports the version it found and the supported range, and no database is added

#### Scenario: MongoDB version string cannot be read

- **WHEN** the MongoDB server answers the version query with a value the system cannot parse
- **THEN** the connection test fails and reports the value it received

## MODIFIED Requirements

### Requirement: The downgrade guard covers every accepted version

The system SHALL refuse to restore a backup taken from a newer server onto an older server for every MySQL, MariaDB and MongoDB version it accepts, including versions added after the guard was written.

#### Scenario: Restore a calendar-scheme MySQL backup onto an earlier line

- **WHEN** an authorized user requests a restore of a backup taken from a MySQL 26.x server onto a MySQL 9.x or 8.x server
- **THEN** the system refuses the restore and explains that the backup comes from a newer server

#### Scenario: Restore a MariaDB 13 backup onto MariaDB 12

- **WHEN** an authorized user requests a restore of a backup taken from a MariaDB 13.x server onto a MariaDB 12.x server
- **THEN** the system refuses the restore and explains that the backup comes from a newer server

#### Scenario: Restore a MongoDB 9 backup onto MongoDB 8

- **WHEN** an authorized user requests a restore of a backup taken from a MongoDB 9.x server onto a MongoDB 8.x server
- **THEN** the system refuses the restore and explains that the backup comes from a newer server

#### Scenario: Restore onto an equal or newer server

- **WHEN** an authorized user requests a restore of a backup onto a server of the same or a newer version identity
- **THEN** the system allows the restore
