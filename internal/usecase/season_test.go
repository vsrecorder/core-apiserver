package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCurrentSeasonLabel(t *testing.T) {
	t.Run("9月1日以降は年をまたいだ翌年がシーズン識別年になる", func(t *testing.T) {
		now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.Local)
		require.Equal(t, "2027", CurrentSeasonLabel(now))
	})

	t.Run("8月31日は前年9月始まりのシーズンに含まれる", func(t *testing.T) {
		now := time.Date(2026, 8, 31, 23, 59, 0, 0, time.Local)
		require.Equal(t, "2026", CurrentSeasonLabel(now))
	})
}

func TestSeasonRange(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.Local)

	t.Run("season空文字なら現在のシーズンを返す", func(t *testing.T) {
		fromDate, toDate, err := seasonRange("", now)

		require.NoError(t, err)
		require.Equal(t, time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), fromDate)
		require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local), toDate)
	})

	t.Run("season指定時はその年のシーズン期間を返す", func(t *testing.T) {
		fromDate, toDate, err := seasonRange("2024", now)

		require.NoError(t, err)
		require.Equal(t, time.Date(2023, 9, 1, 0, 0, 0, 0, time.Local), fromDate)
		require.Equal(t, time.Date(2024, 9, 1, 0, 0, 0, 0, time.Local), toDate)
	})

	t.Run("不正なseasonはエラーを返す", func(t *testing.T) {
		_, _, err := seasonRange("not-a-year", now)

		require.Error(t, err)
	})
}
