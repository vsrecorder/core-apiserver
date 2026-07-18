package presenter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func TestNewCalendarGetByUserIdResponse(t *testing.T) {
	t.Parallel()

	now := time.Now().Local()

	t.Run("正常系_空のカレンダーはnilではなく空配列で返す", func(t *testing.T) {
		res := NewCalendarGetByUserIdResponse(&entity.Calendar{})

		require.NotNil(t, res.Records)
		require.Empty(t, res.Records)
		require.NotNil(t, res.Decks)
		require.Empty(t, res.Decks)
		require.NotNil(t, res.OfficialEvents)
		require.NotNil(t, res.TonamelEvents)
		require.NotNil(t, res.UnofficialEvents)
	})

	t.Run("正常系_記録と対戦と対局が入れ子のまま変換される", func(t *testing.T) {
		game := entity.NewGame("game-1", now, "match-1", "user-1", true, true, 0, 6, "メモ")
		match := entity.NewMatch(
			"match-1", now, "record-1", "deck-1", "", "user-1", "",
			false, false, false, false, false, false, true, false,
			"相手デッキ情報", "メモ",
			[]*entity.Game{game},
			[]*entity.PokemonSprite{entity.NewPokemonSprite("pikachu")},
		)
		record := &entity.Record{ID: "record-1", CreatedAt: now, OfficialEventId: 606466, DeckId: "deck-1"}

		calendar := &entity.Calendar{
			Records: []*entity.CalendarRecord{entity.NewCalendarRecord(record, []*entity.Match{match})},
		}

		res := NewCalendarGetByUserIdResponse(calendar)

		require.Len(t, res.Records, 1)
		require.Equal(t, "record-1", res.Records[0].ID)
		require.Equal(t, uint(606466), res.Records[0].OfficialEventId)
		require.Len(t, res.Records[0].Matches, 1)
		require.True(t, res.Records[0].Matches[0].VictoryFlg)
		require.Equal(t, "相手デッキ情報", res.Records[0].Matches[0].OpponentsDeckInfo)
		require.Len(t, res.Records[0].Matches[0].Games, 1)
		require.True(t, res.Records[0].Matches[0].Games[0].GoFirst)
		require.Len(t, res.Records[0].Matches[0].PokemonSprites, 1)
		require.Equal(t, "pikachu", res.Records[0].Matches[0].PokemonSprites[0].ID)
	})

	t.Run("正常系_未アーカイブのデッキはarchived_atがnullになる", func(t *testing.T) {
		deck := entity.NewDeck("deck-1", now, time.Time{}, "user-1", "テストデッキ", false, nil, nil)
		calendar := &entity.Calendar{
			Decks: []*entity.CalendarDeck{entity.NewCalendarDeck(deck, nil)},
		}

		res := NewCalendarGetByUserIdResponse(calendar)

		require.Len(t, res.Decks, 1)
		require.Nil(t, res.Decks[0].ArchivedAt)
	})

	t.Run("正常系_アーカイブ済みデッキはarchived_atが設定される", func(t *testing.T) {
		archivedAt := now.Add(-time.Hour)
		deck := entity.NewDeck("deck-1", now, archivedAt, "user-1", "テストデッキ", false, nil, nil)
		deckCode := entity.NewDeckCode("code-1", now, "user-1", "deck-1", "5dbFbk-uBwjqP-VVk5Vv", false, "")
		calendar := &entity.Calendar{
			Decks: []*entity.CalendarDeck{entity.NewCalendarDeck(deck, []*entity.DeckCode{deckCode})},
		}

		res := NewCalendarGetByUserIdResponse(calendar)

		require.Len(t, res.Decks, 1)
		require.NotNil(t, res.Decks[0].ArchivedAt)
		require.Equal(t, archivedAt, *res.Decks[0].ArchivedAt)
		require.Len(t, res.Decks[0].DeckCodes, 1)
		require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", res.Decks[0].DeckCodes[0].Code)
	})

	t.Run("正常系_公式イベントの開催日はローカル時刻の0時に正規化される", func(t *testing.T) {
		date := time.Date(2026, 7, 18, 10, 30, 0, 0, time.UTC)
		calendar := &entity.Calendar{
			OfficialEvents: []*entity.OfficialEvent{{ID: 606466, Title: "シティリーグ", Date: date}},
		}

		res := NewCalendarGetByUserIdResponse(calendar)

		require.Len(t, res.OfficialEvents, 1)
		require.Equal(t, time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local), res.OfficialEvents[0].Date)
	})

	t.Run("正常系_Tonamelイベントと自由形式イベントはIDとタイトルのみ返す", func(t *testing.T) {
		calendar := &entity.Calendar{
			TonamelEvents:    []*entity.TonamelEvent{{ID: "61ozP", Title: "Tonamel大会"}},
			UnofficialEvents: []*entity.UnofficialEvent{{ID: "unofficial-1", Title: "自主大会"}},
		}

		res := NewCalendarGetByUserIdResponse(calendar)

		require.Len(t, res.TonamelEvents, 1)
		require.Equal(t, "61ozP", res.TonamelEvents[0].ID)
		require.Equal(t, "Tonamel大会", res.TonamelEvents[0].Title)
		require.Len(t, res.UnofficialEvents, 1)
		require.Equal(t, "自主大会", res.UnofficialEvents[0].Title)
	})
}
