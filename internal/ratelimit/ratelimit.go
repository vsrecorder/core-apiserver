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
}

// New は指定した期間(window)内にキーごとに limit 回まで Allow を許可する Limiter を生成する。
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{
		limit:  limit,
		window: window,
		hits:   make(map[string][]time.Time),
	}
}

// Allow は key に対する1回の試行を消費し、ウィンドウ内の試行回数が制限を超えていなければ true を返す。
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

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
