package namelist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseUniqueNames_WithBlankInput_ReturnsEmptySlice(t *testing.T) {
	assert.Empty(t, ParseUniqueNames(""))
	assert.Empty(t, ParseUniqueNames("   "))
	assert.Empty(t, ParseUniqueNames(",,,"))
	assert.NotNil(t, ParseUniqueNames(""))
}

func Test_ParseUniqueNames_WithPastedMultilineInput_ReturnsTrimmedNames(t *testing.T) {
	pastedNames := "personnel_access_control_event,\npersonnel_real_time_location,\r\n personnel_real_time\t"

	assert.Equal(
		t,
		[]string{"personnel_access_control_event", "personnel_real_time_location", "personnel_real_time"},
		ParseUniqueNames(pastedNames),
	)
}

func Test_ParseUniqueNames_WithDuplicatesAndEmptyEntries_ReturnsUniqueNames(t *testing.T) {
	assert.Equal(t, []string{"orders", "users"}, ParseUniqueNames("orders,, users ,orders"))
}

func Test_NormalizeUniqueNames_WithSeparatorsInsideEntries_SplitsThemApart(t *testing.T) {
	storedNames := []string{"orders", "\nusers, payments", "   ", "orders"}

	assert.Equal(t, []string{"orders", "users", "payments"}, NormalizeUniqueNames(storedNames))
}

func Test_FormatUniqueNames_WithDirtyNames_JoinsNormalizedNamesWithCommas(t *testing.T) {
	assert.Equal(t, "orders,users", FormatUniqueNames([]string{" orders ", "\nusers", "orders", ""}))
	assert.Equal(t, "", FormatUniqueNames(nil))
}

func Test_FormatTruncatedNames_WhenNamesFitTheLimit_ListsThemAll(t *testing.T) {
	assert.Equal(t, "orders, users", FormatTruncatedNames([]string{"orders", "users"}))
	assert.Equal(t, "", FormatTruncatedNames(nil))
}

func Test_FormatTruncatedNames_WhenNamesExceedTheLimit_AppendsTheRemainingCount(t *testing.T) {
	manyNames := []string{"a", "b", "c", "d", "e", "f", "g"}

	assert.Equal(t, "a, b, c, d, e and 2 more", FormatTruncatedNames(manyNames))
}
