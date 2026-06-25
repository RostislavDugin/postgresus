package gfs

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// itemAt builds an Item with a deterministic ID derived from the offset so a
// test can refer to the same instant twice and still get a stable identity.
func itemAt(reference time.Time, offset time.Duration) Item {
	at := reference.Add(offset)

	return Item{
		ID:        uuid.NewSHA1(uuid.Nil, []byte(at.Format(time.RFC3339Nano))),
		CreatedAt: at,
	}
}

func Test_GetItemsToRetain_WhenNoItems_ReturnsEmptyKeepSet(t *testing.T) {
	keep := GetItemsToRetain(nil, 1, 1, 1, 1, 1)

	assert.Empty(t, keep)
}

func Test_GetItemsToRetain_WhenAllRetentionsZero_KeepsNothing(t *testing.T) {
	reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	items := []Item{
		itemAt(reference, 0),
		itemAt(reference, -1*time.Hour),
		itemAt(reference, -24*time.Hour),
	}

	keep := GetItemsToRetain(items, 0, 0, 0, 0, 0)

	assert.Empty(t, keep)
}

func Test_GetItemsToRetain_WhenItemFillsEveryLevel_KeptOnce(t *testing.T) {
	reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	newest := itemAt(reference, 0)
	older := itemAt(reference, -90*24*time.Hour)

	keep := GetItemsToRetain([]Item{newest, older}, 1, 1, 1, 1, 1)

	// The single newest item simultaneously fills the hourly, daily, weekly,
	// monthly, and yearly slot, so each level is satisfied by it alone.
	assert.True(t, keep[newest.ID])
	assert.Len(t, keep, 1)
}

func Test_GetItemsToRetain_WhenInputUnsorted_SortsBeforeApplyingScheme(t *testing.T) {
	reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	newest := itemAt(reference, 0)
	middle := itemAt(reference, -1*time.Hour)
	oldest := itemAt(reference, -2*time.Hour)

	// Deliberately shuffled: the function must sort newest-first itself.
	unsorted := []Item{oldest, newest, middle}

	keep := GetItemsToRetain(unsorted, 2, 0, 0, 0, 0)

	assert.True(t, keep[newest.ID])
	assert.True(t, keep[middle.ID])
	assert.False(t, keep[oldest.ID])
	assert.Len(t, keep, 2)
}

func Test_GetItemsToRetain_DoesNotMutateCallerSlice(t *testing.T) {
	reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	oldest := itemAt(reference, -2*time.Hour)
	newest := itemAt(reference, 0)
	items := []Item{oldest, newest}

	GetItemsToRetain(items, 5, 0, 0, 0, 0)

	assert.Equal(t, oldest.ID, items[0].ID)
	assert.Equal(t, newest.ID, items[1].ID)
}

func Test_GetItemsToRetain_WhenMultipleItemsShareOneHour_KeepsOnlyNewestOfThatHour(t *testing.T) {
	reference := time.Date(2026, 5, 30, 12, 30, 0, 0, time.UTC)
	hourNewest := itemAt(reference, 0)
	hourMiddle := itemAt(reference, -10*time.Minute)
	hourOldest := itemAt(reference, -20*time.Minute)
	previousHour := itemAt(reference, -1*time.Hour)

	keep := GetItemsToRetain(
		[]Item{hourMiddle, hourOldest, hourNewest, previousHour}, 2, 0, 0, 0, 0,
	)

	// All three of the first three share the same hour bucket, so only the
	// newest of them is kept; the previous hour fills the second hourly slot.
	assert.True(t, keep[hourNewest.ID])
	assert.True(t, keep[previousHour.ID])
	assert.False(t, keep[hourMiddle.ID])
	assert.False(t, keep[hourOldest.ID])
	assert.Len(t, keep, 2)
}

func Test_GetItemsToRetain_WhenHourlyCountExceeded_StopsKeeping(t *testing.T) {
	reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	items := []Item{
		itemAt(reference, 0),
		itemAt(reference, -1*time.Hour),
		itemAt(reference, -2*time.Hour),
		itemAt(reference, -3*time.Hour),
	}

	keep := GetItemsToRetain(items, 2, 0, 0, 0, 0)

	assert.True(t, keep[items[0].ID])
	assert.True(t, keep[items[1].ID])
	assert.False(t, keep[items[2].ID])
	assert.False(t, keep[items[3].ID])
	assert.Len(t, keep, 2)
}

func Test_GetItemsToRetain_WithDailyRetention_KeepsOnePerDay(t *testing.T) {
	reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	today := itemAt(reference, 0)
	todayEarlier := itemAt(reference, -3*time.Hour)
	yesterday := itemAt(reference, -24*time.Hour)
	twoDaysAgo := itemAt(reference, -48*time.Hour)

	keep := GetItemsToRetain(
		[]Item{today, todayEarlier, yesterday, twoDaysAgo}, 0, 3, 0, 0, 0,
	)

	assert.True(t, keep[today.ID])
	assert.True(t, keep[yesterday.ID])
	assert.True(t, keep[twoDaysAgo.ID])
	assert.False(t, keep[todayEarlier.ID])
	assert.Len(t, keep, 3)
}

func Test_GetItemsToRetain_WhenItemsShareExactTimestamp_IsDeterministic(t *testing.T) {
	at := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	first := Item{ID: uuid.New(), CreatedAt: at}
	second := Item{ID: uuid.New(), CreatedAt: at}

	// Two items in the same hour bucket: exactly one survives, regardless of
	// the input order, and the choice is stable across repeated runs.
	keepAsGiven := GetItemsToRetain([]Item{first, second}, 1, 0, 0, 0, 0)
	keepReversed := GetItemsToRetain([]Item{second, first}, 1, 0, 0, 0, 0)

	assert.Len(t, keepAsGiven, 1)
	assert.Equal(t, keepAsGiven, keepReversed)
}

func Test_GetItemsToRetain_WithMixedLevels_KeepsExpectedItems(t *testing.T) {
	reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	now := itemAt(reference, 0)
	oneHourAgo := itemAt(reference, -1*time.Hour)
	yesterday := itemAt(reference, -24*time.Hour)
	lastWeek := itemAt(reference, -8*24*time.Hour)
	lastMonth := itemAt(reference, -40*24*time.Hour)
	lastYear := itemAt(reference, -300*24*time.Hour)

	items := []Item{now, oneHourAgo, yesterday, lastWeek, lastMonth, lastYear}

	keep := GetItemsToRetain(items, 2, 2, 2, 2, 2)

	assert.True(t, keep[now.ID])
	assert.True(t, keep[oneHourAgo.ID])
	assert.True(t, keep[yesterday.ID])
	assert.True(t, keep[lastWeek.ID])
	assert.True(t, keep[lastMonth.ID])
	assert.True(t, keep[lastYear.ID])
}

func Test_GetItemsToRetain_DropsItemsOlderThanEveryWindow(t *testing.T) {
	reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	recent := itemAt(reference, 0)
	ancient := itemAt(reference, -10*365*24*time.Hour)

	keep := GetItemsToRetain([]Item{recent, ancient}, 1, 1, 1, 1, 1)

	assert.True(t, keep[recent.ID])
	assert.False(t, keep[ancient.ID])
	assert.Len(t, keep, 1)
}
