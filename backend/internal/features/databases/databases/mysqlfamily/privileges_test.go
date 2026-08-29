package mysqlfamily

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBackupPrivilegesSpec(grantLines, dumpedTableNames []string) BackupPrivilegesSpec {
	dumpedTables := make([]DumpedTable, 0, len(dumpedTableNames))
	for _, tableName := range dumpedTableNames {
		dumpedTables = append(dumpedTables, DumpedTable{Name: tableName})
	}

	return BackupPrivilegesSpec{
		SchemaName:   "appdb",
		Grantee:      "'backup'@'%'",
		DumpedTables: dumpedTables,
		Grants:       CollectSchemaGrants(grantLines, "appdb"),
	}
}

func Test_GetBackupPrivileges_WhenSchemaGrantCoversDumpSet_ReportsSufficient(t *testing.T) {
	backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
		[]string{"GRANT SELECT, SHOW VIEW, TRIGGER ON `appdb`.* TO 'backup'@'%'"},
		[]string{"orders", "payments"},
	))

	assert.True(t, backupPrivileges.IsSufficientForDump())
	assert.Equal(t, "SELECT,SHOW VIEW,TRIGGER", backupPrivileges.GetEffectivePrivilegesCsv())
}

func Test_GetBackupPrivileges_WhenTableGrantsCoverEveryDumpedTable_ReportsSufficient(t *testing.T) {
	backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
		[]string{
			"GRANT SELECT, SHOW VIEW ON `appdb`.`orders` TO 'backup'@'%'",
			"GRANT SELECT, SHOW VIEW ON `appdb`.`payments` TO 'backup'@'%'",
		},
		[]string{"orders", "payments"},
	))

	assert.True(t, backupPrivileges.IsSufficientForDump())
	assert.Equal(t, "SELECT,SHOW VIEW", backupPrivileges.GetEffectivePrivilegesCsv())
}

// mysqldump needs SHOW VIEW only to emit a view's definition, so a schema of plain tables must not
// be rejected for lacking it.
func Test_GetBackupPrivileges_WhenSchemaHoldsNoViews_DoesNotRequireShowView(t *testing.T) {
	backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
		[]string{"GRANT SELECT ON `appdb`.* TO 'backup'@'%'"},
		[]string{"orders", "payments"},
	))

	assert.True(t, backupPrivileges.IsSufficientForDump())
}

// The base table sits in the fixture to show it is not implicated by a view's missing privilege.
func Test_GetBackupPrivileges_WhenEveryViewLacksShowView_ReportsSchemaScope(t *testing.T) {
	backupPrivileges := GetBackupPrivileges(BackupPrivilegesSpec{
		SchemaName: "appdb",
		Grantee:    "'backup'@'%'",
		DumpedTables: []DumpedTable{
			{Name: "orders"},
			{Name: "orders_view", IsView: true},
		},
		Grants: CollectSchemaGrants(
			[]string{"GRANT SELECT ON `appdb`.* TO 'backup'@'%'"},
			"appdb",
		),
	})

	require.Len(t, backupPrivileges.Missing, 1)
	assert.Equal(t, "SHOW VIEW", backupPrivileges.Missing[0].Privilege)
	assert.Equal(t, GrantScopeSchema, backupPrivileges.Missing[0].Scope)
}

func Test_GetBackupPrivileges_WhenOneOfTwoViewsLacksShowView_ListsThatView(t *testing.T) {
	backupPrivileges := GetBackupPrivileges(BackupPrivilegesSpec{
		SchemaName: "appdb",
		Grantee:    "'backup'@'%'",
		DumpedTables: []DumpedTable{
			{Name: "orders"},
			{Name: "granted_view", IsView: true},
			{Name: "ungranted_view", IsView: true},
		},
		Grants: CollectSchemaGrants(
			[]string{
				"GRANT SELECT ON `appdb`.* TO 'backup'@'%'",
				"GRANT SHOW VIEW ON `appdb`.`granted_view` TO 'backup'@'%'",
			},
			"appdb",
		),
	})

	require.Len(t, backupPrivileges.Missing, 1)
	assert.Equal(t, "SHOW VIEW", backupPrivileges.Missing[0].Privilege)
	assert.Equal(t, GrantScopeTable, backupPrivileges.Missing[0].Scope)
	assert.Equal(t, []string{"ungranted_view"}, backupPrivileges.Missing[0].Tables)
}

func Test_GetBackupPrivileges_WhenPrivilegeIsMissingOnSomeTables_ListsThoseTables(t *testing.T) {
	backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
		[]string{
			"GRANT SHOW VIEW ON `appdb`.* TO 'backup'@'%'",
			"GRANT SELECT ON `appdb`.`orders` TO 'backup'@'%'",
		},
		[]string{"orders", "payments"},
	))

	require.Len(t, backupPrivileges.Missing, 1)
	assert.Equal(t, "SELECT", backupPrivileges.Missing[0].Privilege)
	assert.Equal(t, GrantScopeTable, backupPrivileges.Missing[0].Scope)
	assert.Equal(t, []string{"payments"}, backupPrivileges.Missing[0].Tables)
	assert.False(t, backupPrivileges.IsSufficientForDump())
}

// A privilege is recorded only when it covers the whole dump, because the dump flags it gates
// are all-or-nothing for the run.
func Test_GetBackupPrivileges_WhenTriggerCoversOnlySomeTables_OmitsThePrivilege(t *testing.T) {
	backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
		[]string{
			"GRANT SELECT, SHOW VIEW ON `appdb`.* TO 'backup'@'%'",
			"GRANT TRIGGER ON `appdb`.`orders` TO 'backup'@'%'",
		},
		[]string{"orders", "payments"},
	))

	assert.True(t, backupPrivileges.IsSufficientForDump())
	assert.False(t, HasPrivilege(backupPrivileges.GetEffectivePrivilegesCsv(), "TRIGGER"))
}

func Test_GetBackupPrivileges_WhenSchemaHasNoTables_FallsBackToSchemaGrants(t *testing.T) {
	backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
		[]string{"GRANT SELECT, SHOW VIEW ON `appdb`.* TO 'backup'@'%'"},
		nil,
	))

	assert.True(t, backupPrivileges.IsSufficientForDump())
}

func Test_GetBackupPrivileges_WhenRoleHoldsNothing_ReportsTheMissingSelect(t *testing.T) {
	backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
		[]string{"GRANT USAGE ON *.* TO 'backup'@'%'"},
		[]string{"orders"},
	))

	require.Len(t, backupPrivileges.Missing, 1)
	assert.Equal(t, "SELECT", backupPrivileges.Missing[0].Privilege)
	assert.Equal(t, GrantScopeSchema, backupPrivileges.Missing[0].Scope)
	assert.False(t, backupPrivileges.IsSufficientForDump())
}

func Test_GetBackupPrivileges_WhenAllPrivilegesIsTableScoped_OmitsEvent(t *testing.T) {
	backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
		[]string{"GRANT ALL PRIVILEGES ON `appdb`.`orders` TO 'backup'@'%'"},
		[]string{"orders"},
	))

	assert.True(t, backupPrivileges.IsSufficientForDump())
	assert.False(t, HasPrivilege(backupPrivileges.GetEffectivePrivilegesCsv(), "EVENT"))
}

func Test_NewInsufficiencyError_WhenPrivilegeIsMissingOnSomeTables_SuggestsSchemaWideGrant(t *testing.T) {
	backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
		[]string{
			"GRANT SHOW VIEW ON `appdb`.* TO 'backup'@'%'",
			"GRANT SELECT ON `appdb`.`orders` TO 'backup'@'%'",
		},
		[]string{"orders", "payments"},
	))

	err := backupPrivileges.NewInsufficiencyError()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
	assert.Contains(t, err.Error(), "`appdb`.`payments`")
	assert.NotContains(t, err.Error(), "`appdb`.`orders`")
	assert.Contains(t, err.Error(), "GRANT SELECT ON `appdb`.* TO 'backup'@'%';")
}

func Test_NewInsufficiencyError_WhenManyTablesAreMissingSelect_CapsTheTableList(t *testing.T) {
	var dumpedTables []string
	for tableIndex := range 9 {
		dumpedTables = append(dumpedTables, fmt.Sprintf("table_%d", tableIndex))
	}

	backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
		[]string{
			"GRANT SHOW VIEW ON `appdb`.* TO 'backup'@'%'",
			"GRANT SELECT ON `appdb`.`table_0` TO 'backup'@'%'",
		},
		dumpedTables,
	))

	err := backupPrivileges.NewInsufficiencyError()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "and 3 more")
}

func Test_HasPrivilege_WithCommaSeparatedTokens_MatchesWholeTokensOnly(t *testing.T) {
	assert.True(t, HasPrivilege("SELECT,SHOW VIEW,TRIGGER", "SHOW VIEW"))
	assert.False(t, HasPrivilege("SELECT,SHOW VIEW", "TRIGGER"))
	assert.False(t, HasPrivilege("SHOW CREATE ROUTINE", "CREATE"))
	assert.False(t, HasPrivilege("", "SELECT"))
}
