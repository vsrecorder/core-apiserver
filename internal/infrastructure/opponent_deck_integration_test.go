package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// 相手デッキの一覧と一括置き換えを実DBで確認する。
// sqlmock では、OR を含む条件の括弧の付き方(user_id の絞り込みが外れないか)や、
// スプライトを横に並べた結合での一致の判定がスキーマと合っているかは分からない。
func TestIntegrationOpponentDeck(t *testing.T) {
	db := setupIntegrationDB(t, "match_pokemon_sprites", "matches", "records", "pokemon_sprites")

	const (
		userA = "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		userB = "KBp7roRDZobZg1t0OPzFR1kvLeO2"
	)

	now := time.Now().Local().Truncate(time.Microsecond)
	eventDate := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)

	for _, sprite := range []model.PokemonSprite{
		{ID: "0887", Name: "ドラパルト"},
		{ID: "0006", Name: "リザードン"},
		{ID: "0282", Name: "サーナイト"},
	} {
		require.NoError(t, db.Create(&sprite).Error)
	}

	createRecord := func(id string, uid string, deleted bool, ignoreStats bool) {
		t.Helper()

		record := &model.Record{ID: id, CreatedAt: now, UpdatedAt: now, UserId: uid, EventDate: eventDate, IgnoreStatsFlg: ignoreStats}
		require.NoError(t, db.Create(record).Error)
		if deleted {
			require.NoError(t, db.Model(record).Update("deleted_at", gorm.DeletedAt{Time: now, Valid: true}).Error)
		}
	}

	// sprites は position 1, 2 の順。空文字はその枠を空ける
	createMatch := func(id string, recordId string, uid string, deckInfo string, sprites ...string) {
		t.Helper()

		require.NoError(t, db.Create(&model.Match{
			ID: id, CreatedAt: now, UpdatedAt: now, RecordId: recordId, UserId: uid, OpponentsDeckInfo: deckInfo,
		}).Error)
		for i, spriteId := range sprites {
			if spriteId == "" {
				continue
			}
			require.NoError(t, db.Create(&model.MatchPokemonSprite{
				MatchId: id, Position: uint(i + 1), PokemonSpriteId: spriteId,
			}).Error)
		}
	}

	createRecord("rec-a", userA, false, false)
	// 集計対象外の記録の対戦も、一括編集では対象にする
	createRecord("rec-a-ignored", userA, false, true)
	createRecord("rec-a-deleted", userA, true, false)
	createRecord("rec-b", userB, false, false)

	createMatch("m-a-1", "rec-a", userA, "ドラパ")
	createMatch("m-a-2", "rec-a", userA, "ドラパ")
	createMatch("m-a-3", "rec-a-ignored", userA, "ドラパ")
	// 同じ表記でもスプライトが違えば別の組み合わせ(置き換えない)
	createMatch("m-a-4", "rec-a", userA, "ドラパ", "0887")
	// 2体目だけのスプライト
	createMatch("m-a-5", "rec-a", userA, "サナ", "", "0282")
	// 表記もスプライトも無い対戦(不戦勝など)は一覧に出さない
	createMatch("m-a-empty", "rec-a", userA, "")
	// 削除された記録の対戦・他人の対戦は対象外
	createMatch("m-a-deleted-record", "rec-a-deleted", userA, "ドラパ")
	createMatch("m-b-1", "rec-b", userB, "ドラパ")

	r := NewOpponentDeck(db)
	ctx := context.Background()

	decks, err := r.FindByUserId(ctx, userA)
	require.NoError(t, err)
	require.Len(t, decks, 3)

	require.Equal(t, "ドラパ", decks[0].OpponentsDeckInfo)
	require.Equal(t, 3, decks[0].Count)
	require.Empty(t, decks[0].PokemonSprites)
	require.Equal(t, "2026-09-28", decks[0].LastEventDate)

	// 件数が同じ(1件)なら表記順
	require.Equal(t, "サナ", decks[1].OpponentsDeckInfo)
	require.Len(t, decks[1].PokemonSprites, 1)
	require.Equal(t, "0282", decks[1].SpriteIdAt(2))
	require.Equal(t, "ドラパ", decks[2].OpponentsDeckInfo)
	require.Equal(t, "0887", decks[2].SpriteIdAt(1))

	// スプライト無しの「ドラパ」を「ドラパルトex」＋2体に置き換える
	updated, err := r.Replace(ctx, userA,
		entity.NewOpponentDeckSpec("ドラパ", nil),
		entity.NewOpponentDeckSpec("ドラパルトex", []*entity.PokemonSprite{
			entity.NewPokemonSpriteWithPosition("0887", 1),
			entity.NewPokemonSpriteWithPosition("0006", 2),
		}),
	)
	require.NoError(t, err)
	require.Equal(t, 3, updated)

	infoOf := func(id string) string {
		t.Helper()
		var m model.Match
		require.NoError(t, db.Unscoped().Where("id = ?", id).First(&m).Error)
		return m.OpponentsDeckInfo
	}
	spritesOf := func(id string) []string {
		t.Helper()
		var rows []model.MatchPokemonSprite
		require.NoError(t, db.Where("match_id = ?", id).Order("position").Find(&rows).Error)
		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.PokemonSpriteId)
		}
		return ids
	}

	for _, id := range []string{"m-a-1", "m-a-2", "m-a-3"} {
		require.Equal(t, "ドラパルトex", infoOf(id), id)
		require.Equal(t, []string{"0887", "0006"}, spritesOf(id), id)
	}
	// 置き換えないもの: スプライトが違う・削除された記録・他人の対戦
	require.Equal(t, "ドラパ", infoOf("m-a-4"))
	require.Equal(t, []string{"0887"}, spritesOf("m-a-4"))
	require.Equal(t, "ドラパ", infoOf("m-a-deleted-record"))
	require.Equal(t, "ドラパ", infoOf("m-b-1"))
	require.Empty(t, spritesOf("m-b-1"))

	// 既存の組み合わせへ寄せる: スプライト1体の「ドラパ」を、いま作った「ドラパルトex」＋2体へ
	updated, err = r.Replace(ctx, userA,
		entity.NewOpponentDeckSpec("ドラパ", []*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("0887", 1)}),
		entity.NewOpponentDeckSpec("ドラパルトex", []*entity.PokemonSprite{
			entity.NewPokemonSpriteWithPosition("0887", 1),
			entity.NewPokemonSpriteWithPosition("0006", 2),
		}),
	)
	require.NoError(t, err)
	require.Equal(t, 1, updated)

	decks, err = r.FindByUserId(ctx, userA)
	require.NoError(t, err)
	require.Len(t, decks, 2)
	require.Equal(t, "ドラパルトex", decks[0].OpponentsDeckInfo)
	require.Equal(t, 4, decks[0].Count)

	// 2体目だけのスプライトで一致させ、スプライトを外す
	updated, err = r.Replace(ctx, userA,
		entity.NewOpponentDeckSpec("サナ", []*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("0282", 2)}),
		entity.NewOpponentDeckSpec("サーナイトex", nil),
	)
	require.NoError(t, err)
	require.Equal(t, 1, updated)
	require.Equal(t, "サーナイトex", infoOf("m-a-5"))
	require.Empty(t, spritesOf("m-a-5"))

	// 一致するものが無ければ 0 件
	updated, err = r.Replace(ctx, userA, entity.NewOpponentDeckSpec("存在しない表記", nil), entity.NewOpponentDeckSpec("x", nil))
	require.NoError(t, err)
	require.Equal(t, 0, updated)

	// 存在しないスプライトは ErrInvalidReference で、途中まで書き換えた状態を残さない
	_, err = r.Replace(ctx, userA,
		entity.NewOpponentDeckSpec("ドラパルトex", []*entity.PokemonSprite{
			entity.NewPokemonSpriteWithPosition("0887", 1),
			entity.NewPokemonSpriteWithPosition("0006", 2),
		}),
		entity.NewOpponentDeckSpec("壊れた指定", []*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("9999", 1)}),
	)
	require.ErrorIs(t, err, apperror.ErrInvalidReference)
	require.Equal(t, "ドラパルトex", infoOf("m-a-1"))
	require.Equal(t, []string{"0887", "0006"}, spritesOf("m-a-1"))
}
