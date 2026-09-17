// Package ratelimit は単一プロセス内で完結する、シンプルな固定ウィンドウ方式の
// インメモリレート制限を提供する。複数インスタンスにまたがるレート制限には対応しない。
package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time

	// nextSweepAt は次に全キーを掃除する時刻。
	// Allow はそのキーの古い試行しか捨てないため、一度使われて二度と来ないキーの
	// エントリはそのまま残る。キーは認証済みユーザーのIDなので増え方は緩やかだが、
	// APIサーバはメモリ上限を低く抑えたコンテナで長期間動くため、放置すると効いてくる。
	// ウィンドウごとに1回まとめて掃除し、保持するキーを「直近2ウィンドウで試行があった
	// もの」に抑える(試行のたびに全キーを見ると、試行数×キー数の計算になるため)。
	nextSweepAt time.Time

	// now は現在時刻の取得。テストから時計を進めるために差し替えられるようにしている。
	now func() time.Time
}

// New は指定した期間(window)内にキーごとに limit 回まで Allow を許可する Limiter を生成する。
func New(limit int, window time.Duration) *Limiter {
	l := &Limiter{
		limit:  limit,
		window: window,
		hits:   make(map[string][]time.Time),
		now:    time.Now,
	}
	l.nextSweepAt = l.now().Add(window)

	return l
}

// Allow は key に対する1回の試行を消費し、ウィンドウ内の試行回数が制限を超えていなければ true を返す。
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-l.window)

	l.sweepIfDue(now, cutoff)

	times := l.hits[key]
	filtered := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= l.limit {
		l.hits[key] = filtered
		return false
	}

	l.hits[key] = append(filtered, now)
	return true
}

// sweepIfDue はウィンドウごとに1回、ウィンドウ内の試行が1つも残っていないキーを削除する。
// 呼び出し側でロックを取っていること。
func (l *Limiter) sweepIfDue(now time.Time, cutoff time.Time) {
	if now.Before(l.nextSweepAt) {
		return
	}
	l.nextSweepAt = now.Add(l.window)

	for key, times := range l.hits {
		alive := false
		for _, t := range times {
			if t.After(cutoff) {
				alive = true
				break
			}
		}
		if !alive {
			delete(l.hits, key)
		}
	}
}

// Reset は全キーの試行記録を破棄する。テストが互いの消費分に影響されないよう
// 状態を初期化するために使う。
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.hits = make(map[string][]time.Time)
	l.nextSweepAt = l.now().Add(l.window)
}

// keyCount は保持しているキーの数を返す(掃除の検証用)。
func (l *Limiter) keyCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.hits)
}
