package usecases_physical_postgresql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/walmath"
)

func Test_StallTracker_WhenFirstSample_DoesNotRestart(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.observe(walmath.LSN(100), base, time.Minute),
		"the first sample only arms the clock; it can never be a stall")
}

func Test_StallTracker_WhenRestartLsnAdvances_ReArmsAndDoesNotRestart(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.observe(walmath.LSN(100), base, time.Minute))
	require.False(t, tracker.observe(walmath.LSN(200), base.Add(2*time.Minute), time.Minute),
		"a changed restart_lsn means progress — the advance clock must reset")
}

func Test_StallTracker_WhenFrozenWithinTimeout_DoesNotRestart(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.observe(walmath.LSN(100), base, time.Minute))
	require.False(t, tracker.observe(walmath.LSN(100), base.Add(30*time.Second), time.Minute),
		"a frozen restart_lsn within the stall timeout is not yet a stall")
}

func Test_StallTracker_WhenFrozenPastTimeout_RestartsThenReArms(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.observe(walmath.LSN(100), base, time.Minute))
	require.True(t, tracker.observe(walmath.LSN(100), base.Add(90*time.Second), time.Minute),
		"a frozen restart_lsn past the stall timeout must trigger a restart")

	require.False(t, tracker.observe(walmath.LSN(100), base.Add(2*time.Minute), time.Minute),
		"after firing, the clock re-arms so we restart at most once per window")
	require.True(t, tracker.observe(walmath.LSN(100), base.Add(4*time.Minute), time.Minute),
		"a sustained stall fires again only after another full timeout window")
}
