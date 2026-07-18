package infrastructure

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestCalendarInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	expectRecordsQuery := func(mock sqlmock.Sqlmock) *sqlmock.ExpectedQuery {
		return mock.ExpectQuery(`SELECT \* FROM "records" WHERE user_id = \$1 AND "records"\."deleted_at" IS NULL ORDER BY created_at ASC`).WithArgs(uid)
	}
	expectMatchesQuery := func(mock sqlmock.Sqlmock) *sqlmock.ExpectedQuery {
		return mock.ExpectQuery(`(?s)SELECT.*matches\.id AS match_id.*FROM "matches".*LEFT JOIN games`).WithArgs(uid)
	}
	expectDecksQuery := func(mock sqlmock.Sqlmock) *sqlmock.ExpectedQuery {
		return mock.ExpectQuery(`SELECT \* FROM "decks" WHERE user_id = \$1 AND "decks"\."deleted_at" IS NULL ORDER BY created_at ASC`).WithArgs(uid)
	}
	expectDeckCodesQuery := func(mock sqlmock.Sqlmock) *sqlmock.ExpectedQuery {
		return mock.ExpectQuery(`SELECT \* FROM "deck_codes" WHERE user_id = \$1 AND "deck_codes"\."deleted_at" IS NULL ORDER BY created_at ASC`).WithArgs(uid)
	}

	recordColumns := []string{"id", "created_at", "user_id", "official_event_id", "unofficial_event_id", "deck_id"}
	matchJoinColumns := []string{"match_id", "match_created_at", "match_record_id", "match_user_id", "match_victory_flg", "game_id", "game_created_at", "game_winning_flg"}
	deckColumns := []string{"id", "created_at", "user_id", "name", "private_flg", "archived_at"}
	deckCodeColumns := []string{"id", "created_at", "user_id", "deck_id", "code", "private_code_flg", "memo"}

	t.Run("正常系_データが無いユーザには空のカレンダーを返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewCalendar(db)

		expectRecordsQuery(mock).WillReturnRows(sqlmock.NewRows(recordColumns))
		expectMatchesQuery(mock).WillReturnRows(sqlmock.NewRows(matchJoinColumns))
		expectDecksQuery(mock).WillReturnRows(sqlmock.NewRows(deckColumns))
		expectDeckCodesQuery(mock).WillReturnRows(sqlmock.NewRows(deckCodeColumns))

		ret, err := r.FindByUserId(context.Background(), uid)

		require.NoError(t, err)
		require.Empty(t, ret.Records)
		require.Empty(t, ret.Decks)
		require.Empty(t, ret.OfficialEvents)
		require.Empty(t, ret.UnofficialEvents)
		require.Empty(t, ret.TonamelEvents)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_記録と対戦とデッキとデッキコードを紐づけて返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewCalendar(db)

		now := time.Now().Local()
		recordId := "01HD7Y3K8D6FDHMHTZ2GT41TR1"
		matchId := "01HD7Y3K8D6FDHMHTZ2GT41TN1"
		gameId := "01HD7Y3K8D6FDHMHTZ2GT41TG1"
		deckId := "01HD7Y3K8D6FDHMHTZ2GT41TD1"
		deckCodeId := "01HD7Y3K8D6FDHMHTZ2GT41TC1"
		unofficialEventId := "01HD7Y3K8D6FDHMHTZ2GT41TU1"
		date := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)

		// 記録は公式イベント参照と自由形式イベント参照の2件
		expectRecordsQuery(mock).WillReturnRows(sqlmock.NewRows(recordColumns).
			AddRow(recordId, now, uid, 606466, "", deckId).
			AddRow("01HD7Y3K8D6FDHMHTZ2GT41TR2", now, uid, 0, unofficialEventId, ""),
		)

		// 対戦は1件(対局1つ・スプライト1つ)
		expectMatchesQuery(mock).WillReturnRows(sqlmock.NewRows(matchJoinColumns).
			AddRow(matchId, now, recordId, uid, true, gameId, now, true),
		)
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WithArgs(matchId).
			WillReturnRows(sqlmock.NewRows(matchPokemonSpriteColumns).AddRow(matchId, 1, "pikachu"))

		// デッキは1件(スプライト1つ)、デッキコードは1件
		expectDecksQuery(mock).WillReturnRows(sqlmock.NewRows(deckColumns).
			AddRow(deckId, now, uid, "テストデッキ", false, sql.NullTime{}),
		)
		mock.ExpectQuery(`SELECT \* FROM "deck_pokemon_sprites" WHERE deck_id IN`).
			WithArgs(deckId).
			WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns).AddRow(deckId, 1, "gardevoir"))
		expectDeckCodesQuery(mock).WillReturnRows(sqlmock.NewRows(deckCodeColumns).
			AddRow(deckCodeId, now, uid, deckId, "5dbFbk-uBwjqP-VVk5Vv", false, ""),
		)

		// 参照されている公式イベント・自由形式イベントだけをまとめて取得する
		mock.ExpectQuery(`SELECT \* FROM "official_events" WHERE id IN`).
			WithArgs(606466).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(606466, "シティリーグ"))
		mock.ExpectQuery(`SELECT \* FROM "unofficial_events" WHERE id IN`).
			WithArgs(unofficialEventId).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "date"}).
				AddRow(unofficialEventId, uid, "自主大会", date))

		ret, err := r.FindByUserId(context.Background(), uid)

		require.NoError(t, err)

		require.Len(t, ret.Records, 2)
		require.Equal(t, recordId, ret.Records[0].Record.ID)
		require.Len(t, ret.Records[0].Matches, 1)
		require.Equal(t, matchId, ret.Records[0].Matches[0].ID)
		require.Len(t, ret.Records[0].Matches[0].Games, 1)
		require.Equal(t, gameId, ret.Records[0].Matches[0].Games[0].ID)
		require.Equal(t, "pikachu", ret.Records[0].Matches[0].PokemonSprites[0].ID)
		require.Empty(t, ret.Records[1].Matches)

		require.Len(t, ret.Decks, 1)
		require.Equal(t, deckId, ret.Decks[0].Deck.ID)
		require.Equal(t, "gardevoir", ret.Decks[0].Deck.PokemonSprites[0].ID)
		require.Len(t, ret.Decks[0].DeckCodes, 1)
		require.Equal(t, deckCodeId, ret.Decks[0].DeckCodes[0].ID)

		require.Len(t, ret.OfficialEvents, 1)
		require.Equal(t, uint(606466), ret.OfficialEvents[0].ID)
		require.Len(t, ret.UnofficialEvents, 1)
		require.Equal(t, "自主大会", ret.UnofficialEvents[0].Title)

		// TonamelイベントはDBに無いためこの層では常に空(usecase層が補完する)
		require.Empty(t, ret.TonamelEvents)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_記録取得のエラーをそのまま返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewCalendar(db)

		expectRecordsQuery(mock).WillReturnError(sql.ErrConnDone)

		ret, err := r.FindByUserId(context.Background(), uid)

		require.Error(t, err)
		require.Nil(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
