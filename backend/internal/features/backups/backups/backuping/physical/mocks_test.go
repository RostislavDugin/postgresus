package backuping_physical

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
)

// fakeFullExecutor stands in for the real pg_basebackup-driven CreateFullBackupUsecase so
// orchestrator tests can pin the executor's outcome without a live cluster or
// object storage. The fake never touches storage, so these tests stay hermetic.
type fakeFullExecutor struct {
	result    postgresql_executor.PhysicalBackupResult
	err       error
	callCount int
}

func (e *fakeFullExecutor) Execute(
	_ context.Context,
	_ postgresql_executor.FullBackupSpec,
) (postgresql_executor.PhysicalBackupResult, error) {
	e.callCount++

	return e.result, e.err
}

type fakeIncrementalExecutor struct {
	result    postgresql_executor.PhysicalBackupResult
	err       error
	callCount int
}

func (e *fakeIncrementalExecutor) Execute(
	_ context.Context,
	_ postgresql_executor.IncrementalBackupSpec,
) (postgresql_executor.PhysicalBackupResult, error) {
	e.callCount++

	return e.result, e.err
}

type sentNotification struct {
	Notifier *notifiers.Notifier
	Title    string
	Message  string
}

// recordingNotificationSender captures every dispatched notification so tests
// can assert on count (fan-out), routing (which notifier), and content.
type recordingNotificationSender struct {
	sentNotifications []sentNotification
}

func (s *recordingNotificationSender) SendNotification(
	notifier *notifiers.Notifier,
	title string,
	message string,
) {
	s.sentNotifications = append(s.sentNotifications, sentNotification{
		Notifier: notifier,
		Title:    title,
		Message:  message,
	})
}

// backupPrereqs is everything loadBackupContext needs to resolve: a workspace,
// a storage, a notifier, a POSTGRES_PHYSICAL database, and a saved backup
// config pointing at that storage. Tests seed their own backup rows against
// DB.ID / Storage.ID.
type backupPrereqs struct {
	Workspace *workspaces_models.Workspace
	Storage   *storages.Storage
	Notifier  *notifiers.Notifier
	DB        *databases.Database
	Config    *backups_config_physical.PhysicalBackupConfig
}

// seedBackupPrereqs builds the loadBackupContext prerequisites hermetically.
// It mirrors postgresql.SetupPhysicalDBForBackupVersion but deliberately skips
// PopulateDbData (no live PG) and the backup row / in-flight claim (tests own
// those). Cleanup deletes the physical catalog before the database is removed.
func seedBackupPrereqs(t *testing.T) *backupPrereqs {
	t.Helper()

	router := backuping_logical.CreateTestRouter()

	owner := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	workspace := workspaces_testing.CreateTestWorkspace("ws "+uuid.New().String(), owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(workspace, router) })

	testStorage := storages.CreateTestStorage(workspace.ID)
	t.Cleanup(func() { storages.RemoveTestStorage(testStorage.ID) })

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	t.Cleanup(func() { notifiers.RemoveTestNotifier(notifier) })

	db := databases.CreateTestPhysicalPostgresDatabase(workspace.ID, notifier, "17")
	t.Cleanup(func() { databases.RemoveTestDatabase(db) })

	cfgService := backups_config_physical.GetBackupConfigService()

	cfg, err := cfgService.GetBackupConfigByDbId(db.ID)
	require.NoError(t, err)

	timeOfDay := "04:00"

	cfg.IsBackupsEnabled = true
	cfg.StorageID = &testStorage.ID
	cfg.Storage = testStorage
	cfg.Encryption = backups_core_enums.BackupEncryptionNone
	cfg.PostgresqlPhysical = db.PostgresqlPhysical
	cfg.FullBackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}

	savedConfig, err := cfgService.SaveBackupConfig(cfg)
	require.NoError(t, err)

	t.Cleanup(func() { physical_testing.DeleteAllPhysicalCatalogForDatabase(t, db.ID) })

	return &backupPrereqs{
		Workspace: workspace,
		Storage:   testStorage,
		Notifier:  notifier,
		DB:        db,
		Config:    savedConfig,
	}
}
