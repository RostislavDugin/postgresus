package usecases_logical_mysql

import (
	"slices"
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
