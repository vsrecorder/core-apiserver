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

// 組み合わせ 1 つの対戦の一覧を実DBで確認する。イベント名は記録の種類ごとに別のテーブルを
// 外部結合して選ぶので、結合の条件と種類の判定がスキーマと合っているかを見る。
func TestIntegrationOpponentDeckMatches(t *testing.T) {
	db := setupIntegrationDB(t,
		"games", "match_pokemon_sprites", "matches", "records", "decks",
		"unofficial_events", "tonamel_events", "official_events", "pokemon_sprites")

	const (
		userA = "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		userB = "KBp7roRDZobZg1t0OPzFR1kvLeO2"
	)

	now := time.Now().Local().Truncate(time.Microsecond)
	dateOf := func(day int) time.Time { return time.Date(2026, 9, day, 0, 0, 0, 0, time.UTC) }

	require.NoError(t, db.Create(&model.PokemonSprite{ID: "0887", Name: "ドラパルト"}).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO official_events (id, title, address, date) VALUES (?, ?, ?, ?)",
		700001, "シティリーグ 東京", "東京都", dateOf(28)).Error)
	require.NoError(t, db.Create(&model.TonamelEvent{ID: "AbCd1", Title: "Tonamel杯", CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&model.UnofficialEvent{
		ID: "unofficial-1", CreatedAt: now, UpdatedAt: now, UserId: userA, Title: "身内の大会", Date: dateOf(20),
	}).Error)
	require.NoError(t, db.Create(&model.Deck{ID: "deck-a", CreatedAt: now, UpdatedAt: now, UserId: userA, Name: "サーナイト"}).Error)

	createRecord := func(record *model.Record) {
		t.Helper()
		record.CreatedAt, record.UpdatedAt = now, now
		require.NoError(t, db.Create(record).Error)
	}
	createRecord(&model.Record{ID: "rec-official", UserId: userA, OfficialEventId: 700001, DeckId: "deck-a", EventDate: dateOf(28)})
	createRecord(&model.Record{ID: "rec-tonamel", UserId: userA, TonamelEventId: "AbCd1", EventDate: dateOf(25)})
	// 大会名を取得していない Tonamel の記録は、種類だけ分かってタイトルは空
	createRecord(&model.Record{ID: "rec-tonamel-unknown", UserId: userA, TonamelEventId: "ZzZz9", EventDate: dateOf(22)})
	createRecord(&model.Record{ID: "rec-unofficial", UserId: userA, UnofficialEventId: "unofficial-1", EventDate: dateOf(20)})
	createRecord(&model.Record{ID: "rec-b", UserId: userB, OfficialEventId: 700001, EventDate: dateOf(28)})

	createMatch := func(match *model.Match, sprite1 string) {
		t.Helper()
		match.CreatedAt, match.UpdatedAt = now, now
		require.NoError(t, db.Create(match).Error)
		if sprite1 != "" {
			require.NoError(t, db.Create(&model.MatchPokemonSprite{MatchId: match.ID, Position: 1, PokemonSpriteId: sprite1}).Error)
		}
	}
	createGame := func(id string, matchId string, goFirst bool, winning bool) {
		t.Helper()
		require.NoError(t, db.Create(&model.Game{
			ID: id, CreatedAt: now, UpdatedAt: now, MatchId: matchId, UserId: userA, GoFirst: goFirst, WinningFlg: winning,
		}).Error)
		// 対局は作った順に並べる
		now = now.Add(time.Second)
	}

	// 同じ記録の中は対戦の並び順(position)で返す
	createMatch(&model.Match{ID: "m-official-2", RecordId: "rec-official", UserId: userA, OpponentsDeckInfo: "ドラパ", Position: 2, BO3Flg: true, VictoryFlg: true}, "0887")
	createMatch(&model.Match{ID: "m-official-1", RecordId: "rec-official", UserId: userA, OpponentsDeckInfo: "ドラパ", Position: 1}, "0887")
	createMatch(&model.Match{ID: "m-tonamel", RecordId: "rec-tonamel", UserId: userA, OpponentsDeckInfo: "ドラパ", VictoryFlg: true}, "0887")
	createMatch(&model.Match{ID: "m-tonamel-unknown", RecordId: "rec-tonamel-unknown", UserId: userA, OpponentsDeckInfo: "ドラパ"}, "0887")
	createMatch(&model.Match{ID: "m-unofficial", RecordId: "rec-unofficial", UserId: userA, OpponentsDeckInfo: "ドラパ"}, "0887")
	// スプライトが違う(無い)対戦と、他人の対戦は含めない
	createMatch(&model.Match{ID: "m-other-sprite", RecordId: "rec-official", UserId: userA, OpponentsDeckInfo: "ドラパ"}, "")
	createMatch(&model.Match{ID: "m-b", RecordId: "rec-b", UserId: userB, OpponentsDeckInfo: "ドラパ"}, "0887")

	createGame("g-1", "m-official-2", true, true)
	createGame("g-2", "m-official-2", false, false)
	createGame("g-3", "m-official-2", true, true)
	createGame("g-deleted", "m-official-2", true, false)
	require.NoError(t, db.Delete(&model.Game{ID: "g-deleted"}).Error)
	createGame("g-4", "m-tonamel", false, true)

	r := NewOpponentDeck(db)
	matches, err := r.FindMatchesBySpec(context.Background(), userA,
		entity.NewOpponentDeckSpec("ドラパ", []*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("0887", 1)}))
	require.NoError(t, err)

	ids := make([]string, 0, len(matches))
	for _, m := range matches {
		ids = append(ids, m.MatchId)
	}
	require.Equal(t, []string{"m-official-1", "m-official-2", "m-tonamel", "m-tonamel-unknown", "m-unofficial"}, ids)

	official := matches[1]
	require.Equal(t, "rec-official", official.RecordId)
	require.Equal(t, "2026-09-28", official.EventDate)
	require.Equal(t, "official", official.EventType)
	require.Equal(t, "シティリーグ 東京", official.EventTitle)
	require.Equal(t, "サーナイト", official.DeckName)
	require.True(t, official.BO3Flg)
	require.True(t, official.VictoryFlg)
	// 論理削除した対局は含めない
	require.Len(t, official.Games, 3)
	require.True(t, official.Games[0].GoFirst)
	require.False(t, official.Games[1].WinningFlg)
	require.Empty(t, matches[0].Games)

	require.Equal(t, "tonamel", matches[2].EventType)
	require.Equal(t, "Tonamel杯", matches[2].EventTitle)
	require.Equal(t, "", matches[2].DeckName)
	require.Len(t, matches[2].Games, 1)
	require.Equal(t, "tonamel", matches[3].EventType)
	require.Equal(t, "", matches[3].EventTitle)
	require.Equal(t, "unofficial", matches[4].EventType)
	require.Equal(t, "身内の大会", matches[4].EventTitle)
	require.Equal(t, "2026-09-20", matches[4].EventDate)

	// 一致するものが無ければ空
	matches, err = r.FindMatchesBySpec(context.Background(), userA, entity.NewOpponentDeckSpec("存在しない表記", nil))
	require.NoError(t, err)
	require.Empty(t, matches)
}
