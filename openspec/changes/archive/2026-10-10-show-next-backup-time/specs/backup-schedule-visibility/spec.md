## Purpose

Tells users when the next automatic backup of a database will run, so they can confirm a schedule is set up correctly and spot schedule or timezone mistakes without working out the next run themselves.

## ADDED Requirements

### Requirement: The reported next backup time matches when the scheduler will act

For a database with automatic backups enabled, the system SHALL report the earliest moment, at or after the current time, at which the scheduler treats a scheduled backup as due. The calculation SHALL use the same schedule rules and the same last backup the scheduler uses. That last backup is the newest one of any status, including failed and in-progress backups. When a backup is already due, the system SHALL report it as due now instead of naming a future slot.

#### Scenario: Daily schedule before today's slot

- **WHEN** a database has a daily schedule at 03:00 UTC, its last backup ran yesterday at 03:00 UTC, and the current time is 01:00 UTC
- **THEN** the system reports the next backup at 03:00 UTC today

#### Scenario: Daily schedule with a missed slot

- **WHEN** a database has a daily schedule at 03:00 UTC, its last backup ran two days ago, and the current time is 10:00 UTC
- **THEN** the system reports the backup as due now

#### Scenario: Hourly schedule counts a failed backup

- **WHEN** a database has an hourly schedule and its newest backup failed 20 minutes ago
- **THEN** the system reports the next backup one hour after that failed backup started

#### Scenario: Cron schedule

- **WHEN** a database has the cron schedule `0 */6 * * *` and its last backup ran at 06:00 UTC today, and the current time is 08:00 UTC
- **THEN** the system reports the next backup at 12:00 UTC today

#### Scenario: Weekly schedule does not catch up a previous week

- **WHEN** a database has a weekly schedule on Wednesday at 03:00 UTC, its last backup ran three weeks ago, and the current time is Tuesday
- **THEN** the system reports the next backup on Wednesday of the current week at 03:00 UTC

#### Scenario: Monthly schedule on a day the next month lacks

- **WHEN** a database has a monthly schedule on day 31 at 03:00 UTC, its last backup ran on January 31 at 03:00 UTC, and the current time is January 31 at 10:00 UTC
- **THEN** the system reports the next backup on March 31 at 03:00 UTC, because the scheduler skips months without day 31

#### Scenario: No backup exists yet

- **WHEN** a database has automatic backups enabled and no backup exists
- **THEN** the system reports the backup as due now

### Requirement: Physical databases report full and incremental times separately

For a physical database with automatic backups enabled, the system SHALL report the next full backup time from the full backup schedule. When incremental backups are enabled, the system SHALL also report the next incremental backup time from the incremental schedule. The incremental time SHALL be measured from the newest full or incremental backup. When incremental backups are disabled, the system SHALL report no incremental time. WAL streaming SHALL NOT affect either time.

#### Scenario: Full and incremental backups enabled

- **WHEN** a physical database has a weekly full schedule and an hourly incremental schedule
- **THEN** the system reports both the next full backup time and the next incremental backup time

#### Scenario: Full backups only

- **WHEN** a physical database takes full backups only
- **THEN** the system reports the next full backup time and no incremental time

#### Scenario: Incremental backups with WAL streaming

- **WHEN** a physical database takes full and incremental backups and streams WAL
- **THEN** the system reports the next full and the next incremental backup times, computed as if WAL streaming were off

### Requirement: No next backup is reported when automatic backups will not run

The system SHALL report no next backup time when automatic backups are disabled for the database, or when the database has no backup storage. For a physical database, the system SHALL also report no next backup time while automatic backups stay suspended because the database server's identity changed and the change is unresolved. The system SHALL report no next incremental backup time while incremental backups are held back. That happens when no backup chain can be extended and either the chain broke because WAL summarization was off on the server, or none of the newest full backup attempts completed. Reading the next backup time SHALL NOT create or change any backup settings.

#### Scenario: Automatic backups disabled

- **WHEN** automatic backups are disabled for a database
- **THEN** the system reports no next backup time and the interface shows no next-backup line

#### Scenario: No backup storage

- **WHEN** automatic backups are enabled for a database that has no backup storage
- **THEN** the system reports no next backup time

#### Scenario: Database without saved backup settings

- **WHEN** the next backup time is requested for a database whose backup settings were never saved
- **THEN** the system reports no next backup time and creates no backup settings

#### Scenario: Incremental backups held back after WAL summarization was off

- **WHEN** a physical database's incremental chain broke because WAL summarization was off and no chain can be extended
- **THEN** the system reports the next full backup time and no next incremental backup time

#### Scenario: Incremental backups held back after failed full backups

- **WHEN** a physical database has no extendable chain and none of its newest full backup attempts completed
- **THEN** the system reports the next full backup time and no next incremental backup time

#### Scenario: Physical backups suspended after a server identity change

- **WHEN** automatic physical backups are suspended because the database server's identity changed
- **THEN** the system reports no next full and no next incremental backup time

### Requirement: The interface shows the next backup time

The database card SHALL show the next backup time below the last backup time, as a relative time such as "in 3 hours". The read-only backup settings SHALL show the next backup time as a date and time with the relative time beside it, for every schedule type. For a physical database, they SHALL show the next full time under the full schedule and the next incremental time under the incremental schedule. A due backup SHALL read as due now. A reported time that has passed SHALL also read as due now, at the latest when the view next refreshes. The database card SHALL refresh the next backup time whenever it refreshes the last backup time. All times SHALL be displayed in the browser's local timezone, and all new text SHALL be available in every interface language.

#### Scenario: Card for a logical database

- **WHEN** a user views a logical database card whose next backup is in 3 hours
- **THEN** the card shows the last backup line and, under it, a next backup line reading "in 3 hours" in the selected language

#### Scenario: Card for a physical database

- **WHEN** a user views the card of a physical database that takes full and incremental backups
- **THEN** the card shows the next full backup time and the next incremental backup time

#### Scenario: Settings for a daily schedule

- **WHEN** a user opens the read-only backup settings of a database with a daily schedule
- **THEN** the settings show the next backup date and time in the browser's timezone, with the relative time beside it

#### Scenario: Due backup

- **WHEN** the reported next backup is due now or its time has passed
- **THEN** the interface reads it as due now instead of showing a time in the past

### Requirement: A request for the wrong kind of database is rejected

The system SHALL reject a request for a logical database's next backup time when the database is physical, and a request for a physical database's next backup times when the database is logical. The request SHALL fail as invalid instead of reporting that no backup is scheduled.

#### Scenario: Logical request for a physical database

- **WHEN** a user with access asks for the logical next backup time of a physical database
- **THEN** the system rejects the request as invalid

#### Scenario: Physical request for a logical database

- **WHEN** a user with access asks for the physical next backup times of a logical database
- **THEN** the system rejects the request as invalid

### Requirement: Only users who can view a database's backups can read its next backup time

The system SHALL return the next backup time only to a user who is allowed to view that database's backups. Any other request SHALL be rejected and SHALL NOT reveal the schedule, the next backup time, or whether automatic backups are enabled.

#### Scenario: Member of the database's workspace

- **WHEN** a user who can view the database's backups requests its next backup time
- **THEN** the system returns the next backup time

#### Scenario: User outside the database's workspace

- **WHEN** a user without access to the database's workspace requests its next backup time
- **THEN** the system rejects the request and returns no schedule information

#### Scenario: Unauthenticated request

- **WHEN** a request without a valid session asks for a database's next backup time
- **THEN** the system rejects the request as unauthenticated
