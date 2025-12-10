package clusters

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	backups "postgresus-backend/internal/features/backups/backups"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/databases/databases/postgresql"
	"postgresus-backend/internal/features/intervals"
	users_enums "postgresus-backend/internal/features/users/enums"
	users_models "postgresus-backend/internal/features/users/models"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
	"postgresus-backend/internal/util/period"

	"github.com/google/uuid"
)

type ClusterService struct {
	repo                *ClusterRepository
	dbService           *databases.DatabaseService
	backupService       *backups.BackupService
	backupConfigService *backups_config.BackupConfigService
	workspaceService    *workspaces_services.WorkspaceService
}

// discoverClusterDatabases lists accessible databases on the cluster connection
func (s *ClusterService) discoverClusterDatabases(pg *PostgresqlCluster) ([]string, error) {
	p := &postgresql.PostgresqlDatabase{
		Version:  pg.Version,
		Host:     pg.Host,
		Port:     pg.Port,
		Username: pg.Username,
		Password: pg.Password,
		IsHttps:  pg.IsHttps,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	return p.ListAccessibleDatabases(logger)
}

// isSystemDb returns true for Postgres system databases
func isSystemDb(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "postgres", "template0", "template1":
		return true
	default:
		return false
	}
}

// findDb finds an existing DB matching the cluster connection and database name
func findDb(existing []*databases.Database, pg *PostgresqlCluster, dbName string) *databases.Database {
	for _, d := range existing {
		if d == nil || d.Postgresql == nil || d.Type != databases.DatabaseTypePostgres {
			continue
		}
		if d.Postgresql.Database == nil {
			continue
		}
		if d.Postgresql.Host == pg.Host && d.Postgresql.Port == pg.Port && d.Postgresql.Username == pg.Username && d.Postgresql.IsHttps == pg.IsHttps {
			if strings.EqualFold(*d.Postgresql.Database, dbName) {
				return d
			}
		}
	}
	return nil
}

// RunBackupScheduled runs cluster backup as a system task (admin bypass), used by background scheduler.
func (s *ClusterService) RunBackupScheduled(clusterID uuid.UUID) error {
	sys := &users_models.User{Role: users_enums.UserRoleAdmin}
	return s.RunBackup(sys, clusterID)
}

// configureClusterDatabasesForScheduling discovers databases and ensures Database + BackupConfig exist,
// but DOES NOT trigger actual backups. It honors cluster exclusions.
func (s *ClusterService) configureClusterDatabasesForScheduling(
	user *users_models.User,
	cluster *Cluster,
) error {
	// Discover databases on the cluster
	dbNames, err := s.discoverClusterDatabases(&cluster.Postgresql)
	if err != nil {
		return fmt.Errorf("failed to discover databases in cluster: %w", err)
	}

	if cluster.WorkspaceID == nil {
		return errors.New("cluster is not bound to a workspace")
	}

	// Load existing databases for workspace
	existingDbs, err := s.dbService.GetDatabasesByWorkspace(user, *cluster.WorkspaceID)
	if err != nil {
		return err
	}

	// Build excluded set
	excluded := map[string]struct{}{}
	for _, ed := range cluster.ExcludedDatabases {
		if ed.Name == "" {
			continue
		}
		excluded[strings.ToLower(ed.Name)] = struct{}{}
	}

	// Concurrency limit for triggering backups
	const maxParallel = 5
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup

	// Create any missing databases and ensure backup config
	for _, dbName := range dbNames {
		if isSystemDb(dbName) {
			continue
		}
		if _, skip := excluded[strings.ToLower(dbName)]; skip {
			continue
		}

		if db := findDb(existingDbs, &cluster.Postgresql, dbName); db == nil {
			// create database
			created, err := s.createDatabaseForCluster(user, cluster, dbName)
			if err != nil {
				return fmt.Errorf("failed to create database '%s' for cluster: %w", dbName, err)
			}

			// ensure backup config according to cluster defaults
			if err := s.ensureBackupConfig(created.ID, cluster); err != nil {
				return fmt.Errorf("failed to create backup config for '%s': %w", dbName, err)
			}

			// trigger backup if enabled
			cfg, err := s.backupConfigService.GetBackupConfigByDbId(created.ID)
			if err == nil && cfg != nil && cfg.IsBackupsEnabled {
				wg.Add(1)
				sem <- struct{}{}
				go func(dbid uuid.UUID) {
					defer wg.Done()
					defer func() { <-sem }()
					s.backupService.MakeBackup(dbid, true)
				}(created.ID)
			}
		} else {
			// existing DB: ensure backup config exists and apply cluster storage if missing
			if err := s.ensureBackupConfig(db.ID, cluster); err != nil {
				return fmt.Errorf("failed to ensure backup config for existing DB '%s': %w", dbName, err)
			}
			cfg, err := s.backupConfigService.GetBackupConfigByDbId(db.ID)
			if err == nil && cfg != nil && cfg.IsBackupsEnabled {
				wg.Add(1)
				sem <- struct{}{}
				go func(dbid uuid.UUID) {
					defer wg.Done()
					defer func() { <-sem }()
					s.backupService.MakeBackup(dbid, true)
				}(db.ID)
			}
		}
	}

	wg.Wait()
	return nil
}

func (s *ClusterService) CreateCluster(
	user *users_models.User,
	workspaceID uuid.UUID,
	cluster *Cluster,
) (*Cluster, error) {
	canManage, err := s.workspaceService.CanUserManageDBs(workspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to create cluster in this workspace")
	}

	cluster.WorkspaceID = &workspaceID
	// Validation: if backups are enabled, storage must be set
	if cluster.IsBackupsEnabled && cluster.StorageID == nil {
		return nil, errors.New("storage must be selected when backups are enabled for cluster")
	}
	if err := cluster.ValidateBasic(); err != nil {
		return nil, err
	}

	saved, err := s.repo.Save(cluster)
	if err != nil {
		return nil, err
	}

	// Configure scheduled backups for existing databases in the cluster (non-blocking)
	go func() {
		_ = s.configureClusterDatabasesForScheduling(user, saved)
	}()

	return saved, nil
}

func (s *ClusterService) GetClusters(
	user *users_models.User,
	workspaceID uuid.UUID,
) ([]*Cluster, error) {
	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(workspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to access this workspace")
	}

	clusters, err := s.repo.FindByWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}

	for _, c := range clusters {
		c.HideSensitiveData()
	}

	return clusters, nil
}

func (s *ClusterService) UpdateCluster(
	user *users_models.User,
	id uuid.UUID,
	updated *Cluster,
) (*Cluster, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if existing.WorkspaceID == nil {
		return nil, errors.New("cluster is not bound to a workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*existing.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to update this cluster")
	}

	// Preserve identifiers
	updated.ID = id
	updated.WorkspaceID = existing.WorkspaceID

	// Merge Postgresql settings, preserve password if empty
	updated.Postgresql.ID = existing.Postgresql.ID
	updated.Postgresql.ClusterID = existing.ID
	if strings.TrimSpace(updated.Postgresql.Password) == "" {
		updated.Postgresql.Password = existing.Postgresql.Password
	}

	// Validate
	// Validation: if backups are enabled, storage must be set
	if updated.IsBackupsEnabled && updated.StorageID == nil {
		return nil, errors.New("storage must be selected when backups are enabled for cluster")
	}
	if err := updated.ValidateBasic(); err != nil {
		return nil, err
	}

	if existing.WorkspaceID != nil {
		oldExcluded := map[string]struct{}{}
		for _, ed := range existing.ExcludedDatabases {
			if ed.Name == "" {
				continue
			}
			oldExcluded[strings.ToLower(ed.Name)] = struct{}{}
		}

		addedExcluded := make([]string, 0)
		for _, ed := range updated.ExcludedDatabases {
			if ed.Name == "" {
				continue
			}
			nameLower := strings.ToLower(ed.Name)
			if _, was := oldExcluded[nameLower]; !was {
				addedExcluded = append(addedExcluded, nameLower)
			}
		}

		if len(addedExcluded) > 0 {
			dbs, err := s.dbService.GetDatabasesByWorkspace(user, *existing.WorkspaceID)
			if err != nil {
				return nil, err
			}

			for _, d := range dbs {
				if d.Type != databases.DatabaseTypePostgres || d.Postgresql == nil || d.Postgresql.Database == nil {
					continue
				}
				if !s.matchesClusterConn(d, &existing.Postgresql) {
					continue
				}
				dbName := strings.ToLower(*d.Postgresql.Database)
				for _, excludedName := range addedExcluded {
					if dbName == excludedName {
						cfg, err := s.backupConfigService.FindBackupConfigByDbIdNoInit(d.ID)
						if err != nil {
							return nil, fmt.Errorf("failed to load backup config for excluded database '%s': %w", *d.Postgresql.Database, err)
						}
						if cfg != nil {
							cfg.IsBackupsEnabled = false
							cfg.ManagedByCluster = false
							cfg.ClusterID = nil
							if _, err := s.backupConfigService.SaveBackupConfig(cfg); err != nil {
								return nil, fmt.Errorf("failed to disable backup config for excluded database '%s': %w", *d.Postgresql.Database, err)
							}
						}
						break
					}
				}
			}
		}
	}

	saved, err := s.repo.Save(updated)
	if err != nil {
		return nil, err
	}
	saved.HideSensitiveData()
	return saved, nil
}

func (s *ClusterService) ListClusterDatabases(
	user *users_models.User,
	clusterID uuid.UUID,
) ([]string, error) {
	cluster, err := s.repo.FindByID(clusterID)
	if err != nil {
		return nil, err
	}
	if cluster.WorkspaceID == nil {
		return nil, errors.New("cluster is not bound to a workspace")
	}
	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(*cluster.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to access this workspace")
	}

	dbs, err := s.discoverClusterDatabases(&cluster.Postgresql)
	if err != nil {
		return nil, err
	}
	return dbs, nil
}

func (s *ClusterService) RunBackup(
	user *users_models.User,
	clusterID uuid.UUID,
) error {
	cluster, err := s.repo.FindByID(clusterID)
	if err != nil {
		return err
	}

	if cluster.WorkspaceID == nil {
		return errors.New("cluster is not bound to a workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*cluster.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to run backup for this cluster")
	}

	// Discover databases on the cluster
	dbNames, err := s.discoverClusterDatabases(&cluster.Postgresql)
	if err != nil {
		return fmt.Errorf("failed to discover databases in cluster: %w", err)
	}

	// Load existing databases for workspace to avoid duplicate creation
	existingDbs, err := s.dbService.GetDatabasesByWorkspace(user, *cluster.WorkspaceID)
	if err != nil {
		return err
	}

	// Build excluded set
	excluded := map[string]struct{}{}
	for _, ed := range cluster.ExcludedDatabases {
		if ed.Name == "" {
			continue
		}
		excluded[strings.ToLower(ed.Name)] = struct{}{}
	}

	// Concurrency limit for triggering backups
	const maxParallel = 5
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup

	// Create any missing databases and ensure backup config
	for _, dbName := range dbNames {
		if isSystemDb(dbName) {
			continue
		}
		if _, skip := excluded[strings.ToLower(dbName)]; skip {
			continue
		}

		if db := findDb(existingDbs, &cluster.Postgresql, dbName); db == nil {
			// create database
			created, err := s.createDatabaseForCluster(user, cluster, dbName)
			if err != nil {
				return fmt.Errorf("failed to create database '%s' for cluster: %w", dbName, err)
			}

			// ensure backup config according to cluster defaults
			if err := s.ensureBackupConfig(created.ID, cluster); err != nil {
				return fmt.Errorf("failed to create backup config for '%s': %w", dbName, err)
			}

			// trigger backup if enabled
			cfg, err := s.backupConfigService.GetBackupConfigByDbId(created.ID)
			if err == nil && cfg != nil && cfg.IsBackupsEnabled {
				wg.Add(1)
				sem <- struct{}{}
				go func(dbid uuid.UUID) {
					defer wg.Done()
					defer func() { <-sem }()
					s.backupService.MakeBackup(dbid, true)
				}(created.ID)
			}
		} else {
			// Existing DB: ensure config reflects cluster defaults when appropriate, then trigger if enabled
			if err := s.ensureBackupConfig(db.ID, cluster); err != nil {
				return fmt.Errorf("failed to ensure backup config for existing DB '%s': %w", dbName, err)
			}
			cfg, err := s.backupConfigService.GetBackupConfigByDbId(db.ID)
			if err == nil && cfg != nil && cfg.IsBackupsEnabled {
				wg.Add(1)
				sem <- struct{}{}
				go func(dbid uuid.UUID) {
					defer wg.Done()
					defer func() { <-sem }()
					s.backupService.MakeBackup(dbid, true)
				}(db.ID)
			}
		}
	}

	wg.Wait()
	return nil
}

func (s *ClusterService) createDatabaseForCluster(
	user *users_models.User,
	cluster *Cluster,
	dbName string,
) (*databases.Database, error) {
	pg := cluster.Postgresql

	db := &databases.Database{
		ID:          uuid.Nil,
		WorkspaceID: cluster.WorkspaceID,
		Name:        dbName,
		Type:        databases.DatabaseTypePostgres,
		Notifiers:   cluster.Notifiers,
		Postgresql: &postgresql.PostgresqlDatabase{
			ID:       uuid.Nil,
			Version:  pg.Version,
			Host:     pg.Host,
			Port:     pg.Port,
			Username: pg.Username,
			Password: pg.Password,
			Database: &dbName,
			IsHttps:  pg.IsHttps,
		},
	}

	// databaseService.CreateDatabase expects workspace ID not nil
	return s.dbService.CreateDatabase(user, *cluster.WorkspaceID, db)
}

func (s *ClusterService) ensureBackupConfig(
	databaseID uuid.UUID,
	cluster *Cluster,
) error {
	cfg, err := s.backupConfigService.FindBackupConfigByDbIdNoInit(databaseID)
	if err != nil {
		return err
	}
	if cfg != nil {
		// Upgrade default-initialized config to cluster defaults
		if isDefaultBackupConfig(cfg) {
			var intervalObj *intervals.Interval
			var intervalID uuid.UUID
			if cluster.BackupInterval != nil {
				if cluster.BackupInterval.ID != uuid.Nil {
					intervalID = cluster.BackupInterval.ID
				} else {
					intervalObj = &intervals.Interval{
						ID:         uuid.Nil,
						Interval:   cluster.BackupInterval.Interval,
						TimeOfDay:  cluster.BackupInterval.TimeOfDay,
						Weekday:    cluster.BackupInterval.Weekday,
						DayOfMonth: cluster.BackupInterval.DayOfMonth,
					}
				}
			} else if cluster.BackupIntervalID != nil {
				intervalID = *cluster.BackupIntervalID
			} else {
				timeOfDay := "04:00"
				intervalObj = &intervals.Interval{Interval: intervals.IntervalDaily, TimeOfDay: &timeOfDay}
			}

			cfg.IsBackupsEnabled = cluster.IsBackupsEnabled
			cfg.StorePeriod = cluster.StorePeriod
			cfg.BackupIntervalID = intervalID
			cfg.BackupInterval = intervalObj
			cfg.StorageID = cluster.StorageID
			cfg.SendNotificationsOn = parseNotifications(cluster.SendNotificationsOn)
			cfg.IsRetryIfFailed = true
			cfg.MaxFailedTriesCount = 3
			cfg.CpuCount = cluster.CpuCount
			// mark as cluster-managed
			cfg.ManagedByCluster = true
			cfg.ClusterID = &cluster.ID

			_, err = s.backupConfigService.SaveBackupConfig(cfg)
			return err
		}
		return nil
	}

	// Build backup config from cluster defaults
	var intervalObj *intervals.Interval
	var intervalID uuid.UUID
	if cluster.BackupInterval != nil {
		// If cluster has an interval object, reference its ID if present, else copy the object
		if cluster.BackupInterval.ID != uuid.Nil {
			intervalID = cluster.BackupInterval.ID
		} else {
			// Copy object; repository will create it for backup config
			intervalObj = &intervals.Interval{
				ID:         uuid.Nil,
				Interval:   cluster.BackupInterval.Interval,
				TimeOfDay:  cluster.BackupInterval.TimeOfDay,
				Weekday:    cluster.BackupInterval.Weekday,
				DayOfMonth: cluster.BackupInterval.DayOfMonth,
			}
		}
	} else if cluster.BackupIntervalID != nil {
		intervalID = *cluster.BackupIntervalID
	} else {
		// fallback to daily at 04:00
		timeOfDay := "04:00"
		intervalObj = &intervals.Interval{Interval: intervals.IntervalDaily, TimeOfDay: &timeOfDay}
	}

	notifs := parseNotifications(cluster.SendNotificationsOn)

	newCfg := &backups_config.BackupConfig{
		DatabaseID:          databaseID,
		IsBackupsEnabled:    cluster.IsBackupsEnabled,
		StorePeriod:         cluster.StorePeriod,
		BackupIntervalID:    intervalID,
		BackupInterval:      intervalObj,
		StorageID:           cluster.StorageID,
		SendNotificationsOn: notifs,
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		CpuCount:            cluster.CpuCount,
		ManagedByCluster:    true,
		ClusterID:           &cluster.ID,
	}

	_, err = s.backupConfigService.SaveBackupConfig(newCfg)
	return err
}

func (s *ClusterService) PreviewPropagation(
	user *users_models.User,
	clusterID uuid.UUID,
	opts PropagationOptions,
) ([]PropagationChange, error) {
	cluster, err := s.repo.FindByID(clusterID)
	if err != nil {
		return nil, err
	}
	if cluster.WorkspaceID == nil {
		return nil, errors.New("cluster is not bound to a workspace")
	}
	canManage, err := s.workspaceService.CanUserManageDBs(*cluster.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to update this cluster")
	}

	dbs, err := s.dbService.GetDatabasesByWorkspace(user, *cluster.WorkspaceID)
	if err != nil {
		return nil, err
	}

	excluded := map[string]struct{}{}
	if opts.RespectExclusions {
		for _, ed := range cluster.ExcludedDatabases {
			if ed.Name == "" {
				continue
			}
			excluded[strings.ToLower(ed.Name)] = struct{}{}
		}
	}

	clusterIntID, clusterIntObj := s.clusterScheduleToIntervalIDAndObj(cluster)

	res := make([]PropagationChange, 0)
	for _, d := range dbs {
		if d.Type != databases.DatabaseTypePostgres || d.Postgresql == nil {
			continue
		}
		if !s.matchesClusterConn(d, &cluster.Postgresql) {
			continue
		}
		if d.Postgresql.Database == nil || isSystemDb(*d.Postgresql.Database) {
			continue
		}
		if opts.RespectExclusions {
			if _, ok := excluded[strings.ToLower(*d.Postgresql.Database)]; ok {
				continue
			}
		}

		cfg, err := s.backupConfigService.GetBackupConfigByDbId(d.ID)
		if err != nil || cfg == nil {
			continue
		}

		ch := PropagationChange{DatabaseID: d.ID, Name: *d.Postgresql.Database}
		if opts.ApplyStorage && cluster.StorageID != nil {
			if cfg.StorageID == nil || *cfg.StorageID != *cluster.StorageID {
				ch.ChangeStorage = true
			}
		}
		if opts.ApplySchedule {
			if clusterIntID != uuid.Nil {
				if cfg.BackupIntervalID == uuid.Nil || cfg.BackupIntervalID != clusterIntID {
					ch.ChangeSchedule = true
				}
			} else if clusterIntObj != nil {
				if cfg.BackupInterval == nil || !s.intervalsEqual(cfg.BackupInterval, clusterIntObj) {
					ch.ChangeSchedule = true
				}
			}
		}

		if opts.ApplyEnableBackups {
			if cfg.IsBackupsEnabled != cluster.IsBackupsEnabled {
				ch.ChangeEnabled = true
			}
		}

		if ch.ChangeStorage || ch.ChangeSchedule || ch.ChangeEnabled {
			res = append(res, ch)
		}
	}

	return res, nil
}

func (s *ClusterService) ApplyPropagation(
	user *users_models.User,
	clusterID uuid.UUID,
	opts PropagationOptions,
) ([]PropagationChange, error) {
	cluster, err := s.repo.FindByID(clusterID)
	if err != nil {
		return nil, err
	}
	if cluster.WorkspaceID == nil {
		return nil, errors.New("cluster is not bound to a workspace")
	}
	canManage, err := s.workspaceService.CanUserManageDBs(*cluster.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to update this cluster")
	}

	dbs, err := s.dbService.GetDatabasesByWorkspace(user, *cluster.WorkspaceID)
	if err != nil {
		return nil, err
	}

	excluded := map[string]struct{}{}
	if opts.RespectExclusions {
		for _, ed := range cluster.ExcludedDatabases {
			if ed.Name == "" {
				continue
			}
			excluded[strings.ToLower(ed.Name)] = struct{}{}
		}
	}

	clusterIntID, clusterIntObj := s.clusterScheduleToIntervalIDAndObj(cluster)

	applied := make([]PropagationChange, 0)
	for _, d := range dbs {
		if d.Type != databases.DatabaseTypePostgres || d.Postgresql == nil {
			continue
		}
		if !s.matchesClusterConn(d, &cluster.Postgresql) {
			continue
		}
		if d.Postgresql.Database == nil || isSystemDb(*d.Postgresql.Database) {
			continue
		}
		if opts.RespectExclusions {
			if _, ok := excluded[strings.ToLower(*d.Postgresql.Database)]; ok {
				continue
			}
		}

		cfg, err := s.backupConfigService.GetBackupConfigByDbId(d.ID)
		if err != nil || cfg == nil {
			continue
		}

		ch := PropagationChange{DatabaseID: d.ID, Name: *d.Postgresql.Database}
		if opts.ApplyStorage && cluster.StorageID != nil {
			if cfg.StorageID == nil || *cfg.StorageID != *cluster.StorageID {
				cfg.StorageID = cluster.StorageID
				ch.ChangeStorage = true
			}
		}
		if opts.ApplySchedule {
			if clusterIntID != uuid.Nil {
				if cfg.BackupIntervalID == uuid.Nil || cfg.BackupIntervalID != clusterIntID {
					cfg.BackupIntervalID = clusterIntID
					cfg.BackupInterval = nil
					ch.ChangeSchedule = true
				}
			} else if clusterIntObj != nil {
				if cfg.BackupInterval == nil || !s.intervalsEqual(cfg.BackupInterval, clusterIntObj) {
					t := clusterIntObj.TimeOfDay
					dom := clusterIntObj.DayOfMonth
					wd := clusterIntObj.Weekday
					cfg.BackupIntervalID = uuid.Nil
					cfg.BackupInterval = &intervals.Interval{
						ID:         uuid.Nil,
						Interval:   clusterIntObj.Interval,
						TimeOfDay:  t,
						Weekday:    wd,
						DayOfMonth: dom,
					}
					ch.ChangeSchedule = true
				}
			}
		}

		if opts.ApplyEnableBackups {
			if cfg.IsBackupsEnabled != cluster.IsBackupsEnabled {
				cfg.IsBackupsEnabled = cluster.IsBackupsEnabled
				ch.ChangeEnabled = true
			}
		}

		if ch.ChangeStorage || ch.ChangeSchedule || ch.ChangeEnabled {
			// mark as cluster-managed when applying cluster defaults
			cfg.ManagedByCluster = true
			cfg.ClusterID = &cluster.ID
			if _, err := s.backupConfigService.SaveBackupConfig(cfg); err != nil {
				return nil, fmt.Errorf("failed to apply changes to DB '%s': %w", ch.Name, err)
			}
			applied = append(applied, ch)
		}
	}

	return applied, nil
}

func (s *ClusterService) matchesClusterConn(d *databases.Database, pg *PostgresqlCluster) bool {
	if d.Postgresql == nil {
		return false
	}
	return d.Postgresql.Host == pg.Host && d.Postgresql.Port == pg.Port && d.Postgresql.Username == pg.Username && d.Postgresql.IsHttps == pg.IsHttps
}

func (s *ClusterService) clusterScheduleToIntervalIDAndObj(cluster *Cluster) (uuid.UUID, *intervals.Interval) {
	if cluster.BackupInterval != nil {
		if cluster.BackupInterval.ID != uuid.Nil {
			return cluster.BackupInterval.ID, nil
		}
		t := cluster.BackupInterval.TimeOfDay
		dom := cluster.BackupInterval.DayOfMonth
		wd := cluster.BackupInterval.Weekday
		return uuid.Nil, &intervals.Interval{ID: uuid.Nil, Interval: cluster.BackupInterval.Interval, TimeOfDay: t, Weekday: wd, DayOfMonth: dom}
	}
	if cluster.BackupIntervalID != nil {
		return *cluster.BackupIntervalID, nil
	}
	return uuid.Nil, nil
}

func (s *ClusterService) intervalsEqual(a, b *intervals.Interval) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Interval != b.Interval {
		return false
	}
	if (a.TimeOfDay == nil) != (b.TimeOfDay == nil) {
		return false
	}
	if a.TimeOfDay != nil && b.TimeOfDay != nil && *a.TimeOfDay != *b.TimeOfDay {
		return false
	}
	if (a.Weekday == nil) != (b.Weekday == nil) {
		return false
	}
	if a.Weekday != nil && b.Weekday != nil && *a.Weekday != *b.Weekday {
		return false
	}
	if (a.DayOfMonth == nil) != (b.DayOfMonth == nil) {
		return false
	}
	if a.DayOfMonth != nil && b.DayOfMonth != nil && *a.DayOfMonth != *b.DayOfMonth {
		return false
	}
	return true
}

// parseNotifications parses a comma-separated list into BackupNotificationType slice
func parseNotifications(slist string) []backups_config.BackupNotificationType {
	if strings.TrimSpace(slist) == "" {
		return []backups_config.BackupNotificationType{}
	}
	parts := strings.Split(slist, ",")
	res := make([]backups_config.BackupNotificationType, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		switch backups_config.BackupNotificationType(v) {
		case backups_config.NotificationBackupFailed, backups_config.NotificationBackupSuccess:
			res = append(res, backups_config.BackupNotificationType(v))
		default:
			// ignore unknown values
		}
	}
	return res
}

// isDefaultBackupConfig detects the auto-initialized default config
func isDefaultBackupConfig(cfg *backups_config.BackupConfig) bool {
	if cfg == nil {
		return false
	}
	if cfg.IsBackupsEnabled {
		return false
	}
	// Default store period is WEEK
	if cfg.StorePeriod != period.PeriodWeek {
		return false
	}
	// Default interval is DAILY at 04:00
	if cfg.BackupInterval == nil || cfg.BackupInterval.Interval != intervals.IntervalDaily {
		return false
	}
	if cfg.BackupInterval.TimeOfDay == nil || *cfg.BackupInterval.TimeOfDay != "04:00" {
		return false
	}
	// No storage by default
	if cfg.StorageID != nil {
		return false
	}
	// Default CPU count is 1 and retry is enabled with 3 tries
	if cfg.CpuCount != 1 || !cfg.IsRetryIfFailed || cfg.MaxFailedTriesCount != 3 {
		return false
	}
	return true
}
