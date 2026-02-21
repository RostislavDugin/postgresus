# Multiple Backup Schedules Per Database - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow users to create multiple backup schedules with different intervals and retention policies for the same database.

**Architecture:** Change `backup_configs` table primary key from `database_id` to a new `id` UUID column, enabling a 1:N relationship between databases and backup configs.

**Tech Stack:** Go, Gin, GORM, PostgreSQL, Goose migrations

---

## Task 1: Create Database Migration

**Files:**
- Create: `backend/migrations/20260221000000_multiple_backup_schedules.sql`

**Step 1: Write the migration file**

```sql
-- +goose Up
-- +goose StatementBegin

-- Add new ID column
ALTER TABLE backup_configs ADD COLUMN id UUID DEFAULT gen_random_uuid();

-- Populate ID for existing rows
UPDATE backup_configs SET id = gen_random_uuid() WHERE id IS NULL;

-- Set NOT NULL on id column
ALTER TABLE backup_configs ALTER COLUMN id SET NOT NULL;

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

-- Add backup_config_id to backups table
ALTER TABLE backups ADD COLUMN backup_config_id UUID;

-- Populate backup_config_id for existing backups
UPDATE backups b
SET backup_config_id = bc.id
FROM backup_configs bc
WHERE b.database_id = bc.database_id;

-- Add foreign key constraint for backup_config_id
ALTER TABLE backups
    ADD CONSTRAINT fk_backups_backup_config_id
    FOREIGN KEY (backup_config_id)
    REFERENCES backup_configs (id)
    ON DELETE SET NULL;

-- Create index for backups lookup by config
CREATE INDEX idx_backups_backup_config_id ON backups(backup_config_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove index from backups
DROP INDEX IF EXISTS idx_backups_backup_config_id;

-- Remove foreign key from backups
ALTER TABLE backups DROP CONSTRAINT IF EXISTS fk_backups_backup_config_id;

-- Remove backup_config_id from backups
ALTER TABLE backups DROP COLUMN IF EXISTS backup_config_id;

-- Remove unique constraint from backup_configs
ALTER TABLE backup_configs DROP CONSTRAINT IF EXISTS backup_configs_name_unique_per_database;

-- Remove name column
ALTER TABLE backup_configs DROP COLUMN IF EXISTS name;

-- Remove index
DROP INDEX IF EXISTS idx_backup_configs_database_id;

-- Drop new primary key
ALTER TABLE backup_configs DROP CONSTRAINT backup_configs_pkey;

-- Restore old primary key
ALTER TABLE backup_configs ADD PRIMARY KEY (database_id);

-- Remove id column
ALTER TABLE backup_configs DROP COLUMN id;

-- +goose StatementEnd
```

**Step 2: Verify migration file is created**

Run: `ls backend/migrations/20260221000000_multiple_backup_schedules.sql`
Expected: File exists

**Step 3: Commit**

```bash
git add backend/migrations/20260221000000_multiple_backup_schedules.sql
git commit -m "feat: add migration for multiple backup schedules"
```

---

## Task 2: Update BackupConfig Model

**Files:**
- Modify: `backend/internal/features/backups/config/model.go`

**Step 1: Update the model struct**

Change the `BackupConfig` struct to add `ID` as primary key and `Name` field:

```go
type BackupConfig struct {
	ID         uuid.UUID `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;not null"`

	// Human-readable name for the schedule
	Name string `json:"name" gorm:"column:name;type:text;not null"`

	IsBackupsEnabled bool `json:"isBackupsEnabled" gorm:"column:is_backups_enabled;type:boolean;not null"`

	RetentionPolicyType RetentionPolicyType `json:"retentionPolicyType" gorm:"column:retention_policy_type;type:text;not null;default:'TIME_PERIOD'"`
	RetentionTimePeriod period.TimePeriod   `json:"retentionTimePeriod" gorm:"column:retention_time_period;type:text;not null;default:''"`

	RetentionCount     int `json:"retentionCount"     gorm:"column:retention_count;type:int;not null;default:0"`
	RetentionGfsHours  int `json:"retentionGfsHours"  gorm:"column:retention_gfs_hours;type:int;not null;default:0"`
	RetentionGfsDays   int `json:"retentionGfsDays"   gorm:"column:retention_gfs_days;type:int;not null;default:0"`
	RetentionGfsWeeks  int `json:"retentionGfsWeeks"  gorm:"column:retention_gfs_weeks;type:int;not null;default:0"`
	RetentionGfsMonths int `json:"retentionGfsMonths" gorm:"column:retention_gfs_months;type:int;not null;default:0"`
	RetentionGfsYears  int `json:"retentionGfsYears"  gorm:"column:retention_gfs_years;type:int;not null;default:0"`

	BackupIntervalID uuid.UUID           `json:"backupIntervalId"         gorm:"column:backup_interval_id;type:uuid;not null"`
	BackupInterval   *intervals.Interval `json:"backupInterval,omitempty" gorm:"foreignKey:BackupIntervalID"`

	Storage   *storages.Storage `json:"storage"   gorm:"foreignKey:StorageID"`
	StorageID *uuid.UUID        `json:"storageId" gorm:"column:storage_id;type:uuid;"`

	SendNotificationsOn       []BackupNotificationType `json:"sendNotificationsOn" gorm:"-"`
	SendNotificationsOnString string                   `json:"-"                   gorm:"column:send_notifications_on;type:text;not null"`

	IsRetryIfFailed     bool `json:"isRetryIfFailed"     gorm:"column:is_retry_if_failed;type:boolean;not null"`
	MaxFailedTriesCount int  `json:"maxFailedTriesCount" gorm:"column:max_failed_tries_count;type:int;not null"`

	Encryption BackupEncryption `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`

	MaxBackupSizeMB       int64 `json:"maxBackupSizeMb"       gorm:"column:max_backup_size_mb;type:int;not null"`
	MaxBackupsTotalSizeMB int64 `json:"maxBackupsTotalSizeMb" gorm:"column:max_backups_total_size_mb;type:int;not null"`
}
```

**Step 2: Update Validate method to check name**

Add name validation in the `Validate` method:

```go
func (b *BackupConfig) Validate(plan *plans.DatabasePlan) error {
	if b.Name == "" {
		return errors.New("name is required")
	}

	if b.BackupIntervalID == uuid.Nil && b.BackupInterval == nil {
		return errors.New("backup interval is required")
	}

	// ... rest of validation remains unchanged
}
```

**Step 3: Update Copy method**

Update the `Copy` method to include the new fields:

```go
func (b *BackupConfig) Copy(newDatabaseID uuid.UUID) *BackupConfig {
	return &BackupConfig{
		DatabaseID:            newDatabaseID,
		Name:                  b.Name,
		IsBackupsEnabled:      b.IsBackupsEnabled,
		RetentionPolicyType:   b.RetentionPolicyType,
		RetentionTimePeriod:   b.RetentionTimePeriod,
		RetentionCount:        b.RetentionCount,
		RetentionGfsHours:     b.RetentionGfsHours,
		RetentionGfsDays:      b.RetentionGfsDays,
		RetentionGfsWeeks:     b.RetentionGfsWeeks,
		RetentionGfsMonths:    b.RetentionGfsMonths,
		RetentionGfsYears:     b.RetentionGfsYears,
		BackupIntervalID:      uuid.Nil,
		BackupInterval:        b.BackupInterval.Copy(),
		StorageID:             b.StorageID,
		SendNotificationsOn:   b.SendNotificationsOn,
		IsRetryIfFailed:       b.IsRetryIfFailed,
		MaxFailedTriesCount:   b.MaxFailedTriesCount,
		Encryption:            b.Encryption,
		MaxBackupSizeMB:       b.MaxBackupSizeMB,
		MaxBackupsTotalSizeMB: b.MaxBackupsTotalSizeMB,
	}
}
```

**Step 4: Run tests**

Run: `cd backend && go test ./internal/features/backups/config/... -v`
Expected: Tests pass (or update if needed)

**Step 5: Commit**

```bash
git add backend/internal/features/backups/config/model.go
git commit -m "feat: update BackupConfig model for multiple schedules"
```

---

## Task 3: Update BackupConfig Repository

**Files:**
- Modify: `backend/internal/features/backups/config/repository.go`

**Step 1: Add FindByID method**

```go
func (r *BackupConfigRepository) FindByID(id uuid.UUID) (*BackupConfig, error) {
	var backupConfig BackupConfig

	if err := storage.
		GetDb().
		Preload("BackupInterval").
		Preload("Storage").
		Where("id = ?", id).
		First(&backupConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &backupConfig, nil
}
```

**Step 2: Update FindByDatabaseID to return list**

Change the existing `FindByDatabaseID` to return a slice:

```go
func (r *BackupConfigRepository) FindByDatabaseID(databaseID uuid.UUID) ([]*BackupConfig, error) {
	var backupConfigs []*BackupConfig

	if err := storage.
		GetDb().
		Preload("BackupInterval").
		Preload("Storage").
		Where("database_id = ?", databaseID).
		Find(&backupConfigs).Error; err != nil {
		return nil, err
	}

	return backupConfigs, nil
}
```

**Step 3: Add Delete method**

```go
func (r *BackupConfigRepository) Delete(id uuid.UUID) error {
	result := storage.GetDb().Delete(&BackupConfig{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
```

**Step 4: Run tests**

Run: `cd backend && go test ./internal/features/backups/config/... -v`
Expected: Tests may fail due to service changes - continue to next task

**Step 5: Commit**

```bash
git add backend/internal/features/backups/config/repository.go
git commit -m "feat: update BackupConfigRepository for multiple schedules"
```

---

## Task 4: Update BackupConfig Service

**Files:**
- Modify: `backend/internal/features/backups/config/service.go`

**Step 1: Update GetBackupConfigByDbId to GetBackupConfigsByDatabaseID**

Change the method signature and implementation:

```go
func (s *BackupConfigService) GetBackupConfigsByDatabaseID(
	databaseID uuid.UUID,
) ([]*BackupConfig, error) {
	configs, err := s.backupConfigRepository.FindByDatabaseID(databaseID)
	if err != nil {
		return nil, err
	}

	if len(configs) == 0 {
		err = s.initializeDefaultConfig(databaseID)
		if err != nil {
			return nil, err
		}

		return s.backupConfigRepository.FindByDatabaseID(databaseID)
	}

	return configs, nil
}
```

**Step 2: Add GetBackupConfigByID method**

```go
func (s *BackupConfigService) GetBackupConfigByID(id uuid.UUID) (*BackupConfig, error) {
	return s.backupConfigRepository.FindByID(id)
}
```

**Step 3: Update GetBackupConfigByDbIdWithAuth**

```go
func (s *BackupConfigService) GetBackupConfigsByDatabaseIDWithAuth(
	user *users_models.User,
	databaseID uuid.UUID,
) ([]*BackupConfig, error) {
	_, err := s.databaseService.GetDatabase(user, databaseID)
	if err != nil {
		return nil, err
	}

	return s.GetBackupConfigsByDatabaseID(databaseID)
}
```

**Step 4: Add CreateBackupConfig method**

```go
func (s *BackupConfigService) CreateBackupConfig(
	user *users_models.User,
	backupConfig *BackupConfig,
) (*BackupConfig, error) {
	plan, err := s.databasePlanService.GetDatabasePlan(backupConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if err := backupConfig.Validate(plan); err != nil {
		return nil, err
	}

	database, err := s.databaseService.GetDatabase(user, backupConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot create backup config for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to create backup configuration")
	}

	if backupConfig.Storage != nil && backupConfig.Storage.ID != uuid.Nil {
		storage, err := s.storageService.GetStorageByID(backupConfig.Storage.ID)
		if err != nil {
			return nil, err
		}
		if storage.WorkspaceID != *database.WorkspaceID && !storage.IsSystem {
			return nil, errors.New("storage does not belong to the same workspace as the database")
		}
	}

	return s.backupConfigRepository.Save(backupConfig)
}
```

**Step 5: Add UpdateBackupConfig method**

```go
func (s *BackupConfigService) UpdateBackupConfig(
	user *users_models.User,
	id uuid.UUID,
	backupConfig *BackupConfig,
) (*BackupConfig, error) {
	existingConfig, err := s.backupConfigRepository.FindByID(id)
	if err != nil {
		return nil, err
	}

	if existingConfig == nil {
		return nil, errors.New("backup config not found")
	}

	plan, err := s.databasePlanService.GetDatabasePlan(existingConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if err := backupConfig.Validate(plan); err != nil {
		return nil, err
	}

	database, err := s.databaseService.GetDatabase(user, existingConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot update backup config for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to update backup configuration")
	}

	// Preserve the ID and DatabaseID
	backupConfig.ID = id
	backupConfig.DatabaseID = existingConfig.DatabaseID

	if backupConfig.Storage != nil && backupConfig.Storage.ID != uuid.Nil {
		storage, err := s.storageService.GetStorageByID(backupConfig.Storage.ID)
		if err != nil {
			return nil, err
		}
		if storage.WorkspaceID != *database.WorkspaceID && !storage.IsSystem {
			return nil, errors.New("storage does not belong to the same workspace as the database")
		}
	}

	// Check if storage is changing
	if s.dbStorageChangeListener != nil &&
		backupConfig.Storage != nil &&
		!storageIDsEqual(existingConfig.StorageID, &backupConfig.Storage.ID) {
		if err := s.dbStorageChangeListener.OnBeforeBackupsStorageChange(
			backupConfig.DatabaseID,
		); err != nil {
			return nil, err
		}
	}

	return s.backupConfigRepository.Save(backupConfig)
}
```

**Step 6: Add DeleteBackupConfig method**

```go
func (s *BackupConfigService) DeleteBackupConfig(
	user *users_models.User,
	id uuid.UUID,
) error {
	existingConfig, err := s.backupConfigRepository.FindByID(id)
	if err != nil {
		return err
	}

	if existingConfig == nil {
		return errors.New("backup config not found")
	}

	database, err := s.databaseService.GetDatabase(user, existingConfig.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot delete backup config for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to delete backup configuration")
	}

	return s.backupConfigRepository.Delete(id)
}
```

**Step 7: Update initializeDefaultConfig to set name**

```go
func (s *BackupConfigService) initializeDefaultConfig(
	databaseID uuid.UUID,
) error {
	plan, err := s.databasePlanService.GetDatabasePlan(databaseID)
	if err != nil {
		return err
	}

	timeOfDay := "04:00"

	_, err = s.backupConfigRepository.Save(&BackupConfig{
		DatabaseID:            databaseID,
		Name:                  "Default",
		IsBackupsEnabled:      false,
		RetentionPolicyType:   RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod:   plan.MaxStoragePeriod,
		MaxBackupSizeMB:       plan.MaxBackupSizeMB,
		MaxBackupsTotalSizeMB: plan.MaxBackupsTotalSizeMB,
		BackupInterval: &intervals.Interval{
			Interval:  intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
			NotificationBackupSuccess,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          BackupEncryptionNone,
	})

	return err
}
```

**Step 8: Run tests**

Run: `cd backend && go build ./...`
Expected: Build succeeds

**Step 9: Commit**

```bash
git add backend/internal/features/backups/config/service.go
git commit -m "feat: update BackupConfigService for multiple schedules"
```

---

## Task 5: Update BackupConfig Controller

**Files:**
- Modify: `backend/internal/features/backups/config/controller.go`

**Step 1: Update RegisterRoutes**

```go
func (c *BackupConfigController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/backup-configs/:id", c.GetBackupConfigByID)
	router.GET("/backup-configs/database/:databaseId", c.GetBackupConfigsByDatabaseID)
	router.POST("/backup-configs", c.CreateBackupConfig)
	router.PUT("/backup-configs/:id", c.UpdateBackupConfig)
	router.DELETE("/backup-configs/:id", c.DeleteBackupConfig)
	router.GET("/backup-configs/database/:id/plan", c.GetDatabasePlan)
	router.GET("/backup-configs/storage/:id/is-using", c.IsStorageUsing)
	router.GET("/backup-configs/storage/:id/databases-count", c.CountDatabasesForStorage)
	router.POST("/backup-configs/database/:id/transfer", c.TransferDatabase)
}
```

**Step 2: Add GetBackupConfigByID handler**

```go
// GetBackupConfigByID
// @Summary Get backup configuration by ID
// @Description Get a specific backup configuration by its ID
// @Tags backup-configs
// @Produce json
// @Param id path string true "Backup Config ID"
// @Success 200 {object} BackupConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /backup-configs/{id} [get]
func (c *BackupConfigController) GetBackupConfigByID(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup config ID"})
		return
	}

	backupConfig, err := c.backupConfigService.GetBackupConfigByID(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if backupConfig == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "backup configuration not found"})
		return
	}

	// Verify user has access to the database
	_, err = c.backupConfigService.GetDatabasePlan(user, backupConfig.DatabaseID)
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	ctx.JSON(http.StatusOK, backupConfig)
}
```

**Step 3: Update GetBackupConfigByDbID to GetBackupConfigsByDatabaseID**

```go
// GetBackupConfigsByDatabaseID
// @Summary Get backup configurations by database ID
// @Description Get all backup configurations for a specific database
// @Tags backup-configs
// @Produce json
// @Param databaseId path string true "Database ID"
// @Success 200 {array} BackupConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /backup-configs/database/{databaseId} [get]
func (c *BackupConfigController) GetBackupConfigsByDatabaseID(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("databaseId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	backupConfigs, err := c.backupConfigService.GetBackupConfigsByDatabaseIDWithAuth(user, databaseID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, backupConfigs)
}
```

**Step 4: Add CreateBackupConfig handler**

```go
// CreateBackupConfig
// @Summary Create a new backup configuration
// @Description Create a new backup configuration for a database
// @Tags backup-configs
// @Accept json
// @Produce json
// @Param request body BackupConfig true "Backup configuration data"
// @Success 201 {object} BackupConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /backup-configs [post]
func (c *BackupConfigController) CreateBackupConfig(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var requestDTO BackupConfig
	if err := ctx.ShouldBindJSON(&requestDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Clear StorageID to rely on full Storage object
	requestDTO.StorageID = nil

	savedConfig, err := c.backupConfigService.CreateBackupConfig(user, &requestDTO)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, savedConfig)
}
```

**Step 5: Add UpdateBackupConfig handler**

```go
// UpdateBackupConfig
// @Summary Update a backup configuration
// @Description Update an existing backup configuration
// @Tags backup-configs
// @Accept json
// @Produce json
// @Param id path string true "Backup Config ID"
// @Param request body BackupConfig true "Backup configuration data"
// @Success 200 {object} BackupConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /backup-configs/{id} [put]
func (c *BackupConfigController) UpdateBackupConfig(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup config ID"})
		return
	}

	var requestDTO BackupConfig
	if err := ctx.ShouldBindJSON(&requestDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Clear StorageID to rely on full Storage object
	requestDTO.StorageID = nil

	savedConfig, err := c.backupConfigService.UpdateBackupConfig(user, id, &requestDTO)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, savedConfig)
}
```

**Step 6: Add DeleteBackupConfig handler**

```go
// DeleteBackupConfig
// @Summary Delete a backup configuration
// @Description Delete an existing backup configuration
// @Tags backup-configs
// @Param id path string true "Backup Config ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /backup-configs/{id} [delete]
func (c *BackupConfigController) DeleteBackupConfig(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup config ID"})
		return
	}

	if err := c.backupConfigService.DeleteBackupConfig(user, id); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}
```

**Step 7: Remove old SaveBackupConfig handler**

Delete the `SaveBackupConfig` method as it's replaced by `CreateBackupConfig` and `UpdateBackupConfig`.

**Step 8: Run build**

Run: `cd backend && go build ./...`
Expected: Build succeeds

**Step 9: Commit**

```bash
git add backend/internal/features/backups/config/controller.go
git commit -m "feat: update BackupConfigController for multiple schedules"
```

---

## Task 6: Update Backup Model

**Files:**
- Modify: `backend/internal/features/backups/backups/core/model.go`

**Step 1: Add BackupConfigID field**

```go
type Backup struct {
	ID       uuid.UUID `json:"id"       gorm:"column:id;type:uuid;primaryKey"`
	FileName string    `json:"fileName" gorm:"column:file_name;type:text;not null"`

	DatabaseID      uuid.UUID `json:"databaseId"      gorm:"column:database_id;type:uuid;not null"`
	BackupConfigID  *uuid.UUID `json:"backupConfigId" gorm:"column:backup_config_id;type:uuid"`
	StorageID       uuid.UUID `json:"storageId"       gorm:"column:storage_id;type:uuid;not null"`

	Status      BackupStatus `json:"status"      gorm:"column:status;not null"`
	FailMessage *string      `json:"failMessage" gorm:"column:fail_message"`
	IsSkipRetry bool         `json:"isSkipRetry" gorm:"column:is_skip_retry;type:boolean;not null"`

	BackupSizeMb float64 `json:"backupSizeMb" gorm:"column:backup_size_mb;default:0"`

	BackupDurationMs int64 `json:"backupDurationMs" gorm:"column:backup_duration_ms;default:0"`

	EncryptionSalt *string                         `json:"-"          gorm:"column:encryption_salt"`
	EncryptionIV   *string                         `json:"-"          gorm:"column:encryption_iv"`
	Encryption     backups_config.BackupEncryption `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`

	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
}
```

**Step 2: Run build**

Run: `cd backend && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add backend/internal/features/backups/backups/core/model.go
git commit -m "feat: add backup_config_id to Backup model"
```

---

## Task 7: Update Scheduler to Use BackupConfigID

**Files:**
- Modify: `backend/internal/features/backups/backups/backuping/scheduler.go`
- Modify: `backend/internal/features/backups/backups/backuping/backuper.go`

**Step 1: Update StartBackup to accept BackupConfig**

In `scheduler.go`, find the `StartBackup` method and update it to pass the backup config ID to the backup creation.

**Step 2: Update backuper to set BackupConfigID**

In `backuper.go`, update the backup creation to include `BackupConfigID`.

**Step 3: Run build and tests**

Run: `cd backend && go build ./... && go test ./internal/features/backups/... -v`
Expected: Build succeeds, tests may need updates

**Step 4: Commit**

```bash
git add backend/internal/features/backups/backups/backuping/
git commit -m "feat: update scheduler to track backup config ID"
```

---

## Task 8: Update Cleaner for Multiple Configs

**Files:**
- Modify: `backend/internal/features/backups/backups/backuping/cleaner.go`

**Step 1: Update cleaner to handle backups per config**

The cleaner needs to apply retention policies per backup config, not per database.

**Step 2: Run tests**

Run: `cd backend && go test ./internal/features/backups/backups/backuping/... -v`
Expected: Tests pass

**Step 3: Commit**

```bash
git add backend/internal/features/backups/backups/backuping/cleaner.go
git commit -m "feat: update cleaner for multiple backup configs"
```

---

## Task 9: Write Controller Tests

**Files:**
- Modify: `backend/internal/features/backups/config/controller_test.go`

**Step 1: Write test for CreateBackupConfig**

```go
func Test_CreateBackupConfig_ValidInput_ConfigCreated(t *testing.T) {
	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test", user)
	database := databases_testing.CreateTestDatabase("TestDB", workspace.ID, user, router)

	request := backups_config.BackupConfig{
		DatabaseID:         database.ID,
		Name:              "Hourly Backup",
		IsBackupsEnabled:  true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodDay,
		BackupInterval: &intervals.Interval{
			Interval: intervals.IntervalHourly,
		},
		SendNotificationsOn: []backups_config.BackupNotificationType{
			backups_config.NotificationBackupFailed,
		},
	}

	var response backups_config.BackupConfig
	test_utils.MakePostRequestAndUnmarshal(t, router, "/api/v1/backup-configs", 
		"Bearer "+user.Token, request, http.StatusCreated, &response)

	assert.Equal(t, "Hourly Backup", response.Name)
	assert.NotEqual(t, uuid.Nil, response.ID)
}
```

**Step 2: Write test for multiple configs on same database**

```go
func Test_CreateBackupConfig_MultipleConfigsForSameDatabase_BothCreated(t *testing.T) {
	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test", user)
	database := databases_testing.CreateTestDatabase("TestDB", workspace.ID, user, router)

	// Create first config
	request1 := backups_config.BackupConfig{
		DatabaseID:         database.ID,
		Name:              "Hourly",
		IsBackupsEnabled:  true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodDay,
		BackupInterval: &intervals.Interval{Interval: intervals.IntervalHourly},
	}

	var response1 backups_config.BackupConfig
	test_utils.MakePostRequestAndUnmarshal(t, router, "/api/v1/backup-configs", 
		"Bearer "+user.Token, request1, http.StatusCreated, &response1)

	// Create second config
	request2 := backups_config.BackupConfig{
		DatabaseID:         database.ID,
		Name:              "Weekly",
		IsBackupsEnabled:  true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodMonth,
		BackupInterval: &intervals.Interval{Interval: intervals.IntervalWeekly},
	}

	var response2 backups_config.BackupConfig
	test_utils.MakePostRequestAndUnmarshal(t, router, "/api/v1/backup-configs", 
		"Bearer "+user.Token, request2, http.StatusCreated, &response2)

	// Verify both exist
	assert.NotEqual(t, response1.ID, response2.ID)
	assert.Equal(t, "Hourly", response1.Name)
	assert.Equal(t, "Weekly", response2.Name)
}
```

**Step 3: Write test for GetBackupConfigsByDatabaseID**

```go
func Test_GetBackupConfigsByDatabaseID_ReturnsAllConfigs(t *testing.T) {
	router := createTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test", user)
	database := databases_testing.CreateTestDatabase("TestDB", workspace.ID, user, router)

	// Create two configs
	createBackupConfig(t, router, database.ID, "Hourly", user.Token)
	createBackupConfig(t, router, database.ID, "Weekly", user.Token)

	var response []*backups_config.BackupConfig
	test_utils.MakeGetRequestAndUnmarshal(t, router, 
		fmt.Sprintf("/api/v1/backup-configs/database/%s", database.ID),
		"Bearer "+user.Token, http.StatusOK, &response)

	assert.Equal(t, 2, len(response))
}
```

**Step 4: Run tests**

Run: `cd backend && go test ./internal/features/backups/config/... -v`
Expected: Tests pass

**Step 5: Commit**

```bash
git add backend/internal/features/backups/config/controller_test.go
git commit -m "test: add tests for multiple backup configs"
```

---

## Task 10: Run All Tests and Final Verification

**Step 1: Run all backend tests**

Run: `cd backend && go test ./... -v`
Expected: All tests pass

**Step 2: Run linter**

Run: `cd backend && make lint`
Expected: No errors

**Step 3: Build the application**

Run: `cd backend && go build ./...`
Expected: Build succeeds

**Step 4: Final commit**

```bash
git add .
git commit -m "feat: complete multiple backup schedules implementation"
```

---

## Summary

This implementation plan covers:

1. **Database Migration** - Changes primary key and adds name column
2. **Model Updates** - BackupConfig gets ID and Name fields
3. **Repository Updates** - New methods for finding by ID, list by database, delete
4. **Service Updates** - CRUD operations for backup configs
5. **Controller Updates** - RESTful endpoints for managing configs
6. **Backup Model** - Links backups to their config
7. **Scheduler/Cleaner** - Updates to work with multiple configs
8. **Tests** - Verify the feature works correctly

After implementation, users can:
- Create multiple backup schedules per database
- Each schedule has its own interval, retention, and storage
- View all schedules for a database in a list
- Update and delete individual schedules
