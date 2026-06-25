package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	"databasus-backend/internal/util/tools"
)

type fakeSender struct {
	calls []*CollectRequest
	err   error
}

func (f *fakeSender) Send(_ context.Context, req *CollectRequest) error {
	f.calls = append(f.calls, req)
	return f.err
}

type fakeDatabaseLister struct {
	databases []*databases.Database
	err       error
}

func (f *fakeDatabaseLister) GetAllDatabases() ([]*databases.Database, error) {
	return f.databases, f.err
}

type fakeStorageLister struct {
	storages []*storages.Storage
	err      error
}

func (f *fakeStorageLister) GetAllStorages() ([]*storages.Storage, error) {
	return f.storages, f.err
}

type fakeNotifierLister struct {
	notifiers []*notifiers.Notifier
	err       error
}

func (f *fakeNotifierLister) GetAllNotifiers() ([]*notifiers.Notifier, error) {
	return f.notifiers, f.err
}

type fakeBackupChecker struct {
	hasBackupSince map[uuid.UUID]bool
	latestBackups  map[uuid.UUID]*backups_core_logical.LogicalBackup
	err            error
	latestErr      error
}

func (f *fakeBackupChecker) HasSuccessfulBackupSince(
	databaseID uuid.UUID,
	_ time.Time,
) (bool, error) {
	if f.err != nil {
		return false, f.err
	}

	return f.hasBackupSince[databaseID], nil
}

func (f *fakeBackupChecker) GetLatestCompletedBackup(
	databaseID uuid.UUID,
) (*backups_core_logical.LogicalBackup, error) {
	if f.latestErr != nil {
		return nil, f.latestErr
	}

	return f.latestBackups[databaseID], nil
}

type fakePhysicalFullBackupSizer struct {
	fullBackups map[uuid.UUID]*physical_models.PhysicalFullBackup
	err         error
}

func (f *fakePhysicalFullBackupSizer) GetLatestCompletedFullBackup(
	databaseID uuid.UUID,
) (*physical_models.PhysicalFullBackup, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.fullBackups[databaseID], nil
}

type fakeUserCounter struct {
	count int64
	err   error
}

func (f *fakeUserCounter) GetUsersCount() (int64, error) {
	if f.err != nil {
		return 0, f.err
	}

	return f.count, nil
}

type fakeVerificationAgentLister struct {
	agents []*verification_agents.Agent
	err    error
}

func (f *fakeVerificationAgentLister) ListAgents() ([]*verification_agents.Agent, error) {
	return f.agents, f.err
}

type fakeVerificationConfigLister struct {
	enabled []*verification_config.BackupVerificationConfig
	err     error
}

func (f *fakeVerificationConfigLister) ListEnabled() (
	[]*verification_config.BackupVerificationConfig,
	error,
) {
	return f.enabled, f.err
}

func newServiceUnderTest(
	t *testing.T,
	databaseLister databaseLister,
	storageLister storageLister,
	notifierLister notifierLister,
	backupChecker backupChecker,
	verificationAgentLister verificationAgentLister,
	verificationConfigLister verificationConfigLister,
	sender TelemetrySender,
) *TelemetryService {
	t.Helper()

	return newServiceUnderTestWith(
		t,
		databaseLister,
		storageLister,
		notifierLister,
		backupChecker,
		&fakePhysicalFullBackupSizer{},
		verificationAgentLister,
		verificationConfigLister,
		&fakeUserCounter{},
		sender,
	)
}

func newServiceUnderTestWith(
	t *testing.T,
	databaseLister databaseLister,
	storageLister storageLister,
	notifierLister notifierLister,
	backupChecker backupChecker,
	physicalBackupSizer physicalFullBackupSizer,
	verificationAgentLister verificationAgentLister,
	verificationConfigLister verificationConfigLister,
	userCounter userCounter,
	sender TelemetrySender,
) *TelemetryService {
	t.Helper()
	loader := NewInstanceFileLoader(
		filepath.Join(t.TempDir(), "instance.json"),
		slog.New(slog.DiscardHandler),
	)

	return NewTelemetryService(
		loader,
		sender,
		databaseLister,
		storageLister,
		notifierLister,
		backupChecker,
		physicalBackupSizer,
		verificationAgentLister,
		verificationConfigLister,
		userCounter,
		"9.9.9",
		slog.New(slog.DiscardHandler),
	)
}

func availableStatus() *databases.HealthStatus {
	s := databases.HealthStatusAvailable
	return &s
}

func unavailableStatus() *databases.HealthStatus {
	s := databases.HealthStatusUnavailable
	return &s
}

func postgresDatabase(name string, status *databases.HealthStatus) *databases.Database {
	return &databases.Database{
		ID:           uuid.New(),
		Name:         name,
		Type:         databases.DatabaseTypePostgresLogical,
		HealthStatus: status,
		PostgresqlLogical: &postgresql_logical.PostgresqlLogicalDatabase{
			Version: tools.PostgresqlVersion("16"),
		},
	}
}

func physicalDatabase(
	name string,
	backupType postgresql_physical.BackupType,
	status *databases.HealthStatus,
) *databases.Database {
	return &databases.Database{
		ID:           uuid.New(),
		Name:         name,
		Type:         databases.DatabaseTypePostgresPhysical,
		HealthStatus: status,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			Version:    tools.PostgresqlVersion("17"),
			BackupType: backupType,
		},
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func Test_BuildAndSend_ProducesExpectedRequest(t *testing.T) {
	pgDB := postgresDatabase("pg", availableStatus())
	mysqlDB := &databases.Database{
		ID:           uuid.New(),
		Name:         "my",
		Type:         databases.DatabaseTypeMysql,
		HealthStatus: availableStatus(),
		Mysql:        &mysql.MysqlDatabase{Version: tools.MysqlVersion("8.0")},
	}
	mariaDB := &databases.Database{
		ID:           uuid.New(),
		Name:         "maria",
		Type:         databases.DatabaseTypeMariadb,
		HealthStatus: availableStatus(),
		Mariadb:      &mariadb.MariadbDatabase{Version: tools.MariadbVersion("10.6")},
	}
	mongoDB := &databases.Database{
		ID:           uuid.New(),
		Name:         "mongo",
		Type:         databases.DatabaseTypeMongodb,
		HealthStatus: availableStatus(),
		Mongodb:      &mongodb.MongodbDatabase{Version: tools.MongodbVersion("6.0")},
	}

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{pgDB, mysqlDB, mariaDB, mongoDB}},
		&fakeStorageLister{storages: []*storages.Storage{
			{Type: storages.StorageTypeS3},
			{Type: storages.StorageTypeLocal},
		}},
		&fakeNotifierLister{notifiers: []*notifiers.Notifier{
			{NotifierType: notifiers.NotifierTypeEmail},
			{NotifierType: notifiers.NotifierTypeTelegram},
		}},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)

	req := sender.calls[0]
	assert.Equal(t, "9.9.9", req.AppVersion)
	assert.Equal(t, runtime.GOOS, req.OS)
	assert.Equal(t, runtime.GOARCH, req.Arch)
	require.Len(t, req.Databases, 4)

	types := make([]string, 0, len(req.Databases))
	for _, d := range req.Databases {
		types = append(types, d.Type)
	}
	assert.ElementsMatch(t,
		[]string{"POSTGRES_LOGICAL", "MYSQL", "MARIADB", "MONGODB"},
		types,
	)

	assert.Equal(t, []string{"LOCAL", "S3"}, req.Storages)
	assert.Equal(t, []string{"EMAIL", "TELEGRAM"}, req.Notifiers)
	assert.Equal(t, time.Now().UTC().Format("2006-01-02"), req.InstalledAt)
	_, err := uuid.Parse(req.InstanceID)
	require.NoError(t, err)
}

func Test_BuildAndSend_PreservesStorageAndNotifierDuplicates(t *testing.T) {
	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{},
		&fakeStorageLister{storages: []*storages.Storage{
			{Type: storages.StorageTypeS3},
			{Type: storages.StorageTypeS3},
			{Type: storages.StorageTypeS3},
			{Type: storages.StorageTypeLocal},
		}},
		&fakeNotifierLister{notifiers: []*notifiers.Notifier{
			{NotifierType: notifiers.NotifierTypeEmail},
			{NotifierType: notifiers.NotifierTypeEmail},
			{NotifierType: notifiers.NotifierTypeTelegram},
		}},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)

	assert.Equal(t, []string{"LOCAL", "S3", "S3", "S3"}, sender.calls[0].Storages)
	assert.Equal(t, []string{"EMAIL", "EMAIL", "TELEGRAM"}, sender.calls[0].Notifiers)
}

func Test_BuildAndSend_WhenInstanceFileFails_DoesNotCallSender(t *testing.T) {
	// Construct a loader pointing at an unwritable path so LoadOrCreate returns false.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, writeFileForTest(blocker))

	sender := &fakeSender{}
	loader := NewInstanceFileLoader(
		filepath.Join(blocker, "nested", "instance.json"),
		slog.New(slog.DiscardHandler),
	)

	service := NewTelemetryService(
		loader,
		sender,
		&fakeDatabaseLister{},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakePhysicalFullBackupSizer{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		&fakeUserCounter{},
		"9.9.9",
		slog.New(slog.DiscardHandler),
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenSenderFails_PropagatesError(t *testing.T) {
	sendErr := errors.New("network down")
	sender := &fakeSender{err: sendErr}

	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, sendErr)
}

func Test_BuildAndSend_WhenDbHealthStatusAvailable_DbIncluded(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	assert.Len(t, sender.calls[0].Databases, 1)
}

func Test_BuildAndSend_WhenDbHealthStatusUnavailable_DbExcluded(t *testing.T) {
	db := postgresDatabase("pg", unavailableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	assert.Empty(t, sender.calls[0].Databases)
}

func Test_BuildAndSend_WhenHealthcheckOffAndRecentBackup_DbIncluded(t *testing.T) {
	db := postgresDatabase("pg", nil)

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{hasBackupSince: map[uuid.UUID]bool{db.ID: true}},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)
	assert.Equal(t, "POSTGRES_LOGICAL", sender.calls[0].Databases[0].Type)
}

func Test_BuildAndSend_WhenHealthcheckOffAndNoRecentBackup_DbExcluded(t *testing.T) {
	db := postgresDatabase("pg", nil)

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{hasBackupSince: map[uuid.UUID]bool{}},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	assert.Empty(t, sender.calls[0].Databases)
}

func Test_BuildAndSend_WhenBackupCheckerFails_ReturnsError(t *testing.T) {
	db := postgresDatabase("pg", nil)
	checkerErr := errors.New("db down")

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{err: checkerErr},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, checkerErr)
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenLatestBackupHasBothSizes_IncludesBoth(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{
			latestBackups: map[uuid.UUID]*backups_core_logical.LogicalBackup{
				db.ID: {BackupSizeMb: 870.4, BackupRawDbSizeMb: 4321.7},
			},
		},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	entry := sender.calls[0].Databases[0]
	assert.Equal(t, int64(871), entry.BackupSizeMb)
	assert.Equal(t, int64(4322), entry.RawSizeMb)
}

func Test_BuildAndSend_WhenSizesAreSubMb_RoundsUpToOne(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{
			latestBackups: map[uuid.UUID]*backups_core_logical.LogicalBackup{
				db.ID: {BackupSizeMb: 0.3, BackupRawDbSizeMb: 0.1},
			},
		},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	entry := sender.calls[0].Databases[0]
	assert.Equal(t, int64(1), entry.BackupSizeMb)
	assert.Equal(t, int64(1), entry.RawSizeMb)

	encoded, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), "backupSizeMb")
	assert.Contains(t, string(encoded), "rawSizeMb")
}

func Test_BuildAndSend_WhenRawSizeZero_IncludesOnlyBackupSize(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{
			latestBackups: map[uuid.UUID]*backups_core_logical.LogicalBackup{
				db.ID: {BackupSizeMb: 100, BackupRawDbSizeMb: 0},
			},
		},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	entry := sender.calls[0].Databases[0]
	assert.Equal(t, int64(100), entry.BackupSizeMb)
	assert.Equal(t, int64(0), entry.RawSizeMb)

	encoded, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "rawSizeMb")
	assert.Contains(t, string(encoded), "backupSizeMb")
}

func Test_BuildAndSend_WhenBackupSizeZero_IncludesOnlyRawSize(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{
			latestBackups: map[uuid.UUID]*backups_core_logical.LogicalBackup{
				db.ID: {BackupSizeMb: 0, BackupRawDbSizeMb: 999},
			},
		},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	entry := sender.calls[0].Databases[0]
	assert.Equal(t, int64(0), entry.BackupSizeMb)
	assert.Equal(t, int64(999), entry.RawSizeMb)
}

func Test_BuildAndSend_WhenNoCompletedBackup_OmitsBothSizes(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	entry := sender.calls[0].Databases[0]
	assert.Equal(t, int64(0), entry.BackupSizeMb)
	assert.Equal(t, int64(0), entry.RawSizeMb)

	encoded, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "rawSizeMb")
	assert.NotContains(t, string(encoded), "backupSizeMb")
}

func Test_BuildAndSend_WhenLatestBackupLookupFails_ReturnsError(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())
	lookupErr := errors.New("query exploded")

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{latestErr: lookupErr},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, lookupErr)
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenAgentsRegistered_IncludesCapacityRows(t *testing.T) {
	registeredAgents := []*verification_agents.Agent{
		{
			ID:                uuid.New(),
			Name:              "agent-1",
			MaxCPU:            4,
			MaxRAMGb:          16,
			MaxDiskGb:         100,
			MaxConcurrentJobs: 2,
		},
		{
			ID:                uuid.New(),
			Name:              "agent-2",
			MaxCPU:            8,
			MaxRAMGb:          32,
			MaxDiskGb:         200,
			MaxConcurrentJobs: 4,
		},
	}

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{agents: registeredAgents},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)

	assert.Equal(t, []VerificationAgentEntry{
		{MaxCPU: 4, MaxRAMGb: 16, MaxDiskGb: 100, MaxConcurrentJobs: 2},
		{MaxCPU: 8, MaxRAMGb: 32, MaxDiskGb: 200, MaxConcurrentJobs: 4},
	}, sender.calls[0].VerificationAgents)
}

func Test_BuildAndSend_WhenNoAgents_VerificationAgentsIsEmpty(t *testing.T) {
	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)

	require.NotNil(t, sender.calls[0].VerificationAgents)
	assert.Empty(t, sender.calls[0].VerificationAgents)

	encoded, err := json.Marshal(sender.calls[0])
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"verificationAgents":[]`)
}

func Test_BuildAndSend_WhenDbHasAfterBackupConfig_VerificationBlockOmitsIntervalType(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{enabled: []*verification_config.BackupVerificationConfig{
			{
				DatabaseID:                     db.ID,
				IsScheduledVerificationEnabled: true,
				ScheduleType:                   verification_config.VerificationScheduleAfterBackup,
			},
		}},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	entry := sender.calls[0].Databases[0]
	require.NotNil(t, entry.Verification)
	assert.True(t, entry.Verification.IsEnabled)
	assert.Equal(t, "AFTER_BACKUP", entry.Verification.ScheduleType)
	assert.Empty(t, entry.Verification.IntervalType)

	encoded, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"verification"`)
	assert.NotContains(t, string(encoded), "intervalType")
}

func Test_BuildAndSend_WhenDbHasIntervalDailyConfig_IncludesIntervalType(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{enabled: []*verification_config.BackupVerificationConfig{
			{
				DatabaseID:                     db.ID,
				IsScheduledVerificationEnabled: true,
				ScheduleType:                   verification_config.VerificationScheduleInterval,
				VerificationInterval:           intervals.Interval{Type: intervals.IntervalDaily},
			},
		}},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	entry := sender.calls[0].Databases[0]
	require.NotNil(t, entry.Verification)
	assert.True(t, entry.Verification.IsEnabled)
	assert.Equal(t, "INTERVAL", entry.Verification.ScheduleType)
	assert.Equal(t, "DAILY", entry.Verification.IntervalType)

	encoded, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"intervalType":"DAILY"`)
}

func Test_BuildAndSend_WhenDbHasNoEnabledConfig_VerificationBlockAbsent(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	entry := sender.calls[0].Databases[0]
	assert.Nil(t, entry.Verification)

	encoded, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "verification")
}

func Test_BuildAndSend_WhenVerificationAgentListFails_ReturnsError(t *testing.T) {
	listErr := errors.New("agents query exploded")
	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{err: listErr},
		&fakeVerificationConfigLister{},
		sender,
	)

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, listErr)
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenVerificationConfigListFails_ReturnsError(t *testing.T) {
	listErr := errors.New("configs query exploded")
	sender := &fakeSender{}
	service := newServiceUnderTest(
		t,
		&fakeDatabaseLister{},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{err: listErr},
		sender,
	)

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, listErr)
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenPhysicalDatabase_EmitsTypeBackupTypeAndFullBackupSizes(t *testing.T) {
	db := physicalDatabase(
		"pg-physical",
		postgresql_physical.BackupTypeFullIncrementalAndWalStream,
		availableStatus(),
	)

	sender := &fakeSender{}
	service := newServiceUnderTestWith(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakePhysicalFullBackupSizer{
			fullBackups: map[uuid.UUID]*physical_models.PhysicalFullBackup{
				db.ID: {BackupSizeMb: floatPtr(38400.2), RawSizeMb: floatPtr(192000.7)},
			},
		},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		&fakeUserCounter{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	entry := sender.calls[0].Databases[0]
	assert.Equal(t, "POSTGRES_PHYSICAL", entry.Type)
	assert.Equal(t, "17", entry.Version)
	assert.Equal(t, "FULL_INCREMENTAL_WAL_STREAM", entry.BackupType)
	assert.Equal(t, int64(38401), entry.BackupSizeMb)
	assert.Equal(t, int64(192001), entry.RawSizeMb)

	encoded, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"backupType":"FULL_INCREMENTAL_WAL_STREAM"`)
}

func Test_BuildAndSend_WhenPhysicalDatabaseHasNoFullBackup_OmitsSizes(t *testing.T) {
	db := physicalDatabase("pg-physical", postgresql_physical.BackupTypeFullOnly, availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTestWith(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakePhysicalFullBackupSizer{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		&fakeUserCounter{},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	entry := sender.calls[0].Databases[0]
	assert.Equal(t, "FULL", entry.BackupType)
	assert.Equal(t, int64(0), entry.BackupSizeMb)
	assert.Equal(t, int64(0), entry.RawSizeMb)

	encoded, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "rawSizeMb")
	assert.NotContains(t, string(encoded), "backupSizeMb")
}

func Test_BuildAndSend_WhenPhysicalFullBackupLookupFails_ReturnsError(t *testing.T) {
	db := physicalDatabase("pg-physical", postgresql_physical.BackupTypeFullOnly, availableStatus())
	lookupErr := errors.New("physical query exploded")

	sender := &fakeSender{}
	service := newServiceUnderTestWith(
		t,
		&fakeDatabaseLister{databases: []*databases.Database{db}},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakePhysicalFullBackupSizer{err: lookupErr},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		&fakeUserCounter{},
		sender,
	)

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, lookupErr)
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenUsersCounted_PopulatesUserCount(t *testing.T) {
	sender := &fakeSender{}
	service := newServiceUnderTestWith(
		t,
		&fakeDatabaseLister{},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakePhysicalFullBackupSizer{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		&fakeUserCounter{count: 7},
		sender,
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	assert.Equal(t, 7, sender.calls[0].UserCount)
}

func Test_BuildAndSend_WhenUserCounterFails_ReturnsError(t *testing.T) {
	countErr := errors.New("users count exploded")

	sender := &fakeSender{}
	service := newServiceUnderTestWith(
		t,
		&fakeDatabaseLister{},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakePhysicalFullBackupSizer{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		&fakeUserCounter{err: countErr},
		sender,
	)

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, countErr)
	assert.Empty(t, sender.calls)
}

func writeFileForTest(path string) error {
	return os.WriteFile(path, []byte("x"), 0o600)
}
