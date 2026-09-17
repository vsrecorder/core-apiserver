package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// 相手デッキの入力候補の集計を実DBで確認する。
// sqlmock では、出力列の別名(sprite1 / sprite2 / count)を GROUP BY・ORDER BY で使えるかや、
// 結合条件がスキーマと合っているかは分からない。
func TestIntegrationOpponentDeckCandidate(t *testing.T) {
	db := setupIntegrationDB(t, "match_pokemon_sprites", "matches", "records", "pokemon_sprites")

	const (
		userA = "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		userB = "KBp7roRDZobZg1t0OPzFR1kvLeO2"
	)

	now := time.Now().Local().Truncate(time.Microsecond)
	since := now.Add(-90 * 24 * time.Hour)

	require.NoError(t, db.Create(&model.PokemonSprite{ID: "0887", Name: "ザマゼンタ"}).Error)
	require.NoError(t, db.Create(&model.PokemonSprite{ID: "0006", Name: "リザードン"}).Error)

	createRecord := func(id string, uid string, private bool, deleted bool, ignoreStats bool) {
		t.Helper()

		record := &model.Record{ID: id, CreatedAt: now, UpdatedAt: now, UserId: uid, EventDate: now, PrivateFlg: private, IgnoreStatsFlg: ignoreStats}
		require.NoError(t, db.Create(record).Error)
		if deleted {
			require.NoError(t, db.Model(record).Update("deleted_at", gorm.DeletedAt{Time: now, Valid: true}).Error)
		}
	}

	createMatch := func(id string, recordId string, uid string, deckInfo string, createdAt time.Time, defaultVictory bool, sprites ...string) {
		t.Helper()

		require.NoError(t, db.Create(&model.Match{
			ID: id, CreatedAt: createdAt, UpdatedAt: createdAt, RecordId: recordId, UserId: uid,
			OpponentsDeckInfo: deckInfo, DefaultVictoryFlg: defaultVictory,
		}).Error)
		for i, spriteId := range sprites {
			require.NoError(t, db.Create(&model.MatchPokemonSprite{
				MatchId: id, Position: uint(i + 1), PokemonSpriteId: spriteId,
			}).Error)
		}
	}

	// 非公開の記録(A)も公開の記録(B)も数える
	createRecord("rec-a-private", userA, true, false, false)
	createRecord("rec-b-public", userB, false, false, false)
	createRecord("rec-b-deleted", userB, false, true, false)
	// 集計対象外(ignore_stats_flg)の記録は、他の集計と同じく数えない
	createRecord("rec-b-ignored", userB, false, false, true)

	createMatch("m-a-1", "rec-a-private", userA, "ロストバレット", now, false, "0887", "0006")
	createMatch("m-a-2", "rec-a-private", userA, "ロストバレット", now, false, "0887", "0006")
	createMatch("m-b-1", "rec-b-public", userB, "ロストバレット", now.Add(-time.Hour), false, "0887", "0006")
	createMatch("m-b-2", "rec-b-public", userB, "サーナイトex", now, false)
	// 同じ表記でもスプライトが違えば別の候補
	createMatch("m-b-3", "rec-b-public", userB, "ロストバレット", now, false, "0887")
	// 数えないもの: 不戦勝・表記なし・期間外・削除された記録・集計対象外の記録の対戦
	createMatch("m-b-default", "rec-b-public", userB, "不戦勝の相手", now, true)
	createMatch("m-b-empty", "rec-b-public", userB, "", now, false)
	createMatch("m-b-old", "rec-b-public", userB, "古い環境のデッキ", since.Add(-24*time.Hour), false)
	createMatch("m-b-deleted-record", "rec-b-deleted", userB, "消えた記録の相手", now, false)
	createMatch("m-b-ignored-record", "rec-b-ignored", userB, "ロストバレット", now, false, "0887", "0006")

	r := NewOpponentDeckCandidate(db)

	ret, err := r.FindOpponentDeckCandidates(context.Background(), &repository.OpponentDeckCandidateFilter{
		Since: since,
		Limit: 10,
	})
	require.NoError(t, err)

	require.Len(t, ret, 3)

	// 出現回数の多い順。非公開の記録の2件も含めて3回(集計対象外の記録の1件は含めない)
	require.Equal(t, "ロストバレット", ret[0].OpponentsDeckInfo)
	require.Equal(t, 3, ret[0].Count)
	require.Len(t, ret[0].PokemonSprites, 2)
	require.Equal(t, "0887", ret[0].PokemonSprites[0].ID)
	require.Equal(t, uint(1), ret[0].PokemonSprites[0].Position)
	require.Equal(t, "0006", ret[0].PokemonSprites[1].ID)
	require.Equal(t, uint(2), ret[0].PokemonSprites[1].Position)

	// 同数(1回)は最近使われたものが先(m-b-2 / m-b-3 はどちらも now なので表記順)
	require.Equal(t, "サーナイトex", ret[1].OpponentsDeckInfo)
	require.Equal(t, 1, ret[1].Count)
	require.Empty(t, ret[1].PokemonSprites)

	require.Equal(t, "ロストバレット", ret[2].OpponentsDeckInfo)
	require.Equal(t, 1, ret[2].Count)
	require.Len(t, ret[2].PokemonSprites, 1)

	// limit は件数を絞る
	limited, err := r.FindOpponentDeckCandidates(context.Background(), &repository.OpponentDeckCandidateFilter{
		Since: since,
		Limit: 1,
	})
	require.NoError(t, err)
	require.Len(t, limited, 1)
	require.Equal(t, 3, limited[0].Count)

	// UserId を指定すると、そのユーザーが作った対戦だけを数える(他人の対戦は混ざらない)
	ownOnly, err := r.FindOpponentDeckCandidates(context.Background(), &repository.OpponentDeckCandidateFilter{
		UserId: userA,
		Since:  since,
		Limit:  10,
	})
	require.NoError(t, err)
	require.Len(t, ownOnly, 1)
	require.Equal(t, "ロストバレット", ownOnly[0].OpponentsDeckInfo)
	require.Equal(t, 2, ownOnly[0].Count)
}
