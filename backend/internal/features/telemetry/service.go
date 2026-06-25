package telemetry

import (
	"context"
	"log/slog"
	"math"
	"runtime"
	"sort"
	"time"

	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
)

const (
	// activeBackupWindow is how far back a successful backup must have happened
	// for a database with disabled healthcheck to count as "active".
	activeBackupWindow = 7 * 24 * time.Hour

	// maxArrayEntries matches the server-side cap from TELEMETRY.md.
	maxArrayEntries = 200
)

type databaseLister interface {
	GetAllDatabases() ([]*databases.Database, error)
}

type storageLister interface {
	GetAllStorages() ([]*storages.Storage, error)
}

type notifierLister interface {
	GetAllNotifiers() ([]*notifiers.Notifier, error)
}

type backupChecker interface {
	HasSuccessfulBackupSince(databaseID uuid.UUID, since time.Time) (bool, error)
	GetLatestCompletedBackup(databaseID uuid.UUID) (*backups_core_logical.LogicalBackup, error)
}

type physicalFullBackupSizer interface {
	GetLatestCompletedFullBackup(databaseID uuid.UUID) (*physical_models.PhysicalFullBackup, error)
}

type userCounter interface {
	GetUsersCount() (int64, error)
}

type verificationAgentLister interface {
	ListAgents() ([]*verification_agents.Agent, error)
}

type verificationConfigLister interface {
	ListEnabled() ([]*verification_config.BackupVerificationConfig, error)
}

type TelemetryService struct {
	instanceLoader            *InstanceFileLoader
	sender                    TelemetrySender
	databaseService           databaseLister
	storageService            storageLister
	notifierService           notifierLister
	backupService             backupChecker
	physicalBackupService     physicalFullBackupSizer
	verificationAgentService  verificationAgentLister
	verificationConfigService verificationConfigLister
	userService               userCounter
	appVersion                string
	logger                    *slog.Logger
}

func NewTelemetryService(
	instanceLoader *InstanceFileLoader,
	sender TelemetrySender,
	databaseService databaseLister,
	storageService storageLister,
	notifierService notifierLister,
	backupService backupChecker,
	physicalBackupService physicalFullBackupSizer,
	verificationAgentService verificationAgentLister,
	verificationConfigService verificationConfigLister,
	userService userCounter,
	appVersion string,
	logger *slog.Logger,
) *TelemetryService {
	return &TelemetryService{
		instanceLoader:            instanceLoader,
		sender:                    sender,
		databaseService:           databaseService,
		storageService:            storageService,
		notifierService:           notifierService,
		backupService:             backupService,
		physicalBackupService:     physicalBackupService,
		verificationAgentService:  verificationAgentService,
		verificationConfigService: verificationConfigService,
		userService:               userService,
		appVersion:                appVersion,
		logger:                    logger,
	}
}

func (s *TelemetryService) BuildAndSend(ctx context.Context) error {
	instance, ok := s.instanceLoader.LoadOrCreate()
	if !ok {
		return nil
	}

	enabledConfigsByDatabaseID, err := s.loadEnabledVerificationConfigs()
	if err != nil {
		return err
	}

	databaseEntries, err := s.collectActiveDatabases(enabledConfigsByDatabaseID)
	if err != nil {
		return err
	}

	storageTypes, err := s.collectStorageTypes()
	if err != nil {
		return err
	}

	notifierTypes, err := s.collectNotifierTypes()
	if err != nil {
		return err
	}

	verificationAgents, err := s.collectVerificationAgents()
	if err != nil {
		return err
	}

	userCount, err := s.userService.GetUsersCount()
	if err != nil {
		return err
	}

	req := &CollectRequest{
		InstanceID:         instance.InstanceID,
		AppVersion:         s.appVersion,
		OS:                 runtime.GOOS,
		Arch:               runtime.GOARCH,
		InstalledAt:        instance.InstalledAt,
		UserCount:          int(userCount),
		Databases:          capDatabases(databaseEntries),
		Storages:           capStrings(storageTypes),
		Notifiers:          capStrings(notifierTypes),
		VerificationAgents: capAgents(verificationAgents),
	}

	return s.sender.Send(ctx, req)
}

func (s *TelemetryService) loadEnabledVerificationConfigs() (
	map[uuid.UUID]*verification_config.BackupVerificationConfig,
	error,
) {
	enabledConfigs, err := s.verificationConfigService.ListEnabled()
	if err != nil {
		return nil, err
	}

	indexed := make(map[uuid.UUID]*verification_config.BackupVerificationConfig, len(enabledConfigs))
	for _, config := range enabledConfigs {
		indexed[config.DatabaseID] = config
	}

	return indexed, nil
}

func (s *TelemetryService) collectActiveDatabases(
	enabledConfigsByDatabaseID map[uuid.UUID]*verification_config.BackupVerificationConfig,
) ([]DatabaseEntry, error) {
	allDatabases, err := s.databaseService.GetAllDatabases()
	if err != nil {
		return nil, err
	}

	since := time.Now().UTC().Add(-activeBackupWindow)
	entries := make([]DatabaseEntry, 0, len(allDatabases))

	for _, db := range allDatabases {
		isActive, err := s.isDatabaseActive(db, since)
		if err != nil {
			return nil, err
		}

		if !isActive {
			continue
		}

		entry, ok := buildDatabaseEntry(db)
		if !ok {
			continue
		}

		if err := s.attachBackupSizes(&entry, db); err != nil {
			return nil, err
		}

		if config, hasConfig := enabledConfigsByDatabaseID[db.ID]; hasConfig {
			entry.Verification = buildVerificationEntry(config)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func buildVerificationEntry(
	config *verification_config.BackupVerificationConfig,
) *DatabaseVerificationEntry {
	entry := &DatabaseVerificationEntry{
		IsEnabled:    true,
		ScheduleType: string(config.ScheduleType),
	}

	if config.ScheduleType == verification_config.VerificationScheduleInterval {
		entry.IntervalType = string(config.VerificationInterval.Type)
	}

	return entry
}

func (s *TelemetryService) collectVerificationAgents() ([]VerificationAgentEntry, error) {
	listedAgents, err := s.verificationAgentService.ListAgents()
	if err != nil {
		return nil, err
	}

	entries := make([]VerificationAgentEntry, 0, len(listedAgents))
	for _, agent := range listedAgents {
		entries = append(entries, VerificationAgentEntry{
			MaxCPU:            agent.MaxCPU,
			MaxRAMGb:          agent.MaxRAMGb,
			MaxDiskGb:         agent.MaxDiskGb,
			MaxConcurrentJobs: agent.MaxConcurrentJobs,
		})
	}

	return entries, nil
}

// attachBackupSizes records raw/compressed size from the size-of-record backup.
// Physical databases have no logical backup rows, so their size comes from the
// latest completed physical FULL backup; every other type uses the latest
// completed logical backup.
func (s *TelemetryService) attachBackupSizes(
	entry *DatabaseEntry,
	db *databases.Database,
) error {
	if db.Type == databases.DatabaseTypePostgresPhysical {
		return s.attachPhysicalBackupSizes(entry, db.ID)
	}

	backup, err := s.backupService.GetLatestCompletedBackup(db.ID)
	if err != nil {
		return err
	}

	if backup == nil {
		return nil
	}

	if backup.BackupSizeMb > 0 {
		entry.BackupSizeMb = int64(math.Ceil(backup.BackupSizeMb))
	}

	if backup.BackupRawDbSizeMb > 0 {
		entry.RawSizeMb = int64(math.Ceil(backup.BackupRawDbSizeMb))
	}

	return nil
}

func (s *TelemetryService) attachPhysicalBackupSizes(
	entry *DatabaseEntry,
	databaseID uuid.UUID,
) error {
	fullBackup, err := s.physicalBackupService.GetLatestCompletedFullBackup(databaseID)
	if err != nil {
		return err
	}

	if fullBackup == nil {
		return nil
	}

	if fullBackup.BackupSizeMb != nil && *fullBackup.BackupSizeMb > 0 {
		entry.BackupSizeMb = int64(math.Ceil(*fullBackup.BackupSizeMb))
	}

	if fullBackup.RawSizeMb != nil && *fullBackup.RawSizeMb > 0 {
		entry.RawSizeMb = int64(math.Ceil(*fullBackup.RawSizeMb))
	}

	return nil
}

// isDatabaseActive returns true when a database should be counted in telemetry.
//
//   - HealthStatus == AVAILABLE   → active.
//   - HealthStatus == UNAVAILABLE → not active (healthcheck is on and the DB is down).
//   - HealthStatus == nil         → healthcheck is disabled; active only if a
//     successful backup happened inside `since`.
func (s *TelemetryService) isDatabaseActive(
	db *databases.Database,
	since time.Time,
) (bool, error) {
	if db.HealthStatus != nil {
		return *db.HealthStatus == databases.HealthStatusAvailable, nil
	}

	return s.backupService.HasSuccessfulBackupSince(db.ID, since)
}

func buildDatabaseEntry(db *databases.Database) (DatabaseEntry, bool) {
	switch db.Type {
	case databases.DatabaseTypePostgresLogical:
		// The legacy POSTGRES type is the same engine now labelled POSTGRES_LOGICAL;
		// analytics counts the two as one.
		if db.PostgresqlLogical == nil {
			return DatabaseEntry{}, false
		}
		return DatabaseEntry{Type: string(db.Type), Version: string(db.PostgresqlLogical.Version)}, true
	case databases.DatabaseTypePostgresPhysical:
		if db.PostgresqlPhysical == nil {
			return DatabaseEntry{}, false
		}
		return DatabaseEntry{
			Type:       string(db.Type),
			Version:    string(db.PostgresqlPhysical.Version),
			BackupType: string(db.PostgresqlPhysical.BackupType),
		}, true
	case databases.DatabaseTypeMysql:
		if db.Mysql == nil {
			return DatabaseEntry{}, false
		}
		return DatabaseEntry{Type: string(db.Type), Version: string(db.Mysql.Version)}, true
	case databases.DatabaseTypeMariadb:
		if db.Mariadb == nil {
			return DatabaseEntry{}, false
		}
		return DatabaseEntry{Type: string(db.Type), Version: string(db.Mariadb.Version)}, true
	case databases.DatabaseTypeMongodb:
		if db.Mongodb == nil {
			return DatabaseEntry{}, false
		}
		return DatabaseEntry{Type: string(db.Type), Version: string(db.Mongodb.Version)}, true
	}

	return DatabaseEntry{}, false
}

func (s *TelemetryService) collectStorageTypes() ([]string, error) {
	allStorages, err := s.storageService.GetAllStorages()
	if err != nil {
		return nil, err
	}

	types := make([]string, 0, len(allStorages))
	for _, st := range allStorages {
		key := string(st.Type)
		if key == "" {
			continue
		}

		types = append(types, key)
	}

	sort.Strings(types)
	return types, nil
}

func (s *TelemetryService) collectNotifierTypes() ([]string, error) {
	allNotifiers, err := s.notifierService.GetAllNotifiers()
	if err != nil {
		return nil, err
	}

	types := make([]string, 0, len(allNotifiers))
	for _, n := range allNotifiers {
		key := string(n.NotifierType)
		if key == "" {
			continue
		}

		types = append(types, key)
	}

	sort.Strings(types)
	return types, nil
}

func capStrings(in []string) []string {
	if len(in) > maxArrayEntries {
		return in[:maxArrayEntries]
	}

	return in
}

func capDatabases(in []DatabaseEntry) []DatabaseEntry {
	if len(in) > maxArrayEntries {
		return in[:maxArrayEntries]
	}

	return in
}

func capAgents(in []VerificationAgentEntry) []VerificationAgentEntry {
	if len(in) > maxArrayEntries {
		return in[:maxArrayEntries]
	}

	return in
}
