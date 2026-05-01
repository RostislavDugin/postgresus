package clickhouse

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	chmanifest "databasus-backend/internal/features/backups/backups/clickhouse_manifest"
)

// supportedEngineRegex matches every *MergeTree-family engine name we are
// willing to back up data for, including Replicated* and Shared* variants.
// All other engines (Distributed, Kafka, S3, URL, MySQL, PostgreSQL, MongoDB,
// Buffer, Null, Memory, File, Log, TinyLog, StripeLog, Dictionary,
// EmbeddedRocksDB, etc.) are rejected at pre-flight: their data either lives
// elsewhere or is not durable.
var supportedEngineRegex = regexp.MustCompile(
	`^(Replicated|Shared)?` +
		`(MergeTree|ReplacingMergeTree|SummingMergeTree|AggregatingMergeTree|` +
		`CollapsingMergeTree|VersionedCollapsingMergeTree|GraphiteMergeTree)$`,
)

// unsupportedTypeRegex matches column types that are still on-disk-evolving in
// modern CH releases. We refuse to round-trip them via Native rather than risk
// silent corruption when source and restore servers differ in version.
var unsupportedTypeRegex = regexp.MustCompile(
	`(?i)(\bVariant\b|\bDynamic\b|\bJSON\b|\bObject\s*\(\s*'json'\s*\))`,
)

// EnumerateAndValidate inspects system.tables/columns/MVs for the given
// database and returns the planned backup work, or an error describing every
// reason the database cannot be backed up. Rejections are aggregated so the
// user sees the full picture rather than one issue at a time.
func EnumerateAndValidate(
	ctx context.Context,
	conn driver.Conn,
	database string,
) ([]chmanifest.TableHeaderEntry, []chmanifest.MVHeaderEntry, error) {
	if err := rejectRegularViews(ctx, conn, database); err != nil {
		return nil, nil, err
	}

	rows, err := conn.Query(ctx, `
		SELECT name, engine, engine_full, create_table_query
		FROM system.tables
		WHERE database = ?
		  AND name NOT LIKE '.inner.%'
		  AND name NOT LIKE '.inner_id.%'
		ORDER BY name
	`, database)
	if err != nil {
		return nil, nil, fmt.Errorf("query system.tables: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var (
		tables           []chmanifest.TableHeaderEntry
		mvs              []chmanifest.MVHeaderEntry
		engineRejections []string
		mvRejections     []string
	)

	for rows.Next() {
		var name, engine, engineFull, createQuery string
		if err := rows.Scan(&name, &engine, &engineFull, &createQuery); err != nil {
			return nil, nil, fmt.Errorf("scan system.tables row: %w", err)
		}

		if engine == "MaterializedView" {
			toTable, isToForm := parseMVToTarget(createQuery)
			if !isToForm {
				mvRejections = append(mvRejections, fmt.Sprintf("%s.%s (implicit storage)", database, name))
				continue
			}
			mvs = append(mvs, chmanifest.MVHeaderEntry{
				ID:       opaqueID("mv-", name),
				Database: database,
				Name:     name,
				ToTable:  toTable,
				DDLHash:  sha256Hex([]byte(createQuery)),
			})
			continue
		}

		if !supportedEngineRegex.MatchString(engine) {
			engineRejections = append(engineRejections, fmt.Sprintf("%s.%s (engine: %s)", database, name, engine))
			continue
		}

		tables = append(tables, chmanifest.TableHeaderEntry{
			ID:       opaqueID("t-", name),
			Database: database,
			Name:     name,
			Engine:   engine,
			IsMV:     false,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate system.tables: %w", err)
	}

	if len(engineRejections) > 0 {
		return nil, nil, fmt.Errorf(
			"unsupported table engines (only *MergeTree family is supported in v1): %s",
			strings.Join(engineRejections, ", "),
		)
	}
	if len(mvRejections) > 0 {
		return nil, nil, fmt.Errorf(
			"implicit-storage materialized views are not supported in v1; "+
				"recreate as `TO <target>` form, exclude these tables, or wait for v2: %s",
			strings.Join(mvRejections, ", "),
		)
	}

	for i := range tables {
		cols, err := loadPhysicalColumns(ctx, conn, database, tables[i].Name)
		if err != nil {
			return nil, nil, err
		}
		tables[i].Columns = cols
	}

	return tables, mvs, nil
}

// RejectUnsupportedTypes scans system.columns for any Variant / Dynamic / JSON
// usage (including nested forms like Array(Variant), Nullable(JSON), legacy
// Object('json')) and returns an error listing every offender, or nil.
func RejectUnsupportedTypes(ctx context.Context, conn driver.Conn, database string) error {
	rows, err := conn.Query(ctx, `
		SELECT table, name, type
		FROM system.columns
		WHERE database = ?
		ORDER BY table, position
	`, database)
	if err != nil {
		return fmt.Errorf("query system.columns: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var offenders []string
	for rows.Next() {
		var table, name, colType string
		if err := rows.Scan(&table, &name, &colType); err != nil {
			return fmt.Errorf("scan system.columns row: %w", err)
		}
		if unsupportedTypeRegex.MatchString(colType) {
			offenders = append(offenders, fmt.Sprintf("%s.%s.%s (%s)", database, table, name, colType))
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate system.columns: %w", err)
	}

	if len(offenders) > 0 {
		return fmt.Errorf(
			"unsupported column types (Variant/Dynamic/JSON are not supported in v1): %s",
			strings.Join(offenders, ", "),
		)
	}
	return nil
}

// GetDatabaseDDL returns the source database's engine name and full
// `CREATE DATABASE ... ENGINE = ...` DDL.
func GetDatabaseDDL(ctx context.Context, conn driver.Conn, database string) (engine, ddl string, err error) {
	if err := conn.QueryRow(ctx,
		`SELECT engine FROM system.databases WHERE name = ?`,
		database,
	).Scan(&engine); err != nil {
		return "", "", fmt.Errorf("query database engine: %w", err)
	}

	if err := conn.QueryRow(ctx,
		fmt.Sprintf("SHOW CREATE DATABASE `%s`", strings.ReplaceAll(database, "`", "``")),
	).Scan(&ddl); err != nil {
		return "", "", fmt.Errorf("show create database: %w", err)
	}

	return engine, ddl, nil
}

// GetTableDDL returns the verbatim `SHOW CREATE TABLE` output for one table.
func GetTableDDL(ctx context.Context, conn driver.Conn, database, table string) (string, error) {
	q := fmt.Sprintf(
		"SHOW CREATE TABLE `%s`.`%s`",
		strings.ReplaceAll(database, "`", "``"),
		strings.ReplaceAll(table, "`", "``"),
	)

	var ddl string
	if err := conn.QueryRow(ctx, q).Scan(&ddl); err != nil {
		return "", fmt.Errorf("show create table %s.%s: %w", database, table, err)
	}
	return ddl, nil
}

// --- Internals ---

func rejectRegularViews(ctx context.Context, conn driver.Conn, database string) error {
	rows, err := conn.Query(ctx, `
		SELECT name FROM system.tables
		WHERE database = ? AND engine = 'View'
		ORDER BY name
	`, database)
	if err != nil {
		return fmt.Errorf("query views: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scan view row: %w", err)
		}
		names = append(names, fmt.Sprintf("%s.%s", database, name))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate views: %w", err)
	}

	if len(names) > 0 {
		return fmt.Errorf(
			"regular Views are not supported in v1: %s. "+
				"Either remove them or recreate manually post-restore.",
			strings.Join(names, ", "),
		)
	}
	return nil
}

// parseMVToTarget extracts the `TO <db>.<tbl>` target of a CREATE MATERIALIZED
// VIEW statement. Returns ("", false) for implicit-storage MVs (no TO clause).
//
// Examples (whitespace normalised):
//   CREATE MATERIALIZED VIEW db.mv TO db.target AS SELECT ...
//   CREATE MATERIALIZED VIEW db.mv UUID '...' TO db.target AS SELECT ...
//   CREATE MATERIALIZED VIEW db.mv ON CLUSTER cl TO db.target AS SELECT ...
//   CREATE MATERIALIZED VIEW db.mv (cols) ENGINE = X(...) AS SELECT ...   ← implicit
func parseMVToTarget(createQuery string) (string, bool) {
	mvIdx := indexOfKeyword(createQuery, "MATERIALIZED VIEW", 0)
	if mvIdx < 0 {
		return "", false
	}

	cursor := mvIdx + len("MATERIALIZED VIEW")
	cursor = skipSpaces(createQuery, cursor)
	cursor = skipQualifiedIdentifier(createQuery, cursor)
	cursor = skipSpaces(createQuery, cursor)

	// Atomic databases emit `UUID '<uuid>'` after the MV name; replicated
	// schemas may also include `ON CLUSTER <name>`. Walk past any number of
	// these clauses (in either order) before checking for TO. The presence of
	// either says nothing about whether the MV is TO-form vs implicit.
	for {
		moved := false
		if hasKeywordAt(createQuery, cursor, "UUID") {
			next := skipSpaces(createQuery, cursor+len("UUID"))
			if next < len(createQuery) && createQuery[next] == '\'' {
				cursor = skipSingleQuotedString(createQuery, next)
				cursor = skipSpaces(createQuery, cursor)
				moved = true
			}
		}
		if hasKeywordAt(createQuery, cursor, "ON CLUSTER") {
			cursor += len("ON CLUSTER")
			cursor = skipSpaces(createQuery, cursor)
			cursor = skipOneIdentifier(createQuery, cursor)
			cursor = skipSpaces(createQuery, cursor)
			moved = true
		}
		if !moved {
			break
		}
	}

	// What follows is one of:
	//   TO <db>.<tbl> ...     ← TO-form
	//   ( cols )              ← implicit-storage column list
	//   ENGINE = ...          ← implicit-storage with explicit engine
	//   AS SELECT ...         ← bare implicit-storage variant
	if cursor >= len(createQuery) {
		return "", false
	}

	if !hasKeywordAt(createQuery, cursor, "TO") {
		return "", false
	}

	cursor += len("TO")
	cursor = skipSpaces(createQuery, cursor)
	target := readQualifiedIdentifier(createQuery, cursor)
	if target == "" {
		return "", false
	}
	return target, true
}

func skipQualifiedIdentifier(s string, pos int) int {
	pos = skipOneIdentifier(s, pos)
	if pos < len(s) && s[pos] == '.' {
		pos++
		pos = skipOneIdentifier(s, pos)
	}
	return pos
}

func skipOneIdentifier(s string, pos int) int {
	if pos >= len(s) {
		return pos
	}
	switch s[pos] {
	case '`':
		return skipBacktickIdentifier(s, pos)
	case '"':
		return skipDoubleQuotedIdentifier(s, pos)
	default:
		return scanIdentifier(s, pos)
	}
}

func readQualifiedIdentifier(s string, pos int) string {
	start := pos
	end := skipQualifiedIdentifier(s, pos)
	if end == start {
		return ""
	}
	return s[start:end]
}

func hasKeywordAt(s string, pos int, kw string) bool {
	if pos+len(kw) > len(s) {
		return false
	}
	if !strings.EqualFold(s[pos:pos+len(kw)], strings.ToUpper(kw)) {
		return false
	}
	if pos > 0 && isIdentChar(s[pos-1]) {
		return false
	}
	if pos+len(kw) < len(s) && isIdentChar(s[pos+len(kw)]) {
		return false
	}
	return true
}

// loadPhysicalColumns pulls the explicit column list — excluding MATERIALIZED
// and ALIAS columns — for use in `SELECT col1, col2, ... FORMAT Native` and
// the matching `INSERT INTO ... (col1, col2, ...) FORMAT Native`.
func loadPhysicalColumns(ctx context.Context, conn driver.Conn, database, table string) ([]string, error) {
	rows, err := conn.Query(ctx, `
		SELECT name
		FROM system.columns
		WHERE database = ?
		  AND table = ?
		  AND default_kind NOT IN ('MATERIALIZED', 'ALIAS')
		ORDER BY position
	`, database, table)
	if err != nil {
		return nil, fmt.Errorf("query columns of %s.%s: %w", database, table, err)
	}
	defer func() { _ = rows.Close() }()

	var cols []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan column row: %w", err)
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate columns: %w", err)
	}

	if len(cols) == 0 {
		return nil, fmt.Errorf("table %s.%s has no physical (non-MATERIALIZED, non-ALIAS) columns", database, table)
	}
	return cols, nil
}

// opaqueID produces a deterministic short identifier from a name, suitable for
// use in tar paths where the original name might be unsafe.
func opaqueID(prefix, name string) string {
	sum := sha256.Sum256([]byte(name))
	return prefix + hex.EncodeToString(sum[:4])
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

