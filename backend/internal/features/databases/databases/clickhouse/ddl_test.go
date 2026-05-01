package clickhouse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanDDLForBackup_StripsOnClusterClause(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "bare cluster name",
			in:   "CREATE TABLE foo ON CLUSTER mycluster (id Int64) ENGINE = MergeTree() ORDER BY id",
			want: "CREATE TABLE foo (id Int64) ENGINE = MergeTree() ORDER BY id",
		},
		{
			name: "backquoted cluster name",
			in:   "CREATE TABLE foo ON CLUSTER `my-cluster` (id Int64) ENGINE = MergeTree()",
			want: "CREATE TABLE foo (id Int64) ENGINE = MergeTree()",
		},
		{
			name: "single-quoted cluster name",
			in:   "CREATE TABLE foo ON CLUSTER 'cl{shard}' (id Int64) ENGINE = MergeTree()",
			want: "CREATE TABLE foo (id Int64) ENGINE = MergeTree()",
		},
		{
			name: "no ON CLUSTER",
			in:   "CREATE TABLE foo (id Int64) ENGINE = MergeTree()",
			want: "CREATE TABLE foo (id Int64) ENGINE = MergeTree()",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, _, err := CleanDDLForBackup(tc.in)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, out)
		})
	}
}

func TestCleanDDLForBackup_StripsUUIDClause(t *testing.T) {
	in := "CREATE TABLE foo UUID '01234567-89ab-cdef-0123-456789abcdef' (id Int64) ENGINE = MergeTree() ORDER BY id"
	want := "CREATE TABLE foo (id Int64) ENGINE = MergeTree() ORDER BY id"

	out, warnings, err := CleanDDLForBackup(in)
	assert.NoError(t, err)
	assert.Equal(t, want, out)
	assert.Contains(t, warnings, "stripped UUID clause")
}

func TestCleanDDLForBackup_PreservesUUIDColumnType(t *testing.T) {
	// A column named with type UUID must NOT be touched — only the
	// statement-level UUID '...' clause is stripped.
	in := "CREATE TABLE foo (id UUID, name String) ENGINE = MergeTree() ORDER BY id"
	out, _, err := CleanDDLForBackup(in)
	assert.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestCleanDDLForBackup_PreservesReplicatedEngineVerbatim(t *testing.T) {
	in := "CREATE TABLE foo (id Int64) ENGINE = ReplicatedReplacingMergeTree('/clickhouse/tables/{shard}/foo', '{replica}', ver) ORDER BY id"
	out, _, err := CleanDDLForBackup(in)
	assert.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestCleanDDLForBackup_PreservesNestedTupleAndMapTypes(t *testing.T) {
	in := "CREATE TABLE foo (data Tuple(a Int, b Map(String, Array(Int)))) ENGINE = MergeTree() ORDER BY tuple()"
	out, _, err := CleanDDLForBackup(in)
	assert.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestRewriteEngineForRestore_KeepFlagFalse_ConvertsReplicatedReplacingMergeTree_PreservesVerArg(t *testing.T) {
	in := "CREATE TABLE foo (id Int64, ver UInt64) ENGINE = ReplicatedReplacingMergeTree('/p', '{r}', ver) ORDER BY id"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	assert.Contains(t, out, "ENGINE = ReplacingMergeTree(ver)")
	assert.NotContains(t, out, "/p")
	assert.NotContains(t, out, "{r}")
}

func TestRewriteEngineForRestore_KeepFlagFalse_ConvertsReplicatedCollapsingMergeTree_PreservesSign(t *testing.T) {
	in := "CREATE TABLE foo (id Int64, sign Int8) ENGINE = ReplicatedCollapsingMergeTree('/p', '{r}', sign) ORDER BY id"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	assert.Contains(t, out, "ENGINE = CollapsingMergeTree(sign)")
}

func TestRewriteEngineForRestore_KeepFlagFalse_ConvertsVersionedCollapsing_PreservesSignAndVer(t *testing.T) {
	in := "CREATE TABLE foo (id Int64, sign Int8, ver UInt64) ENGINE = ReplicatedVersionedCollapsingMergeTree('/p', '{r}', sign, ver) ORDER BY id"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	assert.Contains(t, out, "ENGINE = VersionedCollapsingMergeTree(sign, ver)")
}

func TestRewriteEngineForRestore_KeepFlagFalse_ConvertsSummingMergeTree_PreservesColumnsList(t *testing.T) {
	in := "CREATE TABLE foo (id Int64, a UInt64, b UInt64) ENGINE = ReplicatedSummingMergeTree('/p', '{r}', (a, b)) ORDER BY id"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	assert.Contains(t, out, "ENGINE = SummingMergeTree((a, b))")
}

func TestRewriteEngineForRestore_KeepFlagFalse_ConvertsGraphiteMergeTree_PreservesConfigSection(t *testing.T) {
	in := "CREATE TABLE foo (id Int64) ENGINE = ReplicatedGraphiteMergeTree('/p', '{r}', 'graphite_rollup') ORDER BY id"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	assert.Contains(t, out, "ENGINE = GraphiteMergeTree('graphite_rollup')")
}

func TestRewriteEngineForRestore_KeepFlagFalse_ConvertsPlainReplicatedMergeTree_NoArgs(t *testing.T) {
	in := "CREATE TABLE foo (id Int64) ENGINE = ReplicatedMergeTree('/p', '{r}') ORDER BY id"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	// Both replication args dropped, no engine-specific tail, so empty parens.
	assert.Contains(t, out, "ENGINE = MergeTree()")
}

func TestRewriteEngineForRestore_KeepFlagFalse_ConvertsSharedMergeTree_StripsSharedPrefix(t *testing.T) {
	in := "CREATE TABLE foo (id Int64) ENGINE = SharedMergeTree('/p', '{r}') ORDER BY id"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	assert.Contains(t, out, "ENGINE = MergeTree")
	assert.NotContains(t, out, "Shared")
}

func TestRewriteEngineForRestore_KeepFlagFalse_RewritesCreateDatabaseReplicatedToAtomic(t *testing.T) {
	in := "CREATE DATABASE my_db ENGINE = Replicated('/clickhouse/databases/my_db', 'shard1', 'replica1')"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	assert.Contains(t, out, "ENGINE = Atomic")
	assert.NotContains(t, out, "Replicated")
}

func TestRewriteEngineForRestore_KeepFlagTrue_PreservesReplicatedEngine(t *testing.T) {
	in := "CREATE TABLE foo (id Int64, ver UInt64) ENGINE = ReplicatedReplacingMergeTree('/p', '{r}', ver) ORDER BY id"
	out, _, err := RewriteEngineForRestore(in, true)
	assert.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestRewriteEngineForRestore_HandlesNestedTupleAndMapInColumns(t *testing.T) {
	in := "CREATE TABLE foo (data Tuple(a Int, b Map(String, Array(Int)))) ENGINE = ReplicatedMergeTree('/p', '{r}') ORDER BY tuple()"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	// Engine got rewritten correctly without confusion from nested type parens.
	assert.Contains(t, out, "ENGINE = MergeTree()")
	// Nested column types preserved verbatim.
	assert.Contains(t, out, "Tuple(a Int, b Map(String, Array(Int)))")
}

func TestRewriteEngineForRestore_NoEngineClause_PassthroughCleanly(t *testing.T) {
	// Some CREATE DATABASE forms have no ENGINE clause (CH defaults to Atomic).
	in := "CREATE DATABASE my_db"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestRewriteIdentifiersForRestore_BackquotedQualifier(t *testing.T) {
	in := "CREATE TABLE `prod_app`.`users` (id Int64) ENGINE = MergeTree() ORDER BY id"
	out, err := RewriteIdentifiersForRestore(in, "prod_app", "app_test")
	assert.NoError(t, err)
	assert.Contains(t, out, "`app_test`.`users`")
	assert.NotContains(t, out, "prod_app")
}

func TestRewriteIdentifiersForRestore_BareQualifier(t *testing.T) {
	in := "CREATE MATERIALIZED VIEW prod_app.daily_agg TO prod_app.daily_summary AS SELECT * FROM prod_app.events"
	out, err := RewriteIdentifiersForRestore(in, "prod_app", "app_test")
	assert.NoError(t, err)
	assert.Contains(t, out, "`app_test`.daily_agg")
	assert.Contains(t, out, "`app_test`.daily_summary")
	assert.Contains(t, out, "`app_test`.events")
	assert.NotContains(t, out, "prod_app.")
}

func TestRewriteIdentifiersForRestore_SkipsInsideStringLiteral(t *testing.T) {
	in := "CREATE MATERIALIZED VIEW v TO target AS SELECT * FROM events WHERE database = 'prod_app'"
	out, err := RewriteIdentifiersForRestore(in, "prod_app", "app_test")
	assert.NoError(t, err)
	// String literal preserved verbatim.
	assert.Contains(t, out, "'prod_app'")
}

func TestRewriteIdentifiersForRestore_DoesNotTouchSimilarTokensWithoutQualifier(t *testing.T) {
	// 'prod_app' as a column alias or comment-substring must NOT be rewritten.
	in := "SELECT prod_app_score FROM other_db.users"
	out, err := RewriteIdentifiersForRestore(in, "prod_app", "app_test")
	assert.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestRewriteIdentifiersForRestore_ErrorsOnEmptyArgs(t *testing.T) {
	_, err := RewriteIdentifiersForRestore("anything", "", "target")
	assert.Error(t, err)
	_, err = RewriteIdentifiersForRestore("anything", "source", "")
	assert.Error(t, err)
}

func TestRewriteIdentifiersForRestore_SourceEqualsTarget_NoOp(t *testing.T) {
	in := "CREATE TABLE prod_app.users (id Int64) ENGINE = MergeTree() ORDER BY id"
	out, err := RewriteIdentifiersForRestore(in, "prod_app", "prod_app")
	assert.NoError(t, err)
	assert.Equal(t, in, out)
}

// quoteIdent is exercised indirectly above; this confirms backtick escaping.
func TestQuoteIdent_EscapesBackticks(t *testing.T) {
	got := quoteIdent("weird`name")
	assert.Equal(t, "`weird``name`", got)
}

// indexOfKeyword is the foundation of every transform — explicit boundary tests.
func TestIndexOfKeyword_BoundaryRequirements(t *testing.T) {
	// Substring inside a longer identifier must not match.
	assert.Equal(t, -1, indexOfKeyword("UUIDValue String", "UUID", 0))
	assert.Equal(t, 0, indexOfKeyword("UUID '...'", "UUID", 0))
	// Case-insensitive.
	assert.Equal(t, 0, indexOfKeyword("uuid '...'", "UUID", 0))
	// Inside a string literal must not match.
	idx := indexOfKeyword("some 'UUID inside string' UUID 'outside'", "UUID", 0)
	assert.True(t, idx > 25, "expected UUID match outside the string, got %d", idx)
}

func TestFindMatchingParen_HandlesNestedAndStrings(t *testing.T) {
	in := "(a, (b, 'c)d'), e)"
	end, err := findMatchingParen(in, 0)
	assert.NoError(t, err)
	assert.Equal(t, len(in)-1, end)
}

func TestFindMatchingParen_UnbalancedReturnsError(t *testing.T) {
	_, err := findMatchingParen("(a, b", 0)
	assert.Error(t, err)
}

func TestTopLevelCommas_IgnoresNestedAndStrings(t *testing.T) {
	in := "a, (b, c), 'd, e', f"
	commas, err := topLevelCommas(in)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(commas), "got commas at %v", commas)
	for _, c := range commas {
		// Sanity: each top-level comma is at depth 0.
		assert.Equal(t, byte(','), in[c])
	}
}

// Round-trip: clean → engine rewrite → identifier rewrite, in the order
// the actual restore pipeline calls them.
func TestEndToEnd_BackupCleanThenRestoreEngineThenIdentifiers(t *testing.T) {
	original := "CREATE TABLE `prod_app`.`events` UUID 'aaaa-bbbb' ON CLUSTER my_cluster (id Int64, sign Int8) ENGINE = ReplicatedCollapsingMergeTree('/clickhouse/tables/{shard}/events', '{replica}', sign) ORDER BY id"

	cleaned, _, err := CleanDDLForBackup(original)
	assert.NoError(t, err)
	assert.NotContains(t, cleaned, "ON CLUSTER")
	assert.NotContains(t, cleaned, "UUID 'aaaa-bbbb'")
	// Engine still Replicated at this stage.
	assert.Contains(t, cleaned, "ReplicatedCollapsingMergeTree")

	engineRewritten, _, err := RewriteEngineForRestore(cleaned, false)
	assert.NoError(t, err)
	assert.Contains(t, engineRewritten, "ENGINE = CollapsingMergeTree(sign)")
	assert.NotContains(t, engineRewritten, "Replicated")

	final, err := RewriteIdentifiersForRestore(engineRewritten, "prod_app", "app_test")
	assert.NoError(t, err)
	assert.Contains(t, final, "`app_test`.`events`")
	assert.NotContains(t, final, "prod_app")
}

// Sanity guard against a regression where stripUUID would loop forever or
// strip non-clause occurrences.
func TestCleanDDLForBackup_TerminatesOnPathologicalInput(t *testing.T) {
	in := strings.Repeat("UUID ", 50) + "'..."
	out, _, err := CleanDDLForBackup(in)
	assert.NoError(t, err)
	// We don't care about the exact output — only that it terminated and produced something.
	assert.NotEmpty(t, out)
}

// Regression: a column named `engine` must not be confused with the engine
// clause keyword. Before this fix, RewriteEngineForRestore returned the input
// unchanged, leaving Replicated*MergeTree DDL in place.
func TestRewriteEngineForRestore_SkipsColumnNamedEngineBackquoted(t *testing.T) {
	in := "CREATE TABLE foo (`engine` String, ver UInt64) ENGINE = ReplicatedReplacingMergeTree('/p', '{r}', ver) ORDER BY ver"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	assert.Contains(t, out, "ENGINE = ReplacingMergeTree(ver)")
	assert.NotContains(t, out, "Replicated")
	// Column name preserved.
	assert.Contains(t, out, "`engine`")
}

func TestRewriteEngineForRestore_SkipsColumnNamedEngineBare(t *testing.T) {
	// CH allows bare 'engine' as an identifier in some contexts; even so, it
	// is inside the column-list paren group and must not be confused with the
	// statement-level ENGINE keyword.
	in := "CREATE TABLE foo (engine_kind String, ver UInt64) ENGINE = ReplicatedMergeTree('/p', '{r}') ORDER BY ver"
	out, _, err := RewriteEngineForRestore(in, false)
	assert.NoError(t, err)
	assert.Contains(t, out, "ENGINE = MergeTree()")
	assert.Contains(t, out, "engine_kind String")
}

func TestIndexOfKeyword_SkipsBackquotedIdentifiers(t *testing.T) {
	// Substring inside a backquoted identifier must not match.
	assert.Equal(t, -1, indexOfKeyword("(`engine` String)", "ENGINE", 0))
	// Real keyword after the backquoted column matches.
	assert.True(t, indexOfKeyword("(`engine` String) ENGINE = MergeTree()", "ENGINE", 0) > 17)
}

func TestIndexOfKeyword_SkipsDoubleQuotedIdentifiers(t *testing.T) {
	assert.Equal(t, -1, indexOfKeyword(`("engine" String)`, "ENGINE", 0))
	assert.True(t, indexOfKeyword(`("engine" String) ENGINE = MergeTree()`, "ENGINE", 0) > 17)
}

func TestParseMVToTarget_PlainToForm(t *testing.T) {
	in := "CREATE MATERIALIZED VIEW db.mv TO db.target AS SELECT * FROM db.events"
	target, ok := parseMVToTarget(in)
	assert.True(t, ok)
	assert.Equal(t, "db.target", target)
}

func TestParseMVToTarget_WithUUIDBeforeTo(t *testing.T) {
	in := "CREATE MATERIALIZED VIEW db.mv UUID '01234567-89ab-cdef-0123-456789abcdef' TO db.target AS SELECT * FROM db.events"
	target, ok := parseMVToTarget(in)
	assert.True(t, ok)
	assert.Equal(t, "db.target", target)
}

func TestParseMVToTarget_WithOnClusterBeforeTo(t *testing.T) {
	in := "CREATE MATERIALIZED VIEW db.mv ON CLUSTER my_cluster TO db.target AS SELECT * FROM db.events"
	target, ok := parseMVToTarget(in)
	assert.True(t, ok)
	assert.Equal(t, "db.target", target)
}

func TestParseMVToTarget_WithUUIDAndOnClusterBeforeTo(t *testing.T) {
	in := "CREATE MATERIALIZED VIEW db.mv UUID '0' ON CLUSTER cl TO db.target AS SELECT * FROM db.events"
	target, ok := parseMVToTarget(in)
	assert.True(t, ok)
	assert.Equal(t, "db.target", target)
}

func TestParseMVToTarget_ImplicitStorageRejected(t *testing.T) {
	cases := []string{
		"CREATE MATERIALIZED VIEW db.mv (cols Int64) ENGINE = MergeTree() ORDER BY cols AS SELECT * FROM db.events",
		"CREATE MATERIALIZED VIEW db.mv ENGINE = SummingMergeTree() AS SELECT * FROM db.events",
		"CREATE MATERIALIZED VIEW db.mv UUID '0' ENGINE = MergeTree() AS SELECT * FROM db.events",
	}
	for _, in := range cases {
		_, ok := parseMVToTarget(in)
		assert.False(t, ok, "expected implicit-storage rejection for: %s", in)
	}
}
