## ADDED Requirements

### Requirement: The MongoDB client tools are the vendor's unmodified build from a recorded source

The system SHALL ship MongoDB client tools that are byte-identical to the programs inside an archive the vendor publishes, and the client documentation SHALL record, for each architecture, the tools release, the archive's download address and the archive's published checksum. Both architectures SHALL come from the same tools release and the same vendor build family, so that one documented release means one build on every architecture.

#### Scenario: An operator verifies the shipped MongoDB tools

- **WHEN** an operator downloads the archive the client documentation names for an architecture and checks it against the recorded checksum
- **THEN** the checksum matches, and the shipped dump and restore programs for that architecture are byte-identical to the ones in the archive

#### Scenario: Both architectures carry the same tools release

- **WHEN** the version embedded in the shipped MongoDB dump and restore programs is read for each supported architecture
- **THEN** every program carries the tools release the client documentation records

## MODIFIED Requirements

### Requirement: Every accepted version identity has a designated client on every supported architecture

The system SHALL ship, for each processor architecture it runs on, a client designated to dump and restore every MySQL, MariaDB and MongoDB version identity it accepts.

For MySQL and MariaDB, the designated client is either the one built for that identity's own release line, or a client from another line documented as serving it — a newer one, as the modern MariaDB client serves every line from 10.2 up, or an older one, as the legacy MariaDB client serves the lines the modern client cannot dump. A MySQL or MariaDB client MAY trail the server inside a single release line, and SHALL never come from a line older than the server's own.

MongoDB clients are numbered independently of server release lines, so the release-line clauses above do not apply to them. MongoDB is served by a single client for every accepted identity, and that client SHALL be a release whose vendor compatibility statement lists every MongoDB major line the system accepts.

#### Scenario: Start on a supported architecture

- **WHEN** the product starts on either supported architecture
- **THEN** it reports each expected client set as verified, except for a client set documented as unavailable on that architecture

#### Scenario: A version identity served by a newer client

- **WHEN** a version identity has no client built specifically for it and a newer shipped client is designated to serve it
- **THEN** the system serves it with that client, and the designation is recorded in the client documentation

#### Scenario: A version identity served by an older client on purpose

- **WHEN** a version identity exists for servers a newer client cannot dump, as MariaDB 5.5 and 10.1 need a client older than 10.2
- **THEN** the product ships a client from a line that can dump them on every architecture, and that client is used for those identities

#### Scenario: An architecture with no client for a version identity

- **WHEN** an architecture ships no client able to serve a version identity, as arm64 ships none for MySQL 5.7
- **THEN** that identity is unsupported on that architecture and a server reporting it is refused rather than attempted

#### Scenario: One MongoDB client serves every accepted MongoDB identity

- **WHEN** a backup or restore runs for a MongoDB database of any accepted identity, from 4 to 9, on either supported architecture
- **THEN** the shipped MongoDB tools perform it, and the vendor's compatibility statement for that tools release lists the server's major line
