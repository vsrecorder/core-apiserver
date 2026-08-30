package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveTargetMonday(t *testing.T) {
	// 2026-08-24(月) 08:00 に実行 → 対象は先週 08-17(月)
	monday := time.Date(2026, 8, 24, 8, 0, 0, 0, time.Local)
	// 2026-08-30(日) 23:00 に実行(週の最終日) → 今週の月曜は 08-24 なので先週は 08-17
	sunday := time.Date(2026, 8, 30, 23, 0, 0, 0, time.Local)

	t.Run("正常系_未指定なら実行日が属する週の1つ前の月曜", func(t *testing.T) {
		for _, now := range []time.Time{monday, sunday} {
			got, err := resolveTargetMonday("", now)

			require.NoError(t, err)
			require.Equal(t, time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local), got, now)
		}
	})

	t.Run("正常系_週内の任意日を指定したらその週の月曜に正規化する", func(t *testing.T) {
		for _, day := range []string{"2026-08-17", "2026-08-19", "2026-08-23"} {
			got, err := resolveTargetMonday(day, monday)

			require.NoError(t, err)
			require.Equal(t, time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local), got, day)
		}
	})

	t.Run("異常系_形式が不正ならエラー", func(t *testing.T) {
		_, err := resolveTargetMonday("2026/08/17", monday)

		require.Error(t, err)
	})
}
