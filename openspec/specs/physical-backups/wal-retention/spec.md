# WAL Retention Specification

## Purpose

Defines which archived WAL segments a database keeps and which the retention sweep may delete, stated in terms of the recovery chains a user can still restore from. A segment is expendable only when no retained backup needs it.

## Requirements

### Requirement: A WAL segment overlapping a retained backup's recovery window is never deleted as unreferenced

The retention sweep SHALL treat a WAL segment as unreferenced only when no retained, successfully completed physical full backup on the same timeline begins before the end of that segment.

Recovery positions inside a full backup do not fall on segment boundaries: a backup begins a short distance into the segment that holds it. The segment holding that position therefore begins earlier than the backup does, and it still carries every write recorded after the backup finished. That segment SHALL be retained.

#### Scenario: The full backup starts and ends inside one segment

- **WHEN** a full backup's start and end positions both fall inside a single WAL segment, and that segment has been archived
- **THEN** the retention sweep leaves the segment in place
- **AND** a restore to any point in time after the backup's end position and covered by the archived WAL succeeds

#### Scenario: A segment entirely older than every retained backup

- **WHEN** an archived WAL segment ends at or before the start of the earliest retained full backup on its timeline
- **THEN** the retention sweep deletes it as unreferenced

#### Scenario: The anchoring backup is pruned

- **WHEN** retention removes the oldest full backup, and a later full backup remains whose start position falls inside a WAL segment that the removed backup used to anchor
- **THEN** that segment is still retained, because the remaining backup starts before the segment ends
- **AND** point-in-time restore from the remaining backup stays available

### Requirement: A completed full backup leaves an unbroken recovery chain

Once a physical full backup reports success and WAL streaming stays healthy, the system SHALL keep the archived WAL contiguous from that backup's end position onward, for as long as the backup is retained. No background maintenance SHALL introduce a gap immediately after a successful backup.

#### Scenario: Retention runs seconds after a full backup completes

- **WHEN** a full backup completes, its boundary WAL segment is archived, and the retention sweep runs before any further backup exists
- **THEN** the archived WAL is unbroken from the backup's end position to the newest archived segment
- **AND** the reported latest restorable point is later than the backup's end position rather than equal to it

#### Scenario: Restoring to a point shortly after the backup

- **WHEN** a user requests a restore to a point in time a few seconds after a successful full backup, and WAL covering that point has been archived
- **THEN** the restore proceeds instead of being refused for a WAL gap

#### Scenario: Restoring while a newer full backup is running

- **WHEN** a newer full backup has started but has not completed, and the archived WAL remains contiguous from the preceding completed full backup
- **THEN** a point-in-time restore during that interval uses the preceding completed full backup
- **AND** the restore set contains the full available contiguous WAL run, including WAL beyond the newer backup's start and stop positions

### Requirement: WAL archived during an active full backup is protected

No orphan or cascade deletion SHALL remove WAL for a database while that database has an active FULL backup claim. Scheduling the claim and every WAL-deleting maintenance path SHALL be serialized for each database, and each deletion transaction SHALL recheck the claim before removing a segment.

#### Scenario: All backups were deleted and a replacement full is running

- **WHEN** no completed full backup remains, a replacement FULL has an active claim, and WAL is archived during that backup
- **THEN** the orphan sweep leaves the WAL in place
- **AND** the WAL becomes anchored when the FULL completes

#### Scenario: An in-progress row has no active claim

- **WHEN** an `IN_PROGRESS` full backup row remains without an active FULL claim and no completed full backup anchors the WAL
- **THEN** the orphan sweep may reclaim the WAL

#### Scenario: Cascade deletion overlaps replacement full claim creation

- **WHEN** full removal or `FULL_BACKUPS` dependent cleanup starts while a replacement FULL claim is being committed
- **THEN** the cascade waits for claim creation to finish
- **AND** it leaves WAL in place after observing the active claim
- **AND** full removal leaves timeline history in place until the active FULL publishes a completed anchor
- **AND** the deletion preview reports zero deletable WAL and timeline history while that claim remains active

### Requirement: Deleting a full backup preserves WAL required by a surviving predecessor

Deleting a completed full backup SHALL retain its WAL when an older completed full backup remains on the same database and timeline. The older backup becomes the restore base after deletion and SHALL remain capable of replaying the shared WAL. The deletion preview SHALL report the same WAL count and size that the destructive operation can remove.

This rule applies when the FULL itself is removed. A `FULL_BACKUPS` retention policy may deliberately keep a FULL as a standalone restore point while removing its incrementals and owned WAL.

#### Scenario: A newer full backup is deleted

- **WHEN** a completed full backup is deleted while an older completed full backup remains on the same timeline
- **THEN** no WAL is deleted with the newer full backup
- **AND** point-in-time restore during the deleted backup's former interval succeeds from the older backup

#### Scenario: The earliest surviving full backup is deleted

- **WHEN** a completed full backup has no older completed predecessor on its timeline
- **THEN** deletion removes only WAL segments fully contained in its ownership span
- **AND** a segment crossing the next full backup's start position remains available
