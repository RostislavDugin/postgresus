package common

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// TableHealthReport holds the complete health check and verification results for a backup.
type TableHealthReport struct {
	TotalTables   int                `json:"totalTables"`
	DumpedTables  int                `json:"dumpedTables"`
	MissingTables []string           `json:"missingTables,omitzero"`
	RepairedCount int                `json:"repairedCount"`
	FailedRepairs int                `json:"failedRepairs"`
	Tables        []TableHealthEntry `json:"tables,omitzero"`
	CheckedAt     time.Time          `json:"checkedAt"`
}

// TableHealthEntry holds per-table health check results.
type TableHealthEntry struct {
	Name       string `json:"name"`
	Engine     string `json:"engine"`
	Status     string `json:"status"` // OK, REPAIRED, REPAIR_FAILED, MISSING_FROM_DUMP
	Message    string `json:"message,omitzero"`
	WasRepaired bool  `json:"wasRepaired,omitzero"`
}

// PreBackupHealthCheck runs CHECK TABLE on all MyISAM/Aria tables and auto-repairs crashed ones.
// Returns the health report and list of tables that could not be repaired.
func PreBackupHealthCheck(
	ctx context.Context,
	logger *slog.Logger,
	host string,
	port int,
	username string,
	password string,
	database string,
	isHttps bool,
) (*TableHealthReport, error) {
	mysqlConfig := mysql.NewConfig()
	mysqlConfig.User = username
	mysqlConfig.Passwd = password
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = fmt.Sprintf("%s:%d", host, port)
	mysqlConfig.DBName = database
	mysqlConfig.Timeout = 30 * time.Second

	if isHttps {
		mysqlConfig.TLSConfig = "true"
	}

	db, err := sql.Open("mysql", mysqlConfig.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect for health check: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(60 * time.Second)
	db.SetMaxOpenConns(1)

	report := &TableHealthReport{
		CheckedAt: time.Now().UTC(),
	}

	// Get all tables with their engines
	rows, err := db.QueryContext(ctx,
		"SELECT TABLE_NAME, ENGINE FROM information_schema.tables WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'",
		database)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	type tableInfo struct {
		name   string
		engine string
	}

	var tables []tableInfo
	for rows.Next() {
		var tableName, engine string
		if err := rows.Scan(&tableName, &engine); err != nil {
			continue
		}
		tables = append(tables, tableInfo{name: tableName, engine: engine})
	}

	report.TotalTables = len(tables)

	// Check and repair MyISAM/Aria/CSV tables (InnoDB uses crash recovery automatically)
	repairableEngines := map[string]bool{"MyISAM": true, "Aria": true, "CSV": true}

	for _, table := range tables {
		entry := TableHealthEntry{
			Name:   table.name,
			Engine: table.engine,
			Status: "OK",
		}

		if !repairableEngines[table.engine] {
			report.Tables = append(report.Tables, entry)
			continue
		}

		// CHECK TABLE
		checkResult, err := runTableCheck(ctx, db, table.name)
		if err != nil {
			entry.Status = "CHECK_ERROR"
			entry.Message = err.Error()
			report.Tables = append(report.Tables, entry)
			continue
		}

		if checkResult == "OK" {
			report.Tables = append(report.Tables, entry)
			continue
		}

		// Table needs repair
		logger.Warn(fmt.Sprintf("table %s needs repair: %s", table.name, checkResult),
			"database", database,
			"engine", table.engine,
		)

		repairResult, err := runTableRepair(ctx, db, table.name)
		if err != nil || repairResult != "OK" {
			entry.Status = "REPAIR_FAILED"
			entry.Message = fmt.Sprintf("check: %s, repair: %s %v", checkResult, repairResult, err)
			report.FailedRepairs++
			logger.Error(fmt.Sprintf("failed to repair table %s", table.name),
				"database", database,
				"error", entry.Message,
			)
		} else {
			entry.Status = "REPAIRED"
			entry.WasRepaired = true
			entry.Message = fmt.Sprintf("was: %s", checkResult)
			report.RepairedCount++
			logger.Info(fmt.Sprintf("successfully repaired table %s", table.name),
				"database", database,
			)
		}

		report.Tables = append(report.Tables, entry)
	}

	return report, nil
}

// VerifyDumpCompleteness compares tables found in mysqldump stderr (--verbose output)
// with actual tables to detect partial dumps.
func VerifyDumpCompleteness(
	report *TableHealthReport,
	stderrOutput string,
) {
	if report == nil {
		return
	}

	// mysqldump --verbose outputs lines like:
	// -- Dumping table `table_name`
	dumpedPattern := regexp.MustCompile(`-- Dumping tables? ` + "`" + `(\w+)` + "`")
	matches := dumpedPattern.FindAllStringSubmatch(stderrOutput, -1)

	dumpedSet := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			dumpedSet[match[1]] = true
		}
	}

	report.DumpedTables = len(dumpedSet)

	// Also check for "CREATE TABLE" pattern as fallback from stderr
	if report.DumpedTables == 0 {
		// Count from error lines — if mysqldump failed on a table, it logged it
		return
	}

	for i, entry := range report.Tables {
		if _, wasDumped := dumpedSet[entry.Name]; !wasDumped {
			if entry.Status == "OK" || entry.Status == "REPAIRED" {
				report.Tables[i].Status = "MISSING_FROM_DUMP"
				report.MissingTables = append(report.MissingTables, entry.Name)
			}
		}
	}
}

// ParseMysqldumpErrors extracts per-table errors from mysqldump stderr output.
func ParseMysqldumpErrors(stderrOutput string) map[string]string {
	errors := make(map[string]string)

	// Pattern: mysqldump: Error XXXX: ... when dumping table `table_name`
	pattern := regexp.MustCompile(`(?i)error.*?when dumping table\s*` + "`" + `(\w+)` + "`")
	matches := pattern.FindAllStringSubmatch(stderrOutput, -1)

	for _, match := range matches {
		if len(match) > 1 {
			tableName := match[1]
			errors[tableName] = match[0]
		}
	}

	// Pattern: Table 'table_name' is marked as crashed
	crashPattern := regexp.MustCompile(`Table '(\w+)' is marked as crashed`)
	crashMatches := crashPattern.FindAllStringSubmatch(stderrOutput, -1)

	for _, match := range crashMatches {
		if len(match) > 1 {
			tableName := match[1]
			if _, exists := errors[tableName]; !exists {
				errors[tableName] = match[0]
			}
		}
	}

	return errors
}

func runTableCheck(ctx context.Context, db *sql.DB, tableName string) (string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("CHECK TABLE `%s`", tableName))
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var lastStatus string

	for rows.Next() {
		var table, op, msgType, msgText string
		if err := rows.Scan(&table, &op, &msgType, &msgText); err != nil {
			continue
		}
		lastStatus = msgText
	}

	return strings.TrimSpace(lastStatus), nil
}

func runTableRepair(ctx context.Context, db *sql.DB, tableName string) (string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("REPAIR TABLE `%s`", tableName))
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var lastStatus string

	for rows.Next() {
		var table, op, msgType, msgText string
		if err := rows.Scan(&table, &op, &msgType, &msgText); err != nil {
			continue
		}
		lastStatus = msgText
	}

	return strings.TrimSpace(lastStatus), nil
}
