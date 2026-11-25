package batchreport

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinish(t *testing.T) {
	t.Run("正常系_失敗が無ければOKで、知らせない", func(t *testing.T) {
		called := false

		code := Finish(Result{AppName: "notify-weekly-report", Candidates: 120}, func(string) error {
			called = true
			return nil
		})

		require.Equal(t, ExitCodeOK, code)
		require.False(t, called)
	})

	t.Run("正常系_対象が0人でも失敗が無ければOK", func(t *testing.T) {
		require.Equal(t, ExitCodeOK, Finish(Result{AppName: "notify-weekly-report"}, nil))
	})

	t.Run("異常系_1人でも失敗があればNGで、知らせる", func(t *testing.T) {
		var got string

		code := Finish(Result{AppName: "notify-weekly-report", Candidates: 120, Failed: 1}, func(message string) error {
			got = message
			return nil
		})

		require.Equal(t, ExitCodeNG, code)
		require.Contains(t, got, "notify-weekly-report")
		require.Contains(t, got, "対象 120 人のうち 1 人の処理に失敗")
		require.NotContains(t, got, "(全員)")
	})

	// DB が落ちたときなど。以前は全員ぶん失敗しても終了コード 0 で終わっていた
	t.Run("異常系_全員ぶん失敗したら、全員と分かる文面で知らせる", func(t *testing.T) {
		var got string

		code := Finish(Result{AppName: "notify-streak-nudge", Candidates: 80, Failed: 80}, func(message string) error {
			got = message
			return nil
		})

		require.Equal(t, ExitCodeNG, code)
		require.Contains(t, got, "対象 80 人のうち 80 人の処理に失敗(全員)")
	})

	t.Run("異常系_対象の一覧を取れなかったときもNGで知らせる", func(t *testing.T) {
		var got string

		code := Finish(Result{AppName: "notify-period-report", ListFailed: 1}, func(message string) error {
			got = message
			return nil
		})

		require.Equal(t, ExitCodeNG, code)
		require.Contains(t, got, "一覧を 1 回取得できませんでした")
	})

	t.Run("異常系_知らせる先が無くてもNGで終える", func(t *testing.T) {
		require.Equal(t, ExitCodeNG, Finish(Result{AppName: "notify-weekly-report", Candidates: 3, Failed: 2}, nil))
	})

	t.Run("異常系_知らせることに失敗してもNGで終える", func(t *testing.T) {
		code := Finish(Result{AppName: "notify-weekly-report", Candidates: 3, Failed: 2}, func(string) error {
			return errors.New("slack is down")
		})

		require.Equal(t, ExitCodeNG, code)
	})
}
