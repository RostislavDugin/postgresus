package usecases_physical_postgresql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_ClassifySlotBreak_WhenWalStatusLost_RebuildsForSlotLost(t *testing.T) {
	var extendedSince time.Time

	reason, shouldRebuild := classifySlotBreak(&SlotState{WalStatus: "lost"}, 0, &extendedSince)

	require.True(t, shouldRebuild)
	require.Equal(t, breakReasonSlotLost, reason)
}

func Test_ClassifySlotBreak_WhenWalStatusUnreserved_RebuildsForWalLag(t *testing.T) {
	var extendedSince time.Time

	reason, shouldRebuild := classifySlotBreak(&SlotState{WalStatus: "unreserved"}, 0, &extendedSince)

	require.True(t, shouldRebuild)
	require.Equal(t, breakReasonWalLag, reason)
}

func Test_ClassifySlotBreak_WhenExtendedPersistsPastHold_RebuildsForWalLag(t *testing.T) {
	extendedSince := time.Now().UTC().Add(-extendedSlotStatusHoldPeriod - time.Second)

	reason, shouldRebuild := classifySlotBreak(&SlotState{WalStatus: "extended"}, 0, &extendedSince)

	require.True(t, shouldRebuild)
	require.Equal(t, breakReasonWalLag, reason)
}

func Test_ClassifySlotBreak_WhenLagExceedsThreshold_RebuildsForWalLag(t *testing.T) {
	var extendedSince time.Time

	reason, shouldRebuild := classifySlotBreak(&SlotState{WalStatus: "reserved", LagBytes: 101}, 100, &extendedSince)

	require.True(t, shouldRebuild)
	require.Equal(t, breakReasonWalLag, reason)
}

func Test_ClassifySlotBreak_WhenForeignConsumerHoldsSlot_RebuildsForSlotStolen(t *testing.T) {
	var extendedSince time.Time

	foreignPID := 4242
	state := &SlotState{
		WalStatus:       "reserved",
		Active:          true,
		ActivePID:       &foreignPID,
		ApplicationName: "some_other_consumer",
	}

	reason, shouldRebuild := classifySlotBreak(state, 0, &extendedSince)

	require.True(t, shouldRebuild)
	require.Equal(t, breakReasonSlotStolen, reason)
}

func Test_ClassifySlotBreak_WhenOurReceiverActive_DoesNotRebuild(t *testing.T) {
	var extendedSince time.Time

	ourPID := 17
	state := &SlotState{
		WalStatus:       "reserved",
		Active:          true,
		ActivePID:       &ourPID,
		ApplicationName: receivewalApplicationNamePrefix + "db-1",
		LagBytes:        10,
	}

	_, shouldRebuild := classifySlotBreak(state, 100, &extendedSince)

	require.False(t, shouldRebuild)
}

func Test_ClassifySlotBreak_WhenSlotHealthy_DoesNotRebuildAndClearsExtendedSample(t *testing.T) {
	extendedSince := time.Now().UTC().Add(-extendedSlotStatusHoldPeriod - time.Second)

	_, shouldRebuild := classifySlotBreak(&SlotState{WalStatus: "reserved", LagBytes: 10}, 100, &extendedSince)

	require.False(t, shouldRebuild)
	require.True(t, extendedSince.IsZero())
}
