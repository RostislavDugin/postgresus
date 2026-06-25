package postgresql_physical

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

type pgFixture struct {
	name    string
	version tools.PostgresqlVersion
	// image boots the throwaway summarize_wal=off / custom-tablespace variants via testcontainers;
	// the standard port() points at the shared compose primary source (the cross-suite fixture).
	image string
	port  func() string
}

func physicalFixtures() []pgFixture {
	return []pgFixture{
		{
			name:    "pg17",
			version: tools.PostgresqlVersion17,
			image:   "postgres:17",
			port:    func() string { return config.GetEnv().TestPhysicalPostgres17Port },
		},
		{
			name:    "pg18",
			version: tools.PostgresqlVersion18,
			image:   "postgres:18",
			port:    func() string { return config.GetEnv().TestPhysicalPostgres18Port },
		},
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func newTestModel(t *testing.T, port string) *PostgresqlPhysicalDatabase {
	t.Helper()

	portInt, err := strconv.Atoi(port)
	require.NoError(t, err)

	return newTestModelAt(t, config.GetEnv().TestLocalhost, portInt)
}

// newTestModelAt builds a test model pointed at an explicit host:port — used for the throwaway
// testcontainers sources, whose host may differ from TestLocalhost under TESTCONTAINERS_HOST_OVERRIDE.
func newTestModelAt(t *testing.T, host string, port int) *PostgresqlPhysicalDatabase {
	t.Helper()

	return &PostgresqlPhysicalDatabase{
		Host:                host,
		Port:                port,
		Username:            "testuser",
		Password:            "testpassword",
		SslMode:             postgresql_shared.PostgresSslModeDisable,
		BackupType:          BackupTypeFullOnly,
		ReplicationSlotName: fmt.Sprintf("databasus_test_%s", uuid.New().String()[:8]),
	}
}

func openTestConn(t *testing.T, port string) *pgx.Conn {
	t.Helper()

	portInt, err := strconv.Atoi(port)
	require.NoError(t, err)

	return openTestConnAt(t, config.GetEnv().TestLocalhost, portInt)
}

func openTestConnAt(t *testing.T, host string, port int) *pgx.Conn {
	t.Helper()

	dsn := fmt.Sprintf(
		"host=%s port=%d user=testuser password=testpassword dbname=postgres sslmode=disable",
		host, port,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = conn.Close(context.Background())
	})

	return conn
}

func createTempUser(t *testing.T, conn *pgx.Conn, extraAttrs string) (string, string) {
	t.Helper()

	username := fmt.Sprintf("databasus_tu_%s", uuid.New().String()[:8])
	password := "Temp1234!Pwd"

	stmt := fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`, username, password)
	if extraAttrs != "" {
		stmt = stmt + " " + extraAttrs
	}

	_, err := conn.Exec(context.Background(), stmt)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = conn.Exec(
			context.Background(),
			fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username),
		)
	})

	return username, password
}

func createMarkerRole(t *testing.T, conn *pgx.Conn, name string) {
	t.Helper()

	_, _ = conn.Exec(context.Background(), fmt.Sprintf(`DROP ROLE IF EXISTS "%s"`, name))

	_, err := conn.Exec(context.Background(), fmt.Sprintf(`CREATE ROLE "%s" NOLOGIN`, name))
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = conn.Exec(
			context.Background(),
			fmt.Sprintf(`DROP ROLE IF EXISTS "%s"`, name),
		)
	})
}

func grantRoleToCurrent(t *testing.T, conn *pgx.Conn, role string) {
	t.Helper()

	_, err := conn.Exec(
		context.Background(),
		fmt.Sprintf(`GRANT "%s" TO current_user`, role),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = conn.Exec(
			context.Background(),
			fmt.Sprintf(`REVOKE "%s" FROM current_user`, role),
		)
	})
}

func roleExistsHelper(t *testing.T, conn *pgx.Conn, name string) bool {
	t.Helper()

	var exists bool
	require.NoError(t, conn.QueryRow(
		context.Background(),
		`SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)`,
		name,
	).Scan(&exists))

	return exists
}

func validModel() *PostgresqlPhysicalDatabase {
	return &PostgresqlPhysicalDatabase{
		Version:    tools.PostgresqlVersion17,
		Host:       "127.0.0.1",
		Port:       5432,
		Username:   "user",
		Password:   "pass",
		SslMode:    postgresql_shared.PostgresSslModeDisable,
		BackupType: BackupTypeFullOnly,
	}
}

func Test_Validate_RejectsBlankHost(t *testing.T) {
	m := validModel()
	m.Host = ""

	assert.Error(t, m.Validate())
}

func Test_Validate_RejectsZeroPort(t *testing.T) {
	m := validModel()
	m.Port = 0

	assert.Error(t, m.Validate())
}

func Test_Validate_RejectsBlankUsername(t *testing.T) {
	m := validModel()
	m.Username = ""

	assert.Error(t, m.Validate())
}

func Test_Validate_RejectsBlankPassword(t *testing.T) {
	m := validModel()
	m.Password = ""

	assert.Error(t, m.Validate())
}

func Test_Validate_DelegatesSslConfigToShared(t *testing.T) {
	m := validModel()
	m.SslMode = "totally-invalid"

	assert.Error(t, m.Validate())
}

func Test_Validate_RejectsUnknownBackupType(t *testing.T) {
	m := validModel()
	m.BackupType = "GIBBERISH"

	assert.Error(t, m.Validate())
}

func Test_Validate_AcceptsAllThreeBackupTypes(t *testing.T) {
	for _, bt := range []BackupType{
		BackupTypeFullOnly,
		BackupTypeFullAndIncremental,
		BackupTypeFullIncrementalAndWalStream,
	} {
		m := validModel()
		m.BackupType = bt
		assert.NoErrorf(t, m.Validate(), "backup type %s", bt)
	}
}

func Test_BackupType_RequiresWalSummary(t *testing.T) {
	cases := map[BackupType]bool{
		BackupTypeFullOnly:                    false,
		BackupTypeFullAndIncremental:          true,
		BackupTypeFullIncrementalAndWalStream: true,
	}

	for backupType, expectsWalSummary := range cases {
		assert.Equal(
			t,
			expectsWalSummary,
			backupType.IsRequireWalSummary(),
			"backup type %s",
			backupType,
		)
	}
}

func Test_ValidateUpdate_RejectsSystemIdentifierChange(t *testing.T) {
	x := "111"
	y := "222"
	old := &PostgresqlPhysicalDatabase{SystemIdentifier: &x}
	next := &PostgresqlPhysicalDatabase{SystemIdentifier: &y}

	err := next.ValidateUpdate(old)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "immutable")
}

func Test_ValidateUpdate_AllowsSystemIdentifierFirstSet(t *testing.T) {
	first := "111"
	old := &PostgresqlPhysicalDatabase{SystemIdentifier: nil}
	next := &PostgresqlPhysicalDatabase{SystemIdentifier: &first}

	assert.NoError(t, next.ValidateUpdate(old))
}

func Test_ValidateUpdate_AllowsUnchangedSystemIdentifier(t *testing.T) {
	same := "111"
	old := &PostgresqlPhysicalDatabase{SystemIdentifier: &same}
	next := &PostgresqlPhysicalDatabase{SystemIdentifier: &same}

	assert.NoError(t, next.ValidateUpdate(old))
}

func Test_SystemIdentifierUint64(t *testing.T) {
	positiveSysID := "7361234567890123456" // fits int64, parses directly
	highBitSysID := "-1"                   // uint64 max stored as a signed bigint
	invalidSysID := "not-a-number"

	scenarios := []struct {
		name     string
		database *PostgresqlPhysicalDatabase
		systemID uint64
	}{
		{"unset yields zero", &PostgresqlPhysicalDatabase{SystemIdentifier: nil}, 0},
		{"positive value", &PostgresqlPhysicalDatabase{SystemIdentifier: &positiveSysID}, 7361234567890123456},
		{
			"high-bit value stored as negative bigint",
			&PostgresqlPhysicalDatabase{SystemIdentifier: &highBitSysID},
			^uint64(0),
		},
		{"unparseable yields zero", &PostgresqlPhysicalDatabase{SystemIdentifier: &invalidSysID}, 0},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			assert.Equal(t, scenario.systemID, scenario.database.SystemIdentifierUint64())
		})
	}
}

func Test_Update_PreservesPasswordWhenIncomingBlank(t *testing.T) {
	existing := validModel()
	existing.Password = "kept"

	incoming := validModel()
	incoming.Password = ""

	existing.Update(incoming)

	assert.Equal(t, "kept", existing.Password)
}

func Test_Update_OverwritesPasswordWhenIncomingNonBlank(t *testing.T) {
	existing := validModel()
	existing.Password = "old"

	incoming := validModel()
	incoming.Password = "new"

	existing.Update(incoming)

	assert.Equal(t, "new", existing.Password)
}

func Test_Update_PreservesSslClientKeyWhenIncomingBlank(t *testing.T) {
	existing := validModel()
	existing.SslMode = postgresql_shared.PostgresSslModeRequire
	existing.SslClientKey = "kept-key"

	incoming := validModel()
	incoming.SslMode = postgresql_shared.PostgresSslModeRequire
	incoming.SslClientKey = ""

	existing.Update(incoming)

	assert.Equal(t, "kept-key", existing.SslClientKey)
}

func Test_Update_NeverOverwritesSystemIdentifierFromIncoming(t *testing.T) {
	original := "111"
	existing := validModel()
	existing.SystemIdentifier = &original

	hostile := "222"
	incoming := validModel()
	incoming.SystemIdentifier = &hostile

	existing.Update(incoming)

	require.NotNil(t, existing.SystemIdentifier)
	assert.Equal(t, "111", *existing.SystemIdentifier)
}

func Test_HideSensitiveData_ZerosPasswordAndSslClientKey(t *testing.T) {
	m := validModel()
	m.Password = "secret"
	m.SslClientKey = "key"
	m.SslClientCert = "visible"

	m.HideSensitiveData()

	assert.Empty(t, m.Password)
	assert.Empty(t, m.SslClientKey)
	assert.Equal(t, "visible", m.SslClientCert)
}

func Test_HideSensitiveData_NilReceiver_DoesNotPanic(t *testing.T) {
	var m *PostgresqlPhysicalDatabase
	assert.NotPanics(t, func() {
		m.HideSensitiveData()
	})
}

type noopEncryptor struct{}

func (noopEncryptor) Encrypt(s string) (string, error) { return "enc:" + s, nil }
func (noopEncryptor) Decrypt(s string) (string, error) {
	if len(s) > 4 && s[:4] == "enc:" {
		return s[4:], nil
	}
	return s, nil
}

func Test_EncryptSensitiveFields_EncryptsAllFourFields(t *testing.T) {
	m := validModel()
	m.Password = "p"
	m.SslClientCert = "c"
	m.SslClientKey = "k"
	m.SslRootCert = "r"

	require.NoError(t, m.EncryptSensitiveFields(noopEncryptor{}))

	assert.Equal(t, "enc:p", m.Password)
	assert.Equal(t, "enc:c", m.SslClientCert)
	assert.Equal(t, "enc:k", m.SslClientKey)
	assert.Equal(t, "enc:r", m.SslRootCert)
}

func Test_EncryptSensitiveFields_SkipsBlankFields(t *testing.T) {
	m := validModel()
	m.Password = "p"
	m.SslClientCert = ""
	m.SslClientKey = ""
	m.SslRootCert = ""

	require.NoError(t, m.EncryptSensitiveFields(noopEncryptor{}))

	assert.Equal(t, "enc:p", m.Password)
	assert.Empty(t, m.SslClientCert)
	assert.Empty(t, m.SslClientKey)
	assert.Empty(t, m.SslRootCert)
}

func Test_PopulateDbData_DetectsVersion(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			m.Version = ""

			require.NoError(t, m.PopulateDbData(testLogger(), nil))
			assert.Equal(t, fx.version, m.Version)
		})
	}
}

func Test_PopulateDbData_CapturesSystemIdentifierOnFirstCall(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			require.Nil(t, m.SystemIdentifier)

			require.NoError(t, m.PopulateDbData(testLogger(), nil))

			require.NotNil(t, m.SystemIdentifier)
			assert.NotEmpty(t, *m.SystemIdentifier)
		})
	}
}

func Test_PopulateDbData_DoesNotOverwriteExistingSystemIdentifier(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			preset := "9999999999999999999"
			m.SystemIdentifier = &preset

			require.NoError(t, m.PopulateDbData(testLogger(), nil))

			require.NotNil(t, m.SystemIdentifier)
			assert.Equal(t, preset, *m.SystemIdentifier)
		})
	}
}

func Test_TestReplicationConnection_SucceedsAgainstReplicationReadyCluster(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			for _, bt := range []BackupType{
				BackupTypeFullOnly,
				BackupTypeFullAndIncremental,
				BackupTypeFullIncrementalAndWalStream,
			} {
				m := newTestModel(t, fx.port())
				m.BackupType = bt

				assert.NoErrorf(t, m.TestReplicationConnection(testLogger(), nil),
					"backup type %s should succeed", bt)
			}
		})
	}
}

func Test_TestReplicationConnection_FailsForFullIncrementalWhenSummarizeWalOff(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			source := containers.StartPhysicalPostgres(t, fx.image, containers.WithoutSummarizer())

			m := newTestModelAt(t, source.Host, source.Port)
			m.BackupType = BackupTypeFullAndIncremental

			err := m.TestReplicationConnection(testLogger(), nil)
			require.Error(t, err)
			assert.Equal(t, postgresql_shared.ConnErrWalSummaryDisabled, connTestErrorCode(t, err))
		})
	}
}

func Test_TestReplicationConnection_FailsForFullIncrementalWalStreamWhenSummarizeWalOff(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			source := containers.StartPhysicalPostgres(t, fx.image, containers.WithoutSummarizer())

			m := newTestModelAt(t, source.Host, source.Port)
			m.BackupType = BackupTypeFullIncrementalAndWalStream

			err := m.TestReplicationConnection(testLogger(), nil)
			require.Error(t, err)
			assert.Equal(t, postgresql_shared.ConnErrWalSummaryDisabled, connTestErrorCode(t, err))
		})
	}
}

func Test_ReadReplicationSettings_OnPreSummarizeWalVersion_DefaultsSummarizeWalToOff(t *testing.T) {
	source := containers.StartPostgres(t, "postgres:16")

	conn := openTestConnAt(t, source.Host, source.Port)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	settings, err := readReplicationSettings(ctx, conn)

	require.NoError(t, err)
	assert.Equal(t, "off", settings.summarizeWal)
}

func Test_TestReplicationConnection_SucceedsForFullOnlyWhenSummarizeWalOff(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			source := containers.StartPhysicalPostgres(t, fx.image, containers.WithoutSummarizer())

			m := newTestModelAt(t, source.Host, source.Port)
			m.BackupType = BackupTypeFullOnly

			assert.NoError(t, m.TestReplicationConnection(testLogger(), nil))
		})
	}
}

func Test_TestReplicationConnection_FailsForWrongPassword(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			m.Password = "wrong-password"

			err := m.TestReplicationConnection(testLogger(), nil)
			assert.Equal(t, postgresql_shared.ConnErrBadCredentials, connTestErrorCode(t, err))
		})
	}
}

func Test_TestReplicationConnection_FailsForNonReplicationUser(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			username, password := createTempUser(t, setupConn, "")

			m := newTestModel(t, fx.port())
			m.Username = username
			m.Password = password

			err := m.TestReplicationConnection(testLogger(), nil)
			assert.Equal(t, postgresql_shared.ConnErrNoReplicationPrivilege, connTestErrorCode(t, err))
		})
	}
}

func Test_TestReplicationConnection_FailsWhenCustomTablespacesPresent(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			source := containers.StartPhysicalPostgres(t, fx.image, containers.WithTablespace())

			m := newTestModelAt(t, source.Host, source.Port)

			err := m.TestReplicationConnection(testLogger(), nil)

			assert.Equal(t, postgresql_shared.ConnErrCustomTablespaces, connTestErrorCode(t, err))
		})
	}
}

func Test_TestReplicationConnection_SummarizeWalOff_ReturnsWalSummaryDisabledCode(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			source := containers.StartPhysicalPostgres(t, fx.image, containers.WithoutSummarizer())

			m := newTestModelAt(t, source.Host, source.Port)
			m.BackupType = BackupTypeFullAndIncremental

			err := m.TestReplicationConnection(testLogger(), nil)

			assert.Equal(t, postgresql_shared.ConnErrWalSummaryDisabled, connTestErrorCode(t, err))
		})
	}
}

// Test_TestReplicationConnection_HostAllButNoReplicationHba_ReturnsPgHbaNoEntryCode pins the gap
// between what the test checks and what backups need: a cluster with only "host all all all" in
// pg_hba accepts an ordinary connect, yet pg_basebackup / pg_receivewal use physical replication,
// which pg_hba matches only via a "host replication" rule. The test must refuse such a cluster.
func Test_TestReplicationConnection_HostAllButNoReplicationHba_ReturnsPgHbaNoEntryCode(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			source := containers.StartPhysicalPostgres(t, fx.image, containers.WithoutReplicationHbaEntry())

			m := newTestModelAt(t, source.Host, source.Port)
			m.BackupType = BackupTypeFullOnly

			err := m.TestReplicationConnection(testLogger(), nil)

			assert.Equal(t, postgresql_shared.ConnErrPgHbaNoEntry, connTestErrorCode(t, err))
			assert.Contains(t, err.Error(),
				fmt.Sprintf("host replication %s all scram-sha-256", m.Username))
		})
	}
}

func connTestErrorCode(t *testing.T, err error) postgresql_shared.ConnectionErrorCode {
	t.Helper()

	var connErr *postgresql_shared.ConnectionTestError
	require.ErrorAs(t, err, &connErr)

	return connErr.Code
}

func Test_GetClusterSizeMb_ReturnsPositiveValue(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())

			size, err := m.GetClusterSizeMb(context.Background(), testLogger(), nil)
			require.NoError(t, err)
			assert.Greater(t, size, 0.0)
		})
	}
}

func Test_IsUserReplicationOnly_DetectsSuperuser(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())

			isMinimal, excessive, err := m.IsUserReplicationOnly(context.Background(), testLogger(), nil)
			require.NoError(t, err)
			assert.False(t, isMinimal)
			assert.Contains(t, excessive, "SUPERUSER")
		})
	}
}

func Test_IsUserReplicationOnly_TrueForFreshlyCreatedReplicationUser(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			provisioner := newTestModel(t, fx.port())
			username, password, err := provisioner.CreateReplicationOnlyUser(
				context.Background(), testLogger(), nil,
			)
			require.NoError(t, err)

			t.Cleanup(func() {
				setupConn := openTestConn(t, fx.port())
				defer setupConn.Close(context.Background())
				_, _ = setupConn.Exec(
					context.Background(),
					fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username),
				)
			})

			m := newTestModel(t, fx.port())
			m.Username = username
			m.Password = password

			isMinimal, excessive, err := m.IsUserReplicationOnly(context.Background(), testLogger(), nil)
			require.NoError(t, err)
			assert.True(t, isMinimal, "excessive=%v", excessive)
			assert.Empty(t, excessive)
		})
	}
}

func Test_IsUserReplicationOnly_DetectsTableWritePrivilege(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			username, password := createTempUser(t, setupConn, "")

			_, err := setupConn.Exec(context.Background(),
				`CREATE TABLE IF NOT EXISTS public.write_probe (id int)`)
			require.NoError(t, err)
			t.Cleanup(func() {
				_, _ = setupConn.Exec(context.Background(),
					`DROP TABLE IF EXISTS public.write_probe`)
			})

			_, err = setupConn.Exec(context.Background(),
				fmt.Sprintf(`GRANT INSERT ON public.write_probe TO "%s"`, username))
			require.NoError(t, err)
			_, err = setupConn.Exec(context.Background(),
				fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO "%s"`, username))
			require.NoError(t, err)

			m := newTestModel(t, fx.port())
			m.Username = username
			m.Password = password

			isMinimal, excessive, err := m.IsUserReplicationOnly(context.Background(), testLogger(), nil)
			require.NoError(t, err)
			assert.False(t, isMinimal)
			assert.Contains(t, excessive, "INSERT")
		})
	}
}

func Test_CreateReplicationOnlyUser_HappyPath(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())

			username, password, err := m.CreateReplicationOnlyUser(
				context.Background(), testLogger(), nil,
			)
			require.NoError(t, err)
			assert.True(t,
				len(username) > len("databasus-") && username[:len("databasus-")] == "databasus-",
				"username=%s", username)
			assert.NotEmpty(t, password)

			t.Cleanup(func() {
				setupConn := openTestConn(t, fx.port())
				defer setupConn.Close(context.Background())
				_, _ = setupConn.Exec(
					context.Background(),
					fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username),
				)
			})

			verifyConn := openTestConn(t, fx.port())
			defer verifyConn.Close(context.Background())

			var isSuper, canCreateRole, canCreateDB, canBypassRLS, canRepl bool
			require.NoError(t, verifyConn.QueryRow(context.Background(), `
				SELECT rolsuper, rolcreaterole, rolcreatedb, rolbypassrls, rolreplication
				FROM pg_roles
				WHERE rolname = $1
			`, username).Scan(&isSuper, &canCreateRole, &canCreateDB, &canBypassRLS, &canRepl))

			assert.False(t, isSuper)
			assert.False(t, canCreateRole)
			assert.False(t, canCreateDB)
			assert.False(t, canBypassRLS)
			assert.True(t, canRepl, "new user must have rolreplication=true on self-managed")
		})
	}
}

func Test_CreateReplicationOnlyUser_NewUserCanOpenReplicationConnection(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			provisioner := newTestModel(t, fx.port())
			username, password, err := provisioner.CreateReplicationOnlyUser(
				context.Background(), testLogger(), nil,
			)
			require.NoError(t, err)

			t.Cleanup(func() {
				setupConn := openTestConn(t, fx.port())
				defer setupConn.Close(context.Background())
				_, _ = setupConn.Exec(
					context.Background(),
					fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username),
				)
			})

			portInt, err := strconv.Atoi(fx.port())
			require.NoError(t, err)

			dsn := fmt.Sprintf(
				"host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable replication=true",
				config.GetEnv().TestLocalhost, portInt, username, password,
			)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			conn, err := pgx.Connect(ctx, dsn)
			require.NoError(t, err)
			defer conn.Close(ctx)
		})
	}
}

func Test_CreateReplicationOnlyUser_FailsWhenCurrentUserCannotCreateRole(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			username, password := createTempUser(t, setupConn, "")

			m := newTestModel(t, fx.port())
			m.Username = username
			m.Password = password

			_, _, err := m.CreateReplicationOnlyUser(context.Background(), testLogger(), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "create roles")
		})
	}
}

func Test_DetectPlatform_SelfManagedByDefault(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			require.False(t, roleExistsHelper(t, setupConn, "rds_replication"))
			require.False(t, roleExistsHelper(t, setupConn, "azure_pg_admin"))
			require.False(t, roleExistsHelper(t, setupConn, "cloudsqlsuperuser"))

			plat := detectPlatform(context.Background(), setupConn)
			assert.Equal(t, platformSelfManaged, plat)
		})
	}
}

func Test_DetectPlatform_RdsWhenRdsReplicationRoleExists(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			createMarkerRole(t, setupConn, "rds_replication")

			plat := detectPlatform(context.Background(), setupConn)
			assert.Equal(t, platformRds, plat)
		})
	}
}

func Test_DetectPlatform_AzureWhenAzurePgAdminExists(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			createMarkerRole(t, setupConn, "azure_pg_admin")

			plat := detectPlatform(context.Background(), setupConn)
			assert.Equal(t, platformAzure, plat)
		})
	}
}

func Test_DetectPlatform_GcpWhenCloudsqlSuperuserExists(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			createMarkerRole(t, setupConn, "cloudsqlsuperuser")

			plat := detectPlatform(context.Background(), setupConn)
			assert.Equal(t, platformGcp, plat)
		})
	}
}

func Test_DetectPlatform_RdsTakesPrecedenceOverOthers(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			createMarkerRole(t, setupConn, "rds_replication")
			createMarkerRole(t, setupConn, "azure_pg_admin")
			createMarkerRole(t, setupConn, "cloudsqlsuperuser")

			plat := detectPlatform(context.Background(), setupConn)
			assert.Equal(t, platformRds, plat)
		})
	}
}

func Test_CreateReplicationOnlyUser_OnSimulatedRds_GrantsRdsReplicationMembership(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			createMarkerRole(t, setupConn, "rds_replication")

			m := newTestModel(t, fx.port())
			username, _, err := m.CreateReplicationOnlyUser(
				context.Background(), testLogger(), nil,
			)
			require.NoError(t, err)

			t.Cleanup(func() {
				_, _ = setupConn.Exec(
					context.Background(),
					fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username),
				)
			})

			var isMember bool
			require.NoError(t, setupConn.QueryRow(
				context.Background(),
				`SELECT pg_has_role($1, 'rds_replication', 'MEMBER')`,
				username,
			).Scan(&isMember))
			assert.True(t, isMember, "user must be member of rds_replication on RDS path")

			var canRepl bool
			require.NoError(t, setupConn.QueryRow(
				context.Background(),
				`SELECT rolreplication FROM pg_roles WHERE rolname = $1`,
				username,
			).Scan(&canRepl))
			assert.False(t, canRepl, "RDS path uses GRANT, not ALTER ROLE REPLICATION")
		})
	}
}

func Test_IsUserReplicationOnly_OnSimulatedRds_FlagsRdsSuperuserMembership(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			createMarkerRole(t, setupConn, "rds_superuser")
			grantRoleToCurrent(t, setupConn, "rds_superuser")

			m := newTestModel(t, fx.port())

			_, excessive, err := m.IsUserReplicationOnly(context.Background(), testLogger(), nil)
			require.NoError(t, err)
			assert.Contains(t, excessive, "rds_superuser (RDS admin)")
		})
	}
}

func Test_IsUserReplicationOnly_OnSimulatedAzure_FlagsAzurePgAdminMembership(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			createMarkerRole(t, setupConn, "azure_pg_admin")
			grantRoleToCurrent(t, setupConn, "azure_pg_admin")

			m := newTestModel(t, fx.port())

			_, excessive, err := m.IsUserReplicationOnly(context.Background(), testLogger(), nil)
			require.NoError(t, err)
			assert.Contains(t, excessive, "azure_pg_admin (Azure admin)")
		})
	}
}

func Test_IsUserReplicationOnly_OnSimulatedGcp_FlagsCloudsqlSuperuserMembership(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			setupConn := openTestConn(t, fx.port())

			createMarkerRole(t, setupConn, "cloudsqlsuperuser")
			grantRoleToCurrent(t, setupConn, "cloudsqlsuperuser")

			m := newTestModel(t, fx.port())

			_, excessive, err := m.IsUserReplicationOnly(context.Background(), testLogger(), nil)
			require.NoError(t, err)
			assert.Contains(t, excessive, "cloudsqlsuperuser (GCP admin)")
		})
	}
}

func dropSlotIfExists(t *testing.T, conn *pgx.Conn, slotName string) {
	t.Helper()

	_, _ = conn.Exec(context.Background(),
		`SELECT pg_drop_replication_slot(slot_name)
		 FROM pg_replication_slots
		 WHERE slot_name = $1`,
		slotName,
	)
}

func skipIfNotLogicalWalLevel(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	var walLevel string
	require.NoError(t, conn.QueryRow(context.Background(), "SHOW wal_level").Scan(&walLevel))

	if walLevel != "logical" {
		t.Fatalf("logical slot path requires wal_level=logical, got %q", walLevel)
	}
}

func Test_VerifyWalSlot_CreatesSlotIfMissing(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			adminConn := openTestConn(t, fx.port())

			t.Cleanup(func() {
				dropSlotIfExists(t, adminConn, m.ReplicationSlotName)
			})

			require.NoError(t, m.VerifyWalSlot(t.Context(), testLogger(), nil))

			var slotType string
			require.NoError(t, adminConn.QueryRow(t.Context(),
				"SELECT slot_type FROM pg_replication_slots WHERE slot_name = $1",
				m.ReplicationSlotName,
			).Scan(&slotType))

			assert.Equal(t, "physical", slotType)
		})
	}
}

func Test_VerifyWalSlot_IsIdempotent(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			adminConn := openTestConn(t, fx.port())

			t.Cleanup(func() {
				dropSlotIfExists(t, adminConn, m.ReplicationSlotName)
			})

			require.NoError(t, m.VerifyWalSlot(t.Context(), testLogger(), nil))
			require.NoError(t, m.VerifyWalSlot(t.Context(), testLogger(), nil))

			var count int
			require.NoError(t, adminConn.QueryRow(t.Context(),
				"SELECT count(*) FROM pg_replication_slots WHERE slot_name = $1",
				m.ReplicationSlotName,
			).Scan(&count))

			assert.Equal(t, 1, count)
		})
	}
}

func Test_VerifyWalSlot_RefusesLogicalSlotWithSameName(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			adminConn := openTestConn(t, fx.port())
			skipIfNotLogicalWalLevel(t, adminConn)

			_, err := adminConn.Exec(t.Context(),
				"SELECT pg_create_logical_replication_slot($1, 'test_decoding')",
				m.ReplicationSlotName,
			)
			require.NoError(t, err)

			t.Cleanup(func() {
				dropSlotIfExists(t, adminConn, m.ReplicationSlotName)
			})

			err = m.VerifyWalSlot(t.Context(), testLogger(), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "expected physical")
		})
	}
}

func Test_VerifyWalSlot_ErrorsOnEmptySlotName(t *testing.T) {
	m := &PostgresqlPhysicalDatabase{ReplicationSlotName: ""}

	err := m.VerifyWalSlot(t.Context(), testLogger(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func Test_DropWalSlot_DropsExistingSlot(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			adminConn := openTestConn(t, fx.port())

			t.Cleanup(func() {
				dropSlotIfExists(t, adminConn, m.ReplicationSlotName)
			})

			_, err := adminConn.Exec(t.Context(),
				"SELECT pg_create_physical_replication_slot($1)",
				m.ReplicationSlotName,
			)
			require.NoError(t, err)

			require.NoError(t, m.DropWalSlot(t.Context(), testLogger(), nil))

			var exists bool
			require.NoError(t, adminConn.QueryRow(t.Context(),
				"SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name = $1)",
				m.ReplicationSlotName,
			).Scan(&exists))

			assert.False(t, exists)
		})
	}
}

func Test_DropWalSlot_NoOpIfSlotMissing(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())

			assert.NoError(t, m.DropWalSlot(t.Context(), testLogger(), nil))
		})
	}
}

func Test_DropWalSlot_RefusesLogicalSlotWithSameName(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			adminConn := openTestConn(t, fx.port())
			skipIfNotLogicalWalLevel(t, adminConn)

			_, err := adminConn.Exec(t.Context(),
				"SELECT pg_create_logical_replication_slot($1, 'test_decoding')",
				m.ReplicationSlotName,
			)
			require.NoError(t, err)

			t.Cleanup(func() {
				dropSlotIfExists(t, adminConn, m.ReplicationSlotName)
			})

			err = m.DropWalSlot(t.Context(), testLogger(), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "expected physical")

			var exists bool
			require.NoError(t, adminConn.QueryRow(t.Context(),
				"SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name = $1)",
				m.ReplicationSlotName,
			).Scan(&exists))

			assert.True(t, exists)
		})
	}
}

func Test_DropWalSlot_ErrorsOnEmptySlotName(t *testing.T) {
	m := &PostgresqlPhysicalDatabase{ReplicationSlotName: ""}

	err := m.DropWalSlot(t.Context(), testLogger(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func Test_DropWalSlotForRemoval_DropsExistingInactiveSlot(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			adminConn := openTestConn(t, fx.port())

			t.Cleanup(func() {
				dropSlotIfExists(t, adminConn, m.ReplicationSlotName)
			})

			_, err := adminConn.Exec(t.Context(),
				"SELECT pg_create_physical_replication_slot($1)",
				m.ReplicationSlotName,
			)
			require.NoError(t, err)

			require.NoError(t, m.DropWalSlotForRemoval(t.Context(), testLogger(), nil))

			assert.False(t, slotExists(t, adminConn, m.ReplicationSlotName))
		})
	}
}

// Test_DropWalSlotForRemoval_TerminatesConsumerAndDropsActiveSlot pins the behavior
// that separates DropWalSlotForRemoval from DropWalSlot: an active slot (a real
// pg_receivewal still attached) is force-dropped — its consumer is evicted and the
// slot removed — rather than refused. This is what guarantees a deleted database
// never leaves WAL pinned. The consumer runs with --no-loop so it does not reconnect
// and re-activate the slot mid-drop.
func Test_DropWalSlotForRemoval_TerminatesConsumerAndDropsActiveSlot(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			adminConn := openTestConn(t, fx.port())

			t.Cleanup(func() {
				dropSlotIfExists(t, adminConn, m.ReplicationSlotName)
			})

			_, err := adminConn.Exec(t.Context(),
				"SELECT pg_create_physical_replication_slot($1, true)",
				m.ReplicationSlotName,
			)
			require.NoError(t, err)

			stopConsumer := startSlotConsumer(t, fx, m.ReplicationSlotName)
			defer stopConsumer()

			waitForSlotActive(t, adminConn, m.ReplicationSlotName, 15*time.Second)

			require.NoError(t, m.DropWalSlotForRemoval(t.Context(), testLogger(), nil))

			assert.False(t, slotExists(t, adminConn, m.ReplicationSlotName),
				"an active slot must be force-dropped when its database is removed")
		})
	}
}

func Test_DropWalSlotForRemoval_NoOpIfSlotMissing(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())

			assert.NoError(t, m.DropWalSlotForRemoval(t.Context(), testLogger(), nil))
		})
	}
}

func Test_DropWalSlotForRemoval_RefusesLogicalSlotWithSameName(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			if fx.port() == "" {
				t.Fatalf("%s port not configured", fx.name)
			}

			m := newTestModel(t, fx.port())
			adminConn := openTestConn(t, fx.port())
			skipIfNotLogicalWalLevel(t, adminConn)

			_, err := adminConn.Exec(t.Context(),
				"SELECT pg_create_logical_replication_slot($1, 'test_decoding')",
				m.ReplicationSlotName,
			)
			require.NoError(t, err)

			t.Cleanup(func() {
				dropSlotIfExists(t, adminConn, m.ReplicationSlotName)
			})

			err = m.DropWalSlotForRemoval(t.Context(), testLogger(), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "expected physical")

			assert.True(t, slotExists(t, adminConn, m.ReplicationSlotName),
				"a same-named logical slot must never be dropped")
		})
	}
}

func Test_DropWalSlotForRemoval_ErrorsOnEmptySlotName(t *testing.T) {
	m := &PostgresqlPhysicalDatabase{ReplicationSlotName: ""}

	err := m.DropWalSlotForRemoval(t.Context(), testLogger(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

// startSlotConsumer attaches a real pg_receivewal to slotName so the slot reports
// active, returning a stop func. It runs with --no-loop (-n) so that once
// DropWalSlotForRemoval terminates the backend, pg_receivewal exits instead of
// reconnecting and re-activating the slot.
func startSlotConsumer(t *testing.T, fx pgFixture, slotName string) func() {
	t.Helper()

	pgBin := tools.GetPostgresqlExecutable(fx.version, tools.PostgresqlExecutablePgReceivewal)
	dsn := fmt.Sprintf(
		"host=%s port=%s user=testuser password=testpassword dbname=postgres sslmode=disable",
		config.GetEnv().TestLocalhost, fx.port(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, pgBin, "-n", "-D", t.TempDir(), "-S", slotName, "-d", dsn)
	require.NoError(t, cmd.Start())

	return func() {
		cancel()
		_ = cmd.Wait()
	}
}

// waitForSlotActive blocks until slotName has an attached consumer (active = true).
func waitForSlotActive(t *testing.T, conn *pgx.Conn, slotName string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var isActive bool
		err := conn.QueryRow(context.Background(),
			"SELECT active FROM pg_replication_slots WHERE slot_name = $1", slotName,
		).Scan(&isActive)
		if err == nil && isActive {
			return
		}

		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("replication slot %q never became active within %s", slotName, timeout)
}

func slotExists(t *testing.T, conn *pgx.Conn, slotName string) bool {
	t.Helper()

	var exists bool
	require.NoError(t, conn.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name = $1)",
		slotName,
	).Scan(&exists))

	return exists
}

func Test_BeforeCreate_FillsReplicationSlotNameIfBlank(t *testing.T) {
	m := &PostgresqlPhysicalDatabase{}

	require.NoError(t, m.BeforeCreate(nil))

	assert.True(t,
		len(m.ReplicationSlotName) > len("databasus_slot_") &&
			m.ReplicationSlotName[:len("databasus_slot_")] == "databasus_slot_",
		"got %q", m.ReplicationSlotName,
	)
}

func Test_BeforeCreate_PreservesExistingReplicationSlotName(t *testing.T) {
	m := &PostgresqlPhysicalDatabase{ReplicationSlotName: "preserved_slot"}

	require.NoError(t, m.BeforeCreate(nil))

	assert.Equal(t, "preserved_slot", m.ReplicationSlotName)
}

var _ encryption.FieldEncryptor = noopEncryptor{}
