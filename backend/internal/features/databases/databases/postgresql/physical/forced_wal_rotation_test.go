package postgresql_physical

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
)

func grantWalSwitchTo(t *testing.T, conn *pgx.Conn, username string) {
	t.Helper()

	_, err := conn.Exec(
		context.Background(),
		fmt.Sprintf(`GRANT EXECUTE ON FUNCTION pg_switch_wal() TO "%s"`, username),
	)
	require.NoError(t, err)
}

func Test_CanForceWalRotation_ForReplicationOnlyRole_ReturnsFalse(t *testing.T) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			conn := openTestConn(t, fixture.port())
			username, _ := createTempUser(t, conn, "REPLICATION")

			canRotate, err := canForceWalRotation(context.Background(), conn, username)

			require.NoError(t, err)
			assert.False(t, canRotate)
		})
	}
}

func Test_CanForceWalRotation_AfterExecuteIsGranted_ReturnsTrue(t *testing.T) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			conn := openTestConn(t, fixture.port())
			username, _ := createTempUser(t, conn, "REPLICATION")
			grantWalSwitchTo(t, conn, username)

			canRotate, err := canForceWalRotation(context.Background(), conn, username)

			require.NoError(t, err)
			assert.True(t, canRotate)
		})
	}
}

// The connected administrator can always execute pg_switch_wal, so a probe that ignored
// its role argument would report every source as capable.
func Test_CanForceWalRotation_WhenAnAdminProbesAnotherRole_AnswersForThatRole(t *testing.T) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			conn := openTestConn(t, fixture.port())
			username, _ := createTempUser(t, conn, "REPLICATION")

			canAdminRotate, err := canForceWalRotation(context.Background(), conn, "testuser")
			require.NoError(t, err)
			require.True(t, canAdminRotate, "the test administrator must be able to rotate")

			canRoleRotate, err := canForceWalRotation(context.Background(), conn, username)

			require.NoError(t, err)
			assert.False(t, canRoleRotate)
		})
	}
}

func Test_TestReplicationConnection_ForWalStreamWhenRoleCannotSwitchWal_ReturnsNoWalSwitchPrivilegeCode(
	t *testing.T,
) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			conn := openTestConn(t, fixture.port())
			username, password := createTempUser(t, conn, "REPLICATION")

			physicalDatabase := newTestModel(t, fixture.port())
			physicalDatabase.Username = username
			physicalDatabase.Password = password
			physicalDatabase.BackupType = BackupTypeFullIncrementalAndWalStream

			err := physicalDatabase.TestReplicationConnection(testLogger(), nil)

			assert.Equal(t, postgresql_shared.ConnErrNoWalSwitchPrivilege, connTestErrorCode(t, err))
		})
	}
}

func Test_TestReplicationConnection_ForWalStreamAfterExecuteIsGranted_Succeeds(t *testing.T) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			conn := openTestConn(t, fixture.port())
			username, password := createTempUser(t, conn, "REPLICATION")
			grantWalSwitchTo(t, conn, username)

			physicalDatabase := newTestModel(t, fixture.port())
			physicalDatabase.Username = username
			physicalDatabase.Password = password
			physicalDatabase.BackupType = BackupTypeFullIncrementalAndWalStream

			assert.NoError(t, physicalDatabase.TestReplicationConnection(testLogger(), nil))
		})
	}
}

// FULL and incremental restores replay no archived WAL, so demanding the privilege for
// them would refuse sources that back up correctly.
func Test_TestReplicationConnection_WhenRoleCannotSwitchWal_SucceedsForBackupTypesWithoutWalReplay(
	t *testing.T,
) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			conn := openTestConn(t, fixture.port())
			username, password := createTempUser(t, conn, "REPLICATION")

			for _, backupType := range []BackupType{BackupTypeFullOnly, BackupTypeFullAndIncremental} {
				physicalDatabase := newTestModel(t, fixture.port())
				physicalDatabase.Username = username
				physicalDatabase.Password = password
				physicalDatabase.BackupType = backupType

				assert.NoErrorf(t, physicalDatabase.TestReplicationConnection(testLogger(), nil),
					"backup type %s must not require the pg_switch_wal privilege", backupType)
			}
		})
	}
}

func dropUserAfterTest(t *testing.T, conn *pgx.Conn, username string) {
	t.Helper()

	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
	})
}

func Test_CreateReplicationOnlyUser_OnSelfManagedSource_GrantsForcedWalRotation(t *testing.T) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			conn := openTestConn(t, fixture.port())

			createdUser, err := newTestModel(t, fixture.port()).
				CreateReplicationOnlyUser(context.Background(), testLogger(), nil)
			require.NoError(t, err)
			dropUserAfterTest(t, conn, createdUser.Username)

			assert.True(t, createdUser.IsForcedWalRotationAvailable)

			physicalDatabase := newTestModel(t, fixture.port())
			physicalDatabase.Username = createdUser.Username
			physicalDatabase.Password = createdUser.Password
			physicalDatabase.BackupType = BackupTypeFullIncrementalAndWalStream

			assert.NoError(t, physicalDatabase.TestReplicationConnection(testLogger(), nil))
		})
	}
}

func Test_CreateReplicationOnlyUser_WhenTheSourceRefusesTheGrant_StillReturnsWorkingCredentials(
	t *testing.T,
) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			conn := openTestConn(t, fixture.port())

			// A role that may create roles and confer REPLICATION, but owns none of the
			// pg_catalog functions, is what a managed platform's administrator looks like.
			adminName, adminPassword := createTempUser(t, conn, "CREATEROLE REPLICATION")

			provisioner := newTestModel(t, fixture.port())
			provisioner.Username = adminName
			provisioner.Password = adminPassword

			createdUser, err := provisioner.CreateReplicationOnlyUser(
				context.Background(), testLogger(), nil,
			)
			require.NoError(t, err)
			dropUserAfterTest(t, conn, createdUser.Username)

			assert.False(t, createdUser.IsForcedWalRotationAvailable)

			physicalDatabase := newTestModel(t, fixture.port())
			physicalDatabase.Username = createdUser.Username
			physicalDatabase.Password = createdUser.Password
			physicalDatabase.BackupType = BackupTypeFullAndIncremental

			assert.NoError(t, physicalDatabase.TestReplicationConnection(testLogger(), nil))
		})
	}
}

func Test_CreateReplicationOnlyUser_ConfersForcedWalRotationOnTheCreatedRoleAlone(t *testing.T) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			conn := openTestConn(t, fixture.port())

			bystanderName, _ := createTempUser(t, conn, "REPLICATION")

			createdUser, err := newTestModel(t, fixture.port()).
				CreateReplicationOnlyUser(context.Background(), testLogger(), nil)
			require.NoError(t, err)
			dropUserAfterTest(t, conn, createdUser.Username)

			canBystanderRotate, err := canForceWalRotation(context.Background(), conn, bystanderName)
			require.NoError(t, err)
			assert.False(t, canBystanderRotate, "the grant must not reach any other role")

			var isSuper, canCreateRole, canCreateDB, canBypassRLS bool
			require.NoError(t, conn.QueryRow(context.Background(), `
				SELECT rolsuper, rolcreaterole, rolcreatedb, rolbypassrls
				FROM pg_roles
				WHERE rolname = $1
			`, createdUser.Username).Scan(&isSuper, &canCreateRole, &canCreateDB, &canBypassRLS))

			assert.False(t, isSuper)
			assert.False(t, canCreateRole)
			assert.False(t, canCreateDB)
			assert.False(t, canBypassRLS)
		})
	}
}
