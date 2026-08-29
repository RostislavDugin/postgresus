package postgresql_logical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetIncludedSchemaNames_WhenNamesArePastedMultiline_ReturnsTrimmedUniqueNames(t *testing.T) {
	filter := DumpFilter{IncludeSchemas: []string{"\npublic", " reporting ", "public"}}

	assert.Equal(t, []string{"public", "reporting"}, filter.getIncludedSchemaNames())
}

func Test_GetIncludedSchemaNames_WhenNoSchemasAreConfigured_ReturnsEmptySlice(t *testing.T) {
	filter := DumpFilter{}

	assert.Empty(t, filter.getIncludedSchemaNames())
	assert.NotNil(t, filter.getIncludedSchemaNames())
}

func Test_IsRelationExcluded_WithVariousPatterns_MatchesPgDumpSemantics(t *testing.T) {
	exclusionCases := []struct {
		name                 string
		relation             relationName
		excludeTablePatterns []string
		isExcluded           bool
	}{
		{
			name:                 "unqualified name matches table in any schema",
			relation:             relationName{SchemaName: "reporting", TableName: "audit_logs"},
			excludeTablePatterns: []string{"audit_logs"},
			isExcluded:           true,
		},
		{
			name:                 "unqualified name does not match a different table",
			relation:             relationName{SchemaName: "public", TableName: "orders"},
			excludeTablePatterns: []string{"audit_logs"},
			isExcluded:           false,
		},
		{
			name:                 "schema qualified name matches only its own schema",
			relation:             relationName{SchemaName: "public", TableName: "audit_logs"},
			excludeTablePatterns: []string{"public.audit_logs"},
			isExcluded:           true,
		},
		{
			name:                 "schema qualified name skips the same table in another schema",
			relation:             relationName{SchemaName: "reporting", TableName: "audit_logs"},
			excludeTablePatterns: []string{"public.audit_logs"},
			isExcluded:           false,
		},
		{
			name:                 "star glob matches a name prefix",
			relation:             relationName{SchemaName: "public", TableName: "logs_2026_07"},
			excludeTablePatterns: []string{"logs_*"},
			isExcluded:           true,
		},
		{
			name:                 "star glob does not cross the schema separator",
			relation:             relationName{SchemaName: "reporting", TableName: "logs_2026_07"},
			excludeTablePatterns: []string{"public.*"},
			isExcluded:           false,
		},
		{
			name:                 "star glob on the schema part matches every table there",
			relation:             relationName{SchemaName: "reporting", TableName: "orders"},
			excludeTablePatterns: []string{"report*.orders"},
			isExcluded:           true,
		},
		{
			name:                 "question mark glob matches a single character",
			relation:             relationName{SchemaName: "public", TableName: "shard_7"},
			excludeTablePatterns: []string{"shard_?"},
			isExcluded:           true,
		},
		{
			name:                 "question mark glob does not match two characters",
			relation:             relationName{SchemaName: "public", TableName: "shard_42"},
			excludeTablePatterns: []string{"shard_?"},
			isExcluded:           false,
		},
		{
			name:                 "character class glob matches a listed character",
			relation:             relationName{SchemaName: "public", TableName: "shard_b"},
			excludeTablePatterns: []string{"shard_[ab]"},
			isExcluded:           true,
		},
		{
			name:                 "quoted part is compared literally including the dot",
			relation:             relationName{SchemaName: "public", TableName: "weird.name"},
			excludeTablePatterns: []string{`"weird.name"`},
			isExcluded:           true,
		},
		{
			name:                 "quoted part does not glob",
			relation:             relationName{SchemaName: "public", TableName: "logs_2026"},
			excludeTablePatterns: []string{`"logs_*"`},
			isExcluded:           false,
		},
		{
			name:                 "quoted schema with unquoted table glob",
			relation:             relationName{SchemaName: "My Schema", TableName: "logs_2026"},
			excludeTablePatterns: []string{`"My Schema".logs_*`},
			isExcluded:           true,
		},
		{
			name:                 "malformed glob falls back to literal comparison",
			relation:             relationName{SchemaName: "public", TableName: "shard_[ab"},
			excludeTablePatterns: []string{"shard_[ab"},
			isExcluded:           true,
		},
		{
			name:                 "malformed glob does not match a different name",
			relation:             relationName{SchemaName: "public", TableName: "shard_a"},
			excludeTablePatterns: []string{"shard_[ab"},
			isExcluded:           false,
		},
		{
			name:                 "empty pattern list excludes nothing",
			relation:             relationName{SchemaName: "public", TableName: "orders"},
			excludeTablePatterns: []string{},
			isExcluded:           false,
		},
		{
			name:                 "any matching pattern in the list wins",
			relation:             relationName{SchemaName: "public", TableName: "orders"},
			excludeTablePatterns: []string{"audit_logs", "public.orders"},
			isExcluded:           true,
		},
	}

	for _, exclusionCase := range exclusionCases {
		t.Run(exclusionCase.name, func(t *testing.T) {
			isExcluded := isRelationExcluded(
				exclusionCase.relation,
				exclusionCase.excludeTablePatterns,
			)

			assert.Equal(t, exclusionCase.isExcluded, isExcluded)
		})
	}
}
