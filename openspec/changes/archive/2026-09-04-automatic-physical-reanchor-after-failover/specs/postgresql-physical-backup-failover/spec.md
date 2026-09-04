## Purpose

Keep PostgreSQL physical backups recoverable after promotion by rejecting stale incremental chains, creating a replacement FULL, and retaining the WAL needed during that transition.

## ADDED Requirements

### Requirement: Incremental backups use their root FULL timeline
The system SHALL run an incremental backup only when the live PostgreSQL timeline matches the timeline of that incremental chain's root FULL.

#### Scenario: Timeline matches the root FULL
- **WHEN** an incremental backup starts and the live timeline equals its root FULL timeline
- **THEN** the system proceeds with the incremental backup

#### Scenario: Promotion happened before the incremental started
- **WHEN** the live timeline is newer than the root FULL timeline
- **THEN** the system refuses the incremental backup as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`

#### Scenario: Source timeline regressed
- **WHEN** the live timeline is older than the root FULL timeline
- **THEN** the system refuses the incremental backup with reason `TIMELINE_REGRESSION`

#### Scenario: Cataloged history is newer than the root FULL
- **WHEN** a history file for the live timeline is already cataloged but the root FULL belongs to an older timeline
- **THEN** the system still refuses the incremental backup as `TIMELINE_SWITCH_DETECTED`

### Requirement: Backup failures are rechecked for promotion
The system SHALL recheck PostgreSQL cluster identity after an unexpected physical backup stream failure while the backup execution remains active.

#### Scenario: Promotion interrupts an incremental stream
- **WHEN** the timeline advances after incremental preflight and the stream fails
- **THEN** the system records the incremental as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`

#### Scenario: Promotion interrupts a FULL stream
- **WHEN** the timeline advances after FULL preflight and the stream fails
- **THEN** the system records the FULL as `ERROR` with reason `FAILOVER_DURING_BACKUP`

#### Scenario: Backup was canceled
- **WHEN** the backup execution context was canceled by an operator or configuration change
- **THEN** the system preserves the cancellation result and does not classify it as a promotion

#### Scenario: Recheck cannot connect
- **WHEN** the identity recheck fails
- **THEN** the system preserves the original backup result

#### Scenario: Cluster identity changed
- **WHEN** the identity recheck reaches a cluster with a different system identifier
- **THEN** the system rejects that cluster and suppresses automatic backup scheduling until an operator explicitly requests a FULL that completes against the expected cluster

#### Scenario: Operator requests a FULL after cluster identity changed
- **WHEN** automatic scheduling is suppressed by a system identifier mismatch and an operator explicitly requests a FULL
- **THEN** the system attempts preflight again and still refuses to write a backup unless the live system identifier matches the configured database

#### Scenario: Forced FULL still reaches a different cluster
- **WHEN** the operator's forced FULL is refused with `SYSTEM_IDENTIFIER_MISMATCH`
- **THEN** automatic backup scheduling remains suppressed

#### Scenario: Forced FULL fails without proving cluster identity
- **WHEN** a forced FULL after a system identifier mismatch fails for another reason or is canceled
- **THEN** automatic backup scheduling remains suppressed because no later FULL has completed against the expected cluster

#### Scenario: Forced FULL reaches the expected cluster
- **WHEN** the operator's forced FULL completes against the configured system identifier
- **THEN** that newer completed FULL supersedes earlier mismatch results and normal automatic scheduling resumes

### Requirement: Timeline failures trigger one immediate replacement FULL
The system SHALL create one immediate replacement FULL after a timeline-related terminal backup result.

#### Scenario: Incremental chain broke after promotion
- **WHEN** an incremental ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED` and no later FULL attempt exists
- **THEN** the system schedules a FULL without waiting for the configured incremental or FULL interval

#### Scenario: FULL was interrupted by promotion
- **WHEN** the latest FULL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`
- **THEN** the system schedules one replacement FULL without waiting for the configured interval

#### Scenario: Replacement FULL fails normally
- **WHEN** the immediate replacement FULL ends with a non-timeline error or is canceled
- **THEN** the system resumes its existing cadence and failed-FULL retry limits instead of scheduling a FULL on every scheduler tick

#### Scenario: Another promotion interrupts the replacement
- **WHEN** the timeline advances again during the replacement FULL
- **THEN** the system schedules one new immediate replacement FULL for the later promotion

#### Scenario: Two schedulers hold the same stale replacement decision
- **WHEN** two scheduler instances decide that the same terminal backup needs an immediate replacement
- **THEN** only the instance that revalidates the still-current trigger creates a FULL row

### Requirement: Active FULL backups retain archived WAL
The system SHALL prevent orphan cleanup from deleting WAL for a database while a FULL backup for that database is active.

#### Scenario: Cleaner runs during the first FULL
- **WHEN** WAL is archived while the database's first FULL is still in progress and orphan cleanup runs
- **THEN** the WAL remains cataloged and stored for use after the FULL completes

#### Scenario: Cleaner finishes before a FULL starts
- **WHEN** orphan cleanup has begun before the scheduler starts a FULL for the same database
- **THEN** the FULL starts only after that cleanup operation has finished

### Requirement: Same-timeline PITR remains available during a newer FULL
The system SHALL allow a point-in-time restore from an older FULL when the target precedes completion of a newer FULL on the same timeline.

#### Scenario: Target falls during the newer FULL
- **WHEN** the requested target is later than the older FULL completion and earlier than the newer FULL completion
- **THEN** the restore set uses the older FULL and contiguous WAL through the target even when that WAL crosses the newer FULL start position

#### Scenario: Target crosses a promotion
- **WHEN** reaching the target would require WAL from more than one timeline
- **THEN** the system reports that the target is not restorable until a FULL on the promoted timeline is available
