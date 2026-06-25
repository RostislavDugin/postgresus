package pg17

import (
	"testing"

	physicaltesting "databasus-backend/internal/features/tests/physical/postgresql/shared"
)

const (
	pgVersion = "17"
	pgImage   = "postgres:17"
)

func Test_PhysicalRestore_FullOnly_RecoversBaseRows(t *testing.T) {
	physicaltesting.RunFullOnlyRecoversBaseRows(t, pgVersion, pgImage)
}

func Test_PhysicalRestore_FullPlusTwoIncrementals_RecoversAllRows(t *testing.T) {
	physicaltesting.RunFullPlusTwoIncrementalsRecoversAllRows(t, pgVersion, pgImage)
}

func Test_PhysicalRestore_FullTwoIncrementalsPlusWal_RecoversToTarget(t *testing.T) {
	physicaltesting.RunFullTwoIncrementalsPlusWalRecoversToTarget(t, pgVersion, pgImage)
}

func Test_PhysicalRestore_WhenWalGapBeforeTarget_TokenRequestReturns422(t *testing.T) {
	physicaltesting.RunWhenWalGapBeforeTargetTokenRequestReturns422(t, pgVersion, pgImage)
}

func Test_PhysicalWalStream_OnTimelineSwitch_CatalogsHistoryOnParentDatabaseID(t *testing.T) {
	physicaltesting.RunWalStreamCatalogsHistoryOnTimelineSwitch(t, pgVersion, pgImage)
}

func Test_PhysicalRestore_BootViaEntrypointVolumeMount_RecoversBaseRows(t *testing.T) {
	physicaltesting.RunBootViaEntrypointVolumeMountRecoversBaseRows(t, pgVersion, pgImage)
}

func Test_PhysicalRestore_PitrBootViaEntrypointVolumeMount_RecoversToTarget(t *testing.T) {
	physicaltesting.RunPitrBootViaEntrypointVolumeMountRecoversToTarget(t, pgVersion, pgImage)
}

func Test_PhysicalRestore_WhenOutputDirIsMisaimed_RecoveryScriptRefuses(t *testing.T) {
	physicaltesting.RunRejectsMisaimedRestoreTarget(t, pgVersion, pgImage)
}

func Test_PhysicalWalSlot_AppearsWhenBackupingStarts_RemovedWhenDatabaseDeleted(t *testing.T) {
	physicaltesting.RunWalSlotAppearsWhenBackupingStartsRemovedWhenDatabaseDeleted(t, pgVersion, pgImage)
}

func Test_PhysicalWalSlot_WhenDatabaseDeletedWithStreamedWal_SlotRemovedSoNoWalStuck(t *testing.T) {
	physicaltesting.RunWalSlotWhenDatabaseDeletedWithStreamedWalSlotRemovedSoNoWalStuck(t, pgVersion, pgImage)
}
