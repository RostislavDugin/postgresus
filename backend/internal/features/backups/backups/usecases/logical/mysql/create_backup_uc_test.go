package usecases_logical_mysql

import (
	"slices"
	"strings"
	"testing"

	mysqltypes "databasus-backend/internal/features/databases/databases/mysql"
	"databasus-backend/internal/util/tools"
)

func Test_BuildMysqldumpArgs_WhenExtendedInsertDisabled_SkipsExtendedInsert(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	database := &mysqltypes.MysqlDatabase{
		Version:             tools.MysqlVersion80,
		IsUseExtendedInsert: false,
	}

	args := uc.buildMysqldumpArgs(database)

	if !slices.Contains(args, "--skip-extended-insert") {
		t.Fatalf("expected --skip-extended-insert when extended inserts are disabled, got %v", args)
	}
}

func Test_BuildMysqldumpArgs_WhenExtendedInsertEnabled_OmitsSkipExtendedInsert(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	database := &mysqltypes.MysqlDatabase{
		Version:             tools.MysqlVersion80,
		IsUseExtendedInsert: true,
	}

	args := uc.buildMysqldumpArgs(database)

	if slices.Contains(args, "--skip-extended-insert") {
		t.Fatalf("expected no --skip-extended-insert when extended inserts are enabled, got %v", args)
	}
}

func Test_BuildMysqldumpArgs_WithExcludedTables_AddsQualifiedIgnoreTableArgs(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	databaseName := "oa_db"
	database := &mysqltypes.MysqlDatabase{
		Version:       tools.MysqlVersion80,
		Database:      &databaseName,
		ExcludeTables: []string{"personnel_access_control_event", "personnel_real_time"},
	}

	args := uc.buildMysqldumpArgs(database)

	if !slices.Contains(args, "--ignore-table=oa_db.personnel_access_control_event") ||
		!slices.Contains(args, "--ignore-table=oa_db.personnel_real_time") {
		t.Fatalf("expected an --ignore-table arg per excluded table, got %v", args)
	}
}

func Test_BuildMysqldumpArgs_WhenExcludedTablesArePastedMultiline_TrimsAndSplitsThem(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	databaseName := "oa_db"
	database := &mysqltypes.MysqlDatabase{
		Version:  tools.MysqlVersion80,
		Database: &databaseName,
		ExcludeTables: []string{
			"personnel_access_control_event",
			"\npersonnel_real_time",
			" ",
			"ext_alarm_message,\nmonitor_toxic_gas",
		},
	}

	args := uc.buildMysqldumpArgs(database)

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

func Test_BuildMysqldumpArgs_WithoutExcludedTables_OmitsIgnoreTableArgs(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	databaseName := "oa_db"
	database := &mysqltypes.MysqlDatabase{
		Version:  tools.MysqlVersion80,
		Database: &databaseName,
	}

	args := uc.buildMysqldumpArgs(database)

	for _, arg := range args {
		if strings.HasPrefix(arg, "--ignore-table=") {
			t.Fatalf("expected no --ignore-table args, got %v", args)
		}
	}
}
