## REMOVED Requirements

### Requirement: The reported next backup time matches when the scheduler will act
**Reason**: The interface no longer reports a next backup time, so nothing computes one.
**Migration**: None. The feature has not shipped in a release, and only the Databasus frontend called its endpoints. Users read the schedule from the interval shown in the backup settings.

### Requirement: Physical databases report full and incremental times separately
**Reason**: Physical databases no longer report next full or incremental backup times.
**Migration**: None. The feature has not shipped in a release, and only the Databasus frontend called its endpoints. Users read the schedule from the interval shown in the backup settings.

### Requirement: No next backup is reported when automatic backups will not run
**Reason**: With no next backup time reported, there is no state in which it is withheld.
**Migration**: None. The feature has not shipped in a release, and only the Databasus frontend called its endpoints. Users read the schedule from the interval shown in the backup settings.

### Requirement: The interface shows the next backup time
**Reason**: The database card and the read-only backup settings no longer show a next backup time. The browser-computed "Next run" line under a cron interval in the read-only settings is removed with it.
**Migration**: None. The feature has not shipped in a release, and only the Databasus frontend called its endpoints. Users read the schedule from the interval shown in the backup settings.

### Requirement: A request for the wrong kind of database is rejected
**Reason**: The endpoints that reported next backup times are deleted, so there is no request to reject.
**Migration**: None. The feature has not shipped in a release, and only the Databasus frontend called its endpoints. Users read the schedule from the interval shown in the backup settings.

### Requirement: Only users who can view a database's backups can read its next backup time
**Reason**: The endpoints that reported next backup times are deleted, so there is nothing left to authorize.
**Migration**: None. The feature has not shipped in a release, and only the Databasus frontend called its endpoints. Users read the schedule from the interval shown in the backup settings.
