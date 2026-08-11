package usecases_logical_mariadb

import (
	"slices"
	"strings"
	"testing"

	mariadbtypes "databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/util/tools"
)

// One INSERT per row costs ~127x on restore and saves no memory: with --quick the
// dumper streams rows and caps a batched statement at net_buffer_length (~1 MB)
// however large the table is (issue #630).
func Test_BuildMariadbDumpArgs_ForAnyDatabase_NeverSkipsExtendedInsert(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	database := &mariadbtypes.MariadbDatabase{
		Version:       tools.MariadbVersion1011,
		Database:      new("oa_db"),
		ExcludeTables: []string{"personnel_real_time"},
	}

	dumpArgs := uc.buildMariadbDumpArgs(database)

	if slices.Contains(dumpArgs, "--skip-extended-insert") {
		t.Fatalf("mariadb-dump args must never contain --skip-extended-insert: %v", dumpArgs)
	}
}

func Test_BuildMariadbDumpArgs_WithExcludedTables_AddsQualifiedIgnoreTableArgs(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	databaseName := "oa_db"
	database := &mariadbtypes.MariadbDatabase{
		Version:       tools.MariadbVersion114,
		Database:      &databaseName,
		ExcludeTables: []string{"personnel_access_control_event", "personnel_real_time"},
	}

	args := uc.buildMariadbDumpArgs(database)

	if !slices.Contains(args, "--ignore-table=oa_db.personnel_access_control_event") ||
		!slices.Contains(args, "--ignore-table=oa_db.personnel_real_time") {
		t.Fatalf("expected an --ignore-table arg per excluded table, got %v", args)
	}
}

func Test_BuildMariadbDumpArgs_WhenExcludedTablesArePastedMultiline_TrimsAndSplitsThem(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	databaseName := "oa_db"
	database := &mariadbtypes.MariadbDatabase{
		Version:  tools.MariadbVersion114,
		Database: &databaseName,
		ExcludeTables: []string{
			"personnel_access_control_event",
			"\npersonnel_real_time",
			" ",
			"ext_alarm_message,\nmonitor_toxic_gas",
		},
	}

	args := uc.buildMariadbDumpArgs(database)

	ignoredTableArgs := []string{
		"--ignore-table=oa_db.personnel_access_control_event",
		"--ignore-table=oa_db.personnel_real_time",
		"--ignore-table=oa_db.ext_alarm_message",
		"--ignore-table=oa_db.monitor_toxic_gas",
	}
	for _, ignoredTableArg := range ignoredTableArgs {
		if !slices.Contains(args, ignoredTableArg) {
			t.Fatalf("expected %s, got %v", ignoredTableArg, args)
		}
	}

	if slices.Contains(args, "--ignore-table=oa_db.") {
		t.Fatalf("expected blank excluded tables to be dropped, got %v", args)
	}
}

func Test_BuildMariadbDumpArgs_WithoutExcludedTables_OmitsIgnoreTableArgs(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	databaseName := "oa_db"
	database := &mariadbtypes.MariadbDatabase{
		Version:  tools.MariadbVersion114,
		Database: &databaseName,
	}

	args := uc.buildMariadbDumpArgs(database)

	for _, arg := range args {
		if strings.HasPrefix(arg, "--ignore-table=") {
			t.Fatalf("expected no --ignore-table args, got %v", args)
		}
	}
}
