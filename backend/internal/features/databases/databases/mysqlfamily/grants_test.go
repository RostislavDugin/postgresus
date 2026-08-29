package mysqlfamily

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseGrantLine_WithGlobalSchemaAndTableScopes_ReturnsScopeAndPrivileges(t *testing.T) {
	grantCases := []struct {
		name          string
		grantLine     string
		privileges    []string
		scope         GrantScope
		schemaPattern string
		tableName     string
	}{
		{
			// issue #568: the privilege list must be split before it is read, or "SHOW CREATE
			// ROUTINE" registers as a CREATE grant.
			name:       "SHOW CREATE ROUTINE is not split into CREATE",
			grantLine:  "GRANT SELECT, SHOW VIEW, SHOW CREATE ROUTINE ON *.* TO 'backup'@'%'",
			privileges: []string{"SELECT", "SHOW VIEW", "SHOW CREATE ROUTINE"},
			scope:      GrantScopeGlobal,
		},
		{
			name:       "global write privileges",
			grantLine:  "GRANT SELECT, INSERT, UPDATE ON *.* TO 'x'@'%'",
			privileges: []string{"SELECT", "INSERT", "UPDATE"},
			scope:      GrantScopeGlobal,
		},
		{
			name:          "schema scoped all privileges",
			grantLine:     "GRANT ALL PRIVILEGES ON `appdb`.* TO 'x'@'%'",
			privileges:    []string{"ALL PRIVILEGES"},
			scope:         GrantScopeSchema,
			schemaPattern: "appdb",
		},
		{
			name:       "usage only line",
			grantLine:  "GRANT USAGE ON *.* TO 'x'@'%'",
			privileges: []string{"USAGE"},
			scope:      GrantScopeGlobal,
		},
		{
			name:          "column level privilege does not count as table wide",
			grantLine:     "GRANT SELECT (col1, col2), SHOW VIEW ON `appdb`.`orders` TO 'x'@'%'",
			privileges:    []string{"SHOW VIEW"},
			scope:         GrantScopeTable,
			schemaPattern: "appdb",
			tableName:     "orders",
		},
		{
			name:          "table scoped grant",
			grantLine:     "GRANT SELECT, SHOW VIEW ON `appdb`.`orders` TO 'backup'@'%'",
			privileges:    []string{"SELECT", "SHOW VIEW"},
			scope:         GrantScopeTable,
			schemaPattern: "appdb",
			tableName:     "orders",
		},
		{
			name:          "wildcard schema pattern is kept as written",
			grantLine:     "GRANT SELECT ON `app%`.* TO 'x'@'%'",
			privileges:    []string{"SELECT"},
			scope:         GrantScopeSchema,
			schemaPattern: "app%",
		},
		{
			name:          "escaped underscore in the schema pattern",
			grantLine:     "GRANT SELECT ON `test\\_db`.* TO 'x'@'%'",
			privileges:    []string{"SELECT"},
			scope:         GrantScopeSchema,
			schemaPattern: `test\_db`,
		},
		{
			name:          "schema name containing a dot",
			grantLine:     "GRANT SELECT ON `my.db`.`orders` TO 'x'@'%'",
			privileges:    []string{"SELECT"},
			scope:         GrantScopeTable,
			schemaPattern: "my.db",
			tableName:     "orders",
		},
		{
			name:       "with grant option trailer is ignored",
			grantLine:  "GRANT SELECT, INSERT ON *.* TO 'x'@'%' WITH GRANT OPTION",
			privileges: []string{"SELECT", "INSERT"},
			scope:      GrantScopeGlobal,
		},
		{
			name:       "mixed case keywords",
			grantLine:  "grant Select, Update on *.* to 'x'@'%'",
			privileges: []string{"SELECT", "UPDATE"},
			scope:      GrantScopeGlobal,
		},
	}

	for _, grantCase := range grantCases {
		t.Run(grantCase.name, func(t *testing.T) {
			grant := ParseGrantLine(grantCase.grantLine)

			require.NotNil(t, grant)
			assert.Equal(t, grantCase.privileges, grant.Privileges)
			assert.Equal(t, grantCase.scope, grant.Scope)
			assert.Equal(t, grantCase.schemaPattern, grant.SchemaPattern)
			assert.Equal(t, grantCase.tableName, grant.TableName)
		})
	}
}

func Test_ParseGrantLine_WithoutAGrantedScope_ReturnsNil(t *testing.T) {
	ignoredGrantLines := []struct {
		name      string
		grantLine string
	}{
		{"role grant has no ON clause", "GRANT `reporting_role`@`%` TO 'u'@'%'"},
		{"proxy grant names an account", "GRANT PROXY ON 'other'@'%' TO 'u'@'%'"},
		{"revoke line is not a grant", "REVOKE SELECT ON `appdb`.* FROM 'u'@'%'"},
		{
			"every privilege is column qualified",
			"GRANT SELECT (on) ON `appdb`.`orders` TO 'x'@'%'",
		},
	}

	for _, ignoredGrantLine := range ignoredGrantLines {
		t.Run(ignoredGrantLine.name, func(t *testing.T) {
			assert.Nil(t, ParseGrantLine(ignoredGrantLine.grantLine))
		})
	}
}

func Test_IsMatchingSchemaPattern_WithWildcardsAndEscapes_MatchesServerRule(t *testing.T) {
	matchCases := []struct {
		name          string
		schemaPattern string
		schemaName    string
		isMatching    bool
	}{
		{"plain name matches itself", "appdb", "appdb", true},
		{"plain name does not match another", "appdb", "otherdb", false},
		{"percent matches a suffix", "app%", "appdb", true},
		{"percent matches everything", "%", "anything", true},
		{"underscore matches one character", "app_b", "appXb", true},
		{"escaped underscore is literal", `test\_db`, "test_db", true},
		{"escaped underscore rejects another character", `test\_db`, "testXdb", false},
		{"pattern must match the whole name", "app", "appdb", false},
	}

	for _, matchCase := range matchCases {
		t.Run(matchCase.name, func(t *testing.T) {
			assert.Equal(
				t,
				matchCase.isMatching,
				isMatchingSchemaPattern(matchCase.schemaPattern, matchCase.schemaName),
			)
		})
	}
}

func Test_CollectSchemaGrants_WhenAllPrivilegesIsTableScoped_DoesNotGrantEvent(t *testing.T) {
	collectedGrants := CollectSchemaGrants(
		[]string{"GRANT ALL PRIVILEGES ON `appdb`.`orders` TO 'x'@'%'"},
		"appdb",
	)

	assert.True(t, collectedGrants.HasForTable("SELECT", "orders"))
	assert.True(t, collectedGrants.HasForTable("SHOW VIEW", "orders"))
	assert.False(t, collectedGrants.HasForTable("EVENT", "orders"))
}

func Test_CollectSchemaGrants_WhenGrantTargetsAnotherSchema_IsIgnored(t *testing.T) {
	collectedGrants := CollectSchemaGrants(
		[]string{"GRANT SELECT ON `otherdb`.* TO 'x'@'%'"},
		"appdb",
	)

	assert.False(t, collectedGrants.HasForSchema("SELECT"))
}

func Test_CollectSchemaGrants_WhenSchemaIsPartiallyRevoked_SubtractsFromGlobal(t *testing.T) {
	collectedGrants := CollectSchemaGrants(
		[]string{
			"GRANT SELECT, SHOW VIEW ON *.* TO 'x'@'%'",
			"REVOKE SELECT ON `appdb`.* FROM 'x'@'%'",
		},
		"appdb",
	)

	assert.False(t, collectedGrants.HasForSchema("SELECT"))
	assert.True(t, collectedGrants.HasForSchema("SHOW VIEW"))
}

func Test_CollectSchemaGrants_WhenRevokeUsesSchemaPattern_SubtractsFromGlobal(t *testing.T) {
	collectedGrants := CollectSchemaGrants(
		[]string{
			"GRANT SELECT ON *.* TO 'x'@'%'",
			"REVOKE SELECT ON `app%`.* FROM 'x'@'%'",
		},
		"appdb",
	)

	assert.False(t, collectedGrants.HasForSchema("SELECT"))
}

func Test_CollectSchemaGrants_WhenAnotherSchemaIsRevoked_GlobalGrantStillCounts(t *testing.T) {
	collectedGrants := CollectSchemaGrants(
		[]string{
			"GRANT SELECT ON *.* TO 'x'@'%'",
			"REVOKE SELECT ON `otherdb`.* FROM 'x'@'%'",
		},
		"appdb",
	)

	assert.True(t, collectedGrants.HasForSchema("SELECT"))
}

func Test_CollectSchemaGrants_WhenRevokedSchemaIsGrantedExplicitly_SchemaGrantWins(t *testing.T) {
	collectedGrants := CollectSchemaGrants(
		[]string{
			"GRANT SELECT ON *.* TO 'x'@'%'",
			"REVOKE SELECT ON `appdb`.* FROM 'x'@'%'",
			"GRANT SELECT ON `appdb`.* TO 'x'@'%'",
		},
		"appdb",
	)

	assert.True(t, collectedGrants.HasForSchema("SELECT"))
}

func Test_CollectSchemaGrants_WhenAllPrivilegesAreRevoked_EveryRequiredOneIsGone(t *testing.T) {
	collectedGrants := CollectSchemaGrants(
		[]string{
			"GRANT ALL PRIVILEGES ON *.* TO 'x'@'%'",
			"REVOKE ALL PRIVILEGES ON `appdb`.* FROM 'x'@'%'",
		},
		"appdb",
	)

	assert.False(t, collectedGrants.HasForSchema("SELECT"))
	assert.False(t, collectedGrants.HasForSchema("SHOW VIEW"))
	assert.False(t, collectedGrants.HasForSchema("TRIGGER"))
	assert.False(t, collectedGrants.HasForSchema("EVENT"))
}
