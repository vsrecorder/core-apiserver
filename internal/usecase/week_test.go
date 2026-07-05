package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWeekRange(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"MondayStartsTheWeek":       test_weekRange_MondayStartsTheWeek,
		"AnyDayResolvesToItsMonday": test_weekRange_AnyDayResolvesToItsMonday,
		"SundayBelongsToSameWeek":   test_weekRange_SundayBelongsToSameWeek,
		"EmptyUsesNow":              test_weekRange_EmptyUsesNow,
		"InvalidFormatReturnsError": test_weekRange_InvalidFormatReturnsError,
		"RangeIsSevenDaysHalfOpen":  test_weekRange_RangeIsSevenDaysHalfOpen,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_weekRange_MondayStartsTheWeek(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.Local) // 2026-07-06 は月曜
	from, to, err := weekRange("2026-07-06", now)
	require.NoError(t, err)
	require.Equal(t, time.Monday, from.Weekday())
	require.Equal(t, time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local), from)
	require.Equal(t, time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local), to)
}

func test_weekRange_AnyDayResolvesToItsMonday(t *testing.T) {
	now := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	// 2026-07-09 は木曜。属する週の月曜は 2026-07-06。
	from, _, err := weekRange("2026-07-09", now)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local), from)
}

func test_weekRange_SundayBelongsToSameWeek(t *testing.T) {
	now := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	// 2026-07-12 は日曜。月曜始まりなので属する週の月曜は 2026-07-06。
	from, to, err := weekRange("2026-07-12", now)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local), from)
	require.Equal(t, time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local), to)
}

func test_weekRange_EmptyUsesNow(t *testing.T) {
	now := time.Date(2026, 7, 9, 15, 30, 0, 0, time.Local) // 木曜
	from, to, err := weekRange("", now)
	require.NoError(t, err)
	require.Equal(t, time.Monday, from.Weekday())
	require.True(t, !now.Before(from) && now.Before(to))
}

func test_weekRange_InvalidFormatReturnsError(t *testing.T) {
	now := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	_, _, err := weekRange("2026/07/06", now)
	require.Error(t, err)
}

func test_weekRange_RangeIsSevenDaysHalfOpen(t *testing.T) {
	now := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	from, to, err := weekRange("2026-07-06", now)
	require.NoError(t, err)
	require.Equal(t, from.AddDate(0, 0, 7), to)
}
