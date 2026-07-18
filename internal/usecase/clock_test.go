package usecase

import (
	"testing"
	"time"
)

// overrideTimeNow は現在時刻を固定値に差し替える。
// パッケージ変数を書き換えるため、これを使うテストは並列実行しないこと。
func overrideTimeNow(t *testing.T, fixed time.Time) {
	t.Helper()

	original := timeNow
	timeNow = func() time.Time { return fixed }
	t.Cleanup(func() { timeNow = original })
}
