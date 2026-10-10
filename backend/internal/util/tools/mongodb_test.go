package tools

import "testing"

func Test_IsMongodbBackupVersionHigherThanRestoreVersion_AcrossSupportedVersions_RefusesDowngrades(t *testing.T) {
	restoreCases := []struct {
		backup, restore MongodbVersion
		isDowngrade     bool
	}{
		{MongodbVersion9, MongodbVersion8, true},
		{MongodbVersion8, MongodbVersion9, false},
		{MongodbVersion9, MongodbVersion9, false},
		{MongodbVersion4, MongodbVersion9, false},
	}

	for _, restoreCase := range restoreCases {
		isDowngrade, err := IsMongodbBackupVersionHigherThanRestoreVersion(restoreCase.backup, restoreCase.restore)
		if err != nil {
			t.Fatalf("ordering %s onto %s: %v", restoreCase.backup, restoreCase.restore, err)
		}

		if isDowngrade != restoreCase.isDowngrade {
			t.Errorf("restoring a %s backup onto %s: got %v, expected %v",
				restoreCase.backup, restoreCase.restore, isDowngrade, restoreCase.isDowngrade)
		}
	}
}

func Test_IsMongodbBackupVersionHigherThanRestoreVersion_WithAnIdentityItCannotRead_ReturnsError(t *testing.T) {
	if _, err := IsMongodbBackupVersionHigherThanRestoreVersion("", MongodbVersion9); err == nil {
		t.Error("an identity the guard cannot order must be an error, not a silent zero")
	}
}
