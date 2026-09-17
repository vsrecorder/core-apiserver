package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// newTestLimiter は時計を差し替えた Limiter を返す。now を進めることでウィンドウの経過を再現する。
func newTestLimiter(limit int, window time.Duration, now *time.Time) *Limiter {
	l := New(limit, window)
	l.now = func() time.Time { return *now }
	l.nextSweepAt = now.Add(window)

	return l
}

func TestLimiter(t *testing.T) {
	base := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

	t.Run("正常系_上限までは許可しそれ以上は拒否する", func(t *testing.T) {
		now := base
		l := newTestLimiter(2, time.Hour, &now)

		require.True(t, l.Allow("a"))
		require.True(t, l.Allow("a"))
		require.False(t, l.Allow("a"))

		// 別のキーには影響しない
		require.True(t, l.Allow("b"))
	})

	t.Run("正常系_ウィンドウが過ぎれば再び許可する", func(t *testing.T) {
		now := base
		l := newTestLimiter(1, time.Hour, &now)

		require.True(t, l.Allow("a"))
		require.False(t, l.Allow("a"))

		now = base.Add(time.Hour + time.Second)
		require.True(t, l.Allow("a"))
	})

	// 一度使われて二度と来ないキーが残り続けないこと。掃除はウィンドウごとに1回、
	// 次の Allow の中で行われる。
	t.Run("正常系_ウィンドウ内に試行の無いキーは掃除される", func(t *testing.T) {
		now := base
		l := newTestLimiter(10, time.Hour, &now)

		require.True(t, l.Allow("stale-1"))
		require.True(t, l.Allow("stale-2"))
		require.Equal(t, 2, l.keyCount())

		// ウィンドウ内はまだ掃除されない
		now = base.Add(30 * time.Minute)
		require.True(t, l.Allow("stale-1"))
		require.Equal(t, 2, l.keyCount())

		// ウィンドウを過ぎた最初の試行で、ウィンドウ内に試行の無いキーが消える
		now = base.Add(2*time.Hour + time.Second)
		require.True(t, l.Allow("fresh"))
		require.Equal(t, 1, l.keyCount())
	})

	t.Run("正常系_掃除後もウィンドウ内のキーの試行回数は保たれる", func(t *testing.T) {
		now := base
		l := newTestLimiter(2, time.Hour, &now)

		require.True(t, l.Allow("stale"))

		// "active" は掃除の直前に試行しているため、掃除後も回数が引き継がれる
		now = base.Add(time.Hour + time.Second)
		require.True(t, l.Allow("active"))
		now = base.Add(time.Hour + 2*time.Second)
		require.True(t, l.Allow("active"))
		require.False(t, l.Allow("active"))
		require.Equal(t, 1, l.keyCount())
	})

	t.Run("正常系_Resetで全ての試行記録が消える", func(t *testing.T) {
		now := base
		l := newTestLimiter(1, time.Hour, &now)

		require.True(t, l.Allow("a"))
		require.False(t, l.Allow("a"))

		l.Reset()

		require.Equal(t, 0, l.keyCount())
		require.True(t, l.Allow("a"))
	})
}
