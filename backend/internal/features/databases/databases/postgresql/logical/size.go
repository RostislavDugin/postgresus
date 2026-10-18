package postgresql_logical

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Materialized views are left out on purpose: pg_dump emits their definition but
// not their contents, so counting them would overstate what a restore recreates.
const dumpedRelationsSizeQuery = `
	SELECT n.nspname,
	       c.relname,
	       pg_catalog.pg_table_is_visible(c.oid),
	       COALESCE(pg_total_relation_size(c.oid), 0)
` + dumpedRelationScope + `
	  AND c.relkind IN ('r', 'p')
`

func getDumpedRelationsSizeBytes(
	ctx context.Context,
	conn *pgx.Conn,
	filter DumpFilter,
) (int64, error) {
	rows, err := queryDumpedRelations(ctx, conn, dumpedRelationsSizeQuery, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to query dumped relations size: %w", err)
	}

	defer rows.Close()

	excludeTablePatterns := filter.getExcludeTablePatterns()
	excludeTableDataPatterns := filter.getExcludeTableDataPatterns()

	var totalSizeBytes int64

	for rows.Next() {
		var relation relationName
		var relationSizeBytes int64

		if err := rows.Scan(
			&relation.SchemaName,
			&relation.TableName,
			&relation.IsVisibleOnSearchPath,
			&relationSizeBytes,
		); err != nil {
			return 0, fmt.Errorf("failed to scan dumped relation size: %w", err)
		}

		if isRelationExcluded(relation, excludeTablePatterns) ||
			isRelationExcluded(relation, excludeTableDataPatterns) {
			continue
		}

		totalSizeBytes += relationSizeBytes
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("failed to read dumped relations size: %w", err)
	}

	return totalSizeBytes, nil
}
