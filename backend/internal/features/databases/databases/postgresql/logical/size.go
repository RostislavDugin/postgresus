package postgresql_logical

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Materialized views are left out on purpose: pg_dump emits their definition but
// not their contents, so counting them would overstate what a restore recreates.
const dumpedRelationsSizeQuery = `
	SELECT n.nspname, c.relname, COALESCE(pg_total_relation_size(c.oid), 0)
	FROM pg_class c
	JOIN pg_namespace n ON n.oid = c.relnamespace
	WHERE c.relkind IN ('r', 'p')
	  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
	  AND n.nspname NOT LIKE 'pg\_toast%'
	  AND n.nspname NOT LIKE 'pg\_temp%'
	  AND (cardinality($1::text[]) = 0 OR n.nspname = ANY($1::text[]))
`

type DumpFilter struct {
	IncludeSchemas       []string
	ExcludeTablePatterns []string
}

type relationName struct {
	SchemaName string
	TableName  string
}

func getDumpedRelationsSizeBytes(
	ctx context.Context,
	conn *pgx.Conn,
	filter DumpFilter,
) (int64, error) {
	includeSchemas := filter.IncludeSchemas
	if includeSchemas == nil {
		includeSchemas = []string{}
	}

	rows, err := conn.Query(ctx, dumpedRelationsSizeQuery, includeSchemas)
	if err != nil {
		return 0, fmt.Errorf("failed to query dumped relations size: %w", err)
	}

	defer rows.Close()

	var totalSizeBytes int64

	for rows.Next() {
		var relation relationName
		var relationSizeBytes int64

		if err := rows.Scan(&relation.SchemaName, &relation.TableName, &relationSizeBytes); err != nil {
			return 0, fmt.Errorf("failed to scan dumped relation size: %w", err)
		}

		if isRelationExcluded(relation, filter.ExcludeTablePatterns) {
			continue
		}

		totalSizeBytes += relationSizeBytes
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("failed to read dumped relations size: %w", err)
	}

	return totalSizeBytes, nil
}

func isRelationExcluded(relation relationName, excludeTablePatterns []string) bool {
	for _, excludeTablePattern := range excludeTablePatterns {
		if parseDumpTablePattern(excludeTablePattern).isMatchingRelation(relation) {
			return true
		}
	}

	return false
}

type dumpTablePattern struct {
	SchemaGlob        globPattern
	TableGlob         globPattern
	IsSchemaQualified bool
}

func (p dumpTablePattern) isMatchingRelation(relation relationName) bool {
	if p.IsSchemaQualified && !p.SchemaGlob.isMatchingName(relation.SchemaName) {
		return false
	}

	return p.TableGlob.isMatchingName(relation.TableName)
}

// Mirrors pg_dump's --exclude-table: an unqualified pattern hits the table name in
// every schema, and the split happens on the first dot outside double quotes.
func parseDumpTablePattern(excludeTablePattern string) dumpTablePattern {
	isInsideQuotes := false

	for index, character := range excludeTablePattern {
		switch character {
		case '"':
			isInsideQuotes = !isInsideQuotes
		case '.':
			if !isInsideQuotes {
				return dumpTablePattern{
					SchemaGlob:        parseGlobPattern(excludeTablePattern[:index]),
					TableGlob:         parseGlobPattern(excludeTablePattern[index+1:]),
					IsSchemaQualified: true,
				}
			}
		}
	}

	return dumpTablePattern{TableGlob: parseGlobPattern(excludeTablePattern)}
}

type globPattern struct {
	Value     string
	IsLiteral bool
}

func parseGlobPattern(patternPart string) globPattern {
	if len(patternPart) >= 2 &&
		strings.HasPrefix(patternPart, `"`) &&
		strings.HasSuffix(patternPart, `"`) {
		unquoted := strings.ReplaceAll(patternPart[1:len(patternPart)-1], `""`, `"`)

		return globPattern{Value: unquoted, IsLiteral: true}
	}

	return globPattern{Value: patternPart}
}

func (g globPattern) isMatchingName(name string) bool {
	if g.IsLiteral {
		return g.Value == name
	}

	isMatching, err := path.Match(g.Value, name)
	if err != nil {
		return g.Value == name
	}

	return isMatching
}
