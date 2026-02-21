# Design: Multiple Backup Schedules Per Database

**Date:** 2026-02-21
**Status:** Approved
**Author:** User + AI Assistant

## Problem Statement

Currently, each database can only have one backup configuration. Users cannot create multiple backup schedules with different intervals and retention policies for the same database.

**Example Use Case:**
- Hourly backups kept for 1 day
- Weekly backups kept for 1 month
- Both for the same database

## Current Architecture

```
BackupConfig
├── DatabaseID (PK) ← Forces 1:1 relationship
├── BackupInterval
├── RetentionPolicy
├── StorageID
└── ... other fields
```

## Proposed Solution

Change `DatabaseID` from primary key to a foreign key, introducing a new `ID` UUID primary key. This allows multiple backup configurations per database.

## Database Schema Changes

### Migration: `alter_backup_configs_pk.sql`

```sql
-- Add new ID column as primary key
ALTER TABLE backup_configs ADD COLUMN id UUID DEFAULT gen_random_uuid();

-- Populate ID for existing rows
UPDATE backup_configs SET id = gen_random_uuid() WHERE id IS NULL;

-- Drop existing primary key constraint
ALTER TABLE backup_configs DROP CONSTRAINT backup_configs_pkey;

-- Set new primary key
ALTER TABLE backup_configs ADD PRIMARY KEY (id);

-- Create index on database_id for faster lookups
CREATE INDEX idx_backup_configs_database_id ON backup_configs(database_id);

-- Add name column for human-readable schedule identification
ALTER TABLE backup_configs ADD COLUMN name TEXT NOT NULL DEFAULT 'Default';

-- Add unique constraint for backup config naming within database
ALTER TABLE backup_configs ADD CONSTRAINT backup_configs_name_unique_per_database 
    UNIQUE (database_id, name);

*(Note: The API handles this constraint gracefully by checking for name uniqueness before insertion and returning a 400 Bad Request if a duplicate name is provided for the same database).*
```

## Model Changes

### File: `backend/internal/features/backups/config/model.go`

```go
type BackupConfig struct {
    ID         uuid.UUID `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
    DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;not null"`
    
    // NEW: Human-readable name for the schedule
    Name string `json:"name" gorm:"column:name;type:text;not null"`
    
    IsBackupsEnabled bool `json:"isBackupsEnabled" gorm:"column:is_backups_enabled;type:boolean;not null"`
    
    // ... existing fields remain unchanged
}
```

## Repository Changes

### File: `backend/internal/features/backups/config/repository.go`

**Current:**
- `FindByDatabaseID(databaseID uuid.UUID) (*BackupConfig, error)` - Returns single config

**New:**
- `FindByID(id uuid.UUID) (*BackupConfig, error)` - Find by config ID
- `FindByDatabaseID(databaseID uuid.UUID) ([]*BackupConfig, error)` - Returns list of configs
- `GetWithEnabledBackups() ([]*BackupConfig, error)` - No changes needed

## Service Changes

### File: `backend/internal/features/backups/config/service.go`

**New Methods:**
- `CreateBackupConfig(config *BackupConfig) (*BackupConfig, error)` - Create new config
- `UpdateBackupConfig(id uuid.UUID, config *BackupConfig) (*BackupConfig, error)` - Update by ID
- `DeleteBackupConfig(id uuid.UUID) error` - Delete by ID
- `GetBackupConfigsByDatabaseID(databaseID uuid.UUID) ([]*BackupConfig, error)` - List all configs

**Modified Methods:**
- `GetBackupConfigByDbId()` → `GetBackupConfigsByDatabaseID()` (returns list)
- `SaveBackupConfig()` → Split into Create/Update

## API Changes

### File: `backend/internal/features/backups/config/controller.go`

**Current Endpoints:**
```
GET  /api/v1/backup-configs/:databaseId      → Single BackupConfig
POST /api/v1/backup-configs                  → Create/Update (upsert)
```

**New Endpoints:**
```
GET    /api/v1/backup-configs?databaseId=:id → List[BackupConfig]
POST   /api/v1/backup-configs                → Create BackupConfig
PUT    /api/v1/backup-configs/:id            → Update BackupConfig by ID
DELETE /api/v1/backup-configs/:id            → Delete BackupConfig by ID
```

## Scheduler Changes

### File: `backend/internal/features/backups/backups/backuping/scheduler.go`

The scheduler already calls `GetBackupConfigsWithEnabledBackups()` which returns all enabled configs. No changes needed to the scheduler logic - it will naturally pick up all configs and schedule backups accordingly.

## Backup Model Changes

### File: `backend/internal/features/backups/backups/core/model.go`

Add reference to backup config:

```go
type Backup struct {
    ID             uuid.UUID  `json:"id" gorm:"primaryKey"`
    DatabaseID     uuid.UUID  `json:"databaseId" gorm:"column:database_id"`
    BackupConfigID *uuid.UUID `json:"backupConfigId" gorm:"column:backup_config_id"` // NEW
    // ... existing fields
}
```

This allows tracking which schedule created each backup.

## Backup File Naming

Backups will be stored with reference to the config:

```
backup_<timestamp>_<database-name>_<config-name-slug>
```

Example:
```
backup_20260221_143000_mydb_hourly.sql.gz
backup_20260221_040000_mydb_weekly.sql.gz
```

## Frontend Changes (High-Level)

1. **Database Detail Page:**
   - Show list of backup schedules instead of single config
   - Add "Create Schedule" button
   - Each schedule card shows: name, interval, retention, enabled status

2. **Schedule Editor:**
   - New/Edit modal for individual schedules
   - Name field (required, unique per database)
   - Interval, retention, storage settings

3. **Migration UX:**
   - Existing configs will be migrated with name "Default"
   - Users can rename after migration

## Migration Strategy

1. **Pre-deployment:**
   - Create migration SQL file

2. **During deployment:**
   - Run migration to add `id` column and populate
   - Run migration to change primary key
   - Run migration to add `name` column with default "Default"
   - Add unique constraint

3. **Post-deployment:**
   - Deploy new backend code
   - Deploy new frontend code

## Backward Compatibility

- All existing backup configs will be preserved
- Each existing config gets a generated UUID and name "Default"
- Existing backups will be linked to the migrated config
- API clients using old endpoints will need to update to new endpoints

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Data loss during migration | Test migration on staging; backup database before migration |
| Breaking existing API clients | Version API endpoints; provide deprecation notice |
| Frontend complexity | Incremental UI updates; keep simple use case simple |
| Performance with many configs | Add pagination; index on database_id |

## Success Criteria

1. User can create multiple backup schedules for one database
2. Each schedule has independent interval and retention settings
3. Backups are correctly attributed to their schedule
4. Existing databases continue to work after migration
5. UI clearly shows all schedules for a database
