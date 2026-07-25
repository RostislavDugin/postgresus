package usecases_physical_postgresql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_IsOwnedReceiverBackend_WhenOurReceiverHoldsSlot_ReturnsTrue(t *testing.T) {
	state := &SlotState{
		Active:          true,
		ActivePID:       new(4321),
		ApplicationName: receivewalApplicationNamePrefix + "11111111-1111-1111-1111-111111111111",
	}

	require.True(t, isOwnedReceiverBackend(state))
}

func Test_IsOwnedReceiverBackend_WhenForeignConsumerHoldsSlot_ReturnsFalse(t *testing.T) {
	state := &SlotState{
		Active:          true,
		ActivePID:       new(9999),
		ApplicationName: "some_other_replica",
	}

	require.False(t, isOwnedReceiverBackend(state))
}

func Test_IsOwnedReceiverBackend_WhenNoActiveBackend_ReturnsFalse(t *testing.T) {
	state := &SlotState{
		Active:          false,
		ActivePID:       nil,
		ApplicationName: receivewalApplicationNamePrefix + "22222222-2222-2222-2222-222222222222",
	}

	require.False(t, isOwnedReceiverBackend(state))
}

func Test_IsOwnedReceiverBackend_WhenActiveButEmptyApplicationName_ReturnsFalse(t *testing.T) {
	state := &SlotState{
		Active:          true,
		ActivePID:       new(1234),
		ApplicationName: "",
	}

	require.False(t, isOwnedReceiverBackend(state))
}

func Test_WalStream_RebuildAttemptCap_StopsFourthAttemptInHour(t *testing.T) {
	supervisor := &WalStreamSupervisor{}

	require.True(t, supervisor.recordRebuildAttemptWithinCap())
	require.True(t, supervisor.recordRebuildAttemptWithinCap())
	require.True(t, supervisor.recordRebuildAttemptWithinCap())
	require.False(t, supervisor.recordRebuildAttemptWithinCap())
}
