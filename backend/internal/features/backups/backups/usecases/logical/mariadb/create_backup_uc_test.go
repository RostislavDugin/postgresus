package usecases_logical_mariadb

import (
	"slices"
	"testing"

	mariadbtypes "databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/util/tools"
)

func Test_BuildMariadbDumpArgs_WhenExtendedInsertDisabled_SkipsExtendedInsert(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	database := &mariadbtypes.MariadbDatabase{
		Version:             tools.MariadbVersion1011,
		IsUseExtendedInsert: false,
	}

	args := uc.buildMariadbDumpArgs(database)

	if !slices.Contains(args, "--skip-extended-insert") {
		t.Fatalf("expected --skip-extended-insert when extended inserts are disabled, got %v", args)
	}
}

func Test_BuildMariadbDumpArgs_WhenExtendedInsertEnabled_OmitsSkipExtendedInsert(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	database := &mariadbtypes.MariadbDatabase{
		Version:             tools.MariadbVersion1011,
		IsUseExtendedInsert: true,
	}

	args := uc.buildMariadbDumpArgs(database)

	if slices.Contains(args, "--skip-extended-insert") {
		t.Fatalf("expected no --skip-extended-insert when extended inserts are enabled, got %v", args)
	}
}
