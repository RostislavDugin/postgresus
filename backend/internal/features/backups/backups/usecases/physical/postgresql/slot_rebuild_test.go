package usecases_physical_postgresql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test_IsOwnedReceiverBackend_WhenOurReceiverHoldsSlot_ReturnsTrue confirms the
// rebuild attribution accepts a slot held by our own pg_receivewal (active PID +
// our application-name prefix) — the only case where force-termination is safe.
func Test_IsOwnedReceiverBackend_WhenOurReceiverHoldsSlot_ReturnsTrue(t *testing.T) {
	state := &SlotState{
		Active:          true,
		ActivePID:       new(4321),
		ApplicationName: receivewalApplicationNamePrefix + "11111111-1111-1111-1111-111111111111",
	}

	require.True(t, isOwnedReceiverBackend(state))
}

// Test_IsOwnedReceiverBackend_WhenForeignConsumerHoldsSlot_ReturnsFalse is the
// slot-stolen guard: a slot held by a consumer whose application name is not ours
// must not be attributed to us, so rebuildSlot refuses to terminate or drop it.
func Test_IsOwnedReceiverBackend_WhenForeignConsumerHoldsSlot_ReturnsFalse(t *testing.T) {
	state := &SlotState{
		Active:          true,
		ActivePID:       new(9999),
		ApplicationName: "some_other_replica",
	}

	require.False(t, isOwnedReceiverBackend(state))
}

// Test_IsOwnedReceiverBackend_WhenNoActiveBackend_ReturnsFalse: a slot with no
// active PID has no walsender to terminate — there is nothing to attribute.
func Test_IsOwnedReceiverBackend_WhenNoActiveBackend_ReturnsFalse(t *testing.T) {
	state := &SlotState{
		Active:          false,
		ActivePID:       nil,
		ApplicationName: receivewalApplicationNamePrefix + "22222222-2222-2222-2222-222222222222",
	}

	require.False(t, isOwnedReceiverBackend(state))
}

// Test_IsOwnedReceiverBackend_WhenActiveButEmptyApplicationName_ReturnsFalse:
// an active backend we cannot name (no pg_stat_replication join, empty app name)
// is treated as foreign — we never drop a slot we cannot prove is ours.
func Test_IsOwnedReceiverBackend_WhenActiveButEmptyApplicationName_ReturnsFalse(t *testing.T) {
	state := &SlotState{
		Active:          true,
		ActivePID:       new(1234),
		ApplicationName: "",
	}

	require.False(t, isOwnedReceiverBackend(state))
}
