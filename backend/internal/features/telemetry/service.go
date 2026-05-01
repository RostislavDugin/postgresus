package telemetry

import (
	"context"
	"log/slog"
	"runtime"
	"sort"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
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
}

type TelemetryService struct {
	instanceLoader  *InstanceFileLoader
	sender          TelemetrySender
	databaseService databaseLister
	storageService  storageLister
	notifierService notifierLister
	backupService   backupChecker
	appVersion      string
	logger          *slog.Logger
}

func NewTelemetryService(
	instanceLoader *InstanceFileLoader,
	sender TelemetrySender,
	databaseService databaseLister,
	storageService storageLister,
	notifierService notifierLister,
	backupService backupChecker,
	appVersion string,
	logger *slog.Logger,
) *TelemetryService {
	return &TelemetryService{
		instanceLoader:  instanceLoader,
		sender:          sender,
		databaseService: databaseService,
		storageService:  storageService,
		notifierService: notifierService,
		backupService:   backupService,
		appVersion:      appVersion,
		logger:          logger,
	}
}

func (s *TelemetryService) BuildAndSend(ctx context.Context) error {
	instance, ok := s.instanceLoader.LoadOrCreate()
	if !ok {
		return nil
	}

	databaseEntries, err := s.collectActiveDatabases()
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

	req := &CollectRequest{
		InstanceID:  instance.InstanceID,
		AppVersion:  s.appVersion,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		InstalledAt: instance.InstalledAt,
		Databases:   capDatabases(databaseEntries),
		Storages:    capStrings(storageTypes),
		Notifiers:   capStrings(notifierTypes),
	}

	return s.sender.Send(ctx, req)
}

func (s *TelemetryService) collectActiveDatabases() ([]DatabaseEntry, error) {
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

		entries = append(entries, entry)
	}

	return entries, nil
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
	case databases.DatabaseTypePostgres:
		if db.Postgresql == nil {
			return DatabaseEntry{}, false
		}
		return DatabaseEntry{Type: string(db.Type), Version: string(db.Postgresql.Version)}, true
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
	case databases.DatabaseTypeClickhouse:
		if db.Clickhouse == nil {
			return DatabaseEntry{}, false
		}
		return DatabaseEntry{Type: string(db.Type), Version: string(db.Clickhouse.Version)}, true
	}

	return DatabaseEntry{}, false
}

func (s *TelemetryService) collectStorageTypes() ([]string, error) {
	allStorages, err := s.storageService.GetAllStorages()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(allStorages))
	types := make([]string, 0, len(allStorages))

	for _, st := range allStorages {
		key := string(st.Type)
		if key == "" {
			continue
		}

		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
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

	seen := make(map[string]struct{}, len(allNotifiers))
	types := make([]string, 0, len(allNotifiers))

	for _, n := range allNotifiers {
		key := string(n.NotifierType)
		if key == "" {
			continue
		}

		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
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
