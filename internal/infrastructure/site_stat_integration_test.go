package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIntegrationSiteStat(t *testing.T) {
	db := setupIntegrationDB(t, "users", "matches", "deck_codes")
	r := NewSiteStat(db)

	now := time.Now().Local().Truncate(time.Microsecond)

	// 退会(論理削除)したユーザは数えない
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, created_at, updated_at, deleted_at) VALUES (?, ?, ?, NULL), (?, ?, ?, ?)`,
		"site-stat-active", now, now,
		"site-stat-deleted", now, now, now,
	).Error)

	stat, err := r.Find(context.Background())

	require.NoError(t, err)
	require.Equal(t, int64(1), stat.UserCount)
	require.Equal(t, int64(0), stat.MatchCount)
	require.Equal(t, int64(0), stat.DeckCodeCount)
}
