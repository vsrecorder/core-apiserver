package infrastructure

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

var weeklyMatchRowColumns = []string{"match_id", "user_id", "deck_id", "victory_flg", "opponents_deck_info"}

func TestWeeklyDeckUsageStatInfrastructure(t *testing.T) {
	fromDate := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)
	toDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local)

	expectWeeklyMatchQuery := func(mock sqlmock.Sqlmock) *sqlmock.ExpectedQuery {
		return mock.ExpectQuery(`SELECT matches\.id AS match_id, records\.user_id AS user_id, records\.deck_id AS deck_id, matches\.victory_flg AS victory_flg, matches\.opponents_deck_info AS opponents_deck_info FROM "matches" JOIN records`).
			WithArgs(fromDate, toDate)
	}

	// 辞書ロード(deck_name_aliases → pokemon_sprites)の期待を積む共通ヘルパー。
	expectMatcherQueries := func(mock sqlmock.Sqlmock, aliasRows *sqlmock.Rows, spriteRows *sqlmock.Rows) {
		mock.ExpectQuery(`SELECT \* FROM "deck_name_aliases" ORDER BY alias ASC, position ASC`).
			WillReturnRows(aliasRows)
		mock.ExpectQuery(`SELECT \* FROM "pokemon_sprites" ORDER BY id ASC`).
			WillReturnRows(spriteRows)
	}

	t.Run("正常系_対象週のマッチが無ければ空の統計を返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		expectWeeklyMatchQuery(mock).WillReturnRows(sqlmock.NewRows(weeklyMatchRowColumns))

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)

		require.NoError(t, err)
		require.Equal(t, fromDate, ret.WeekStart)
		require.Zero(t, ret.TotalVotes)
		require.Zero(t, ret.ContributorCount)
		require.Empty(t, ret.Decks)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 相手デッキの指紋(match_pokemon_sprites)は「記録者が負けた=その指紋が勝った」として票になる
	t.Run("正常系_出現数が閾値未満の変種はその他へ集約する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// 6マッチ(デッキ未登録)。相手指紋Aが5回(うち記録者が3敗=A側3勝)、指紋Bが1回。
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		matchResults := []bool{false, false, false, true, true, true} // 記録者側の勝敗
		for i, victory := range matchResults {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, "", victory, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		spriteRows := sqlmock.NewRows(matchPokemonSpriteColumns)
		for i := 0; i < 5; i++ {
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 1, "pikachu")
		}
		spriteRows = spriteRows.AddRow("match-6", 1, "eevee")
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).WillReturnRows(spriteRows)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)

		require.NoError(t, err)
		require.Equal(t, 6, ret.TotalVotes)
		require.Equal(t, 1, ret.ContributorCount)
		require.Len(t, ret.Decks, 2)

		// 指紋A(pikachu): 5票、記録者が3敗しているのでA側3勝2敗
		require.Equal(t, 5, ret.Decks[0].Count)
		require.Equal(t, 3, ret.Decks[0].Wins)
		require.Equal(t, 2, ret.Decks[0].Losses)
		require.InDelta(t, float64(5)/6, ret.Decks[0].UsageRate, 1e-9)
		require.Len(t, ret.Decks[0].PokemonSprites, 1)
		require.Equal(t, "pikachu", ret.Decks[0].PokemonSprites[0].ID)

		// 指紋B(eevee)は1票で閾値未満のため「その他」に集約される
		require.Equal(t, 1, ret.Decks[1].Count)
		require.Zero(t, ret.Decks[1].Wins)
		require.Empty(t, ret.Decks[1].PokemonSprites)

		// 「その他」には集約した個別変種の内訳(Members)が残り、アコーディオンで一覧表示できる。
		require.Len(t, ret.Decks[1].Members, 1)
		require.Equal(t, 1, ret.Decks[1].Members[0].Count)
		require.Len(t, ret.Decks[1].Members[0].PokemonSprites, 1)
		require.Equal(t, "eevee", ret.Decks[1].Members[0].PokemonSprites[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 自分デッキの指紋(deck_pokemon_sprites)は記録者の勝敗がそのまま票になる
	t.Run("正常系_自分のデッキの指紋も票として集計する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		deckId := "01HD7Y3K8D6FDHMHTZ2GT41TD1"

		// 5マッチとも同じデッキで全勝。相手指紋は未登録のため票にならない。
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 5; i++ {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, deckId, true, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(sqlmock.NewRows(matchPokemonSpriteColumns))
		mock.ExpectQuery(`SELECT \* FROM "deck_pokemon_sprites" WHERE deck_id IN`).
			WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns).AddRow(deckId, 1, "gardevoir"))

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)

		require.NoError(t, err)
		require.Equal(t, 5, ret.TotalVotes)
		require.Len(t, ret.Decks, 1)
		require.Equal(t, 5, ret.Decks[0].Count)
		require.Equal(t, 5, ret.Decks[0].Wins)
		require.InDelta(t, 1.0, ret.Decks[0].WinRate, 1e-9)
		require.Equal(t, "gardevoir", ret.Decks[0].PokemonSprites[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 相手デッキ名も空のときは推測できないため、名前・辞書のクエリを発行せず従来どおり除外する
	// (ExpectationsWereMet で追加クエリが無いことも担保する)
	t.Run("正常系_スプライト未付与でデッキ名も無いマッチは除外する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		expectWeeklyMatchQuery(mock).WillReturnRows(
			sqlmock.NewRows(weeklyMatchRowColumns).AddRow("match-1", "user-1", "", true, ""),
		)
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(sqlmock.NewRows(matchPokemonSpriteColumns))

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)

		require.NoError(t, err)
		require.Zero(t, ret.TotalVotes)
		require.Empty(t, ret.Decks)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// スプライト未設定でも相手デッキ名が辞書にヒットすれば票として救済される
	t.Run("正常系_相手デッキ名からスプライトを推測して集計する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// 5マッチ(デッキ未登録)。記録者が全敗=相手デッキの5勝。スプライトは未設定。
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 5; i++ {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, "", false, "リザードンＥＸ")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(sqlmock.NewRows(matchPokemonSpriteColumns))
		expectMatcherQueries(mock,
			sqlmock.NewRows(deckNameAliasColumns).AddRow("リザ", 1, "0006"),
			sqlmock.NewRows(pokemonSpriteColumns),
		)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)

		require.NoError(t, err)
		require.Equal(t, 5, ret.TotalVotes)
		require.Equal(t, 1, ret.ContributorCount)
		require.Len(t, ret.Decks, 1)
		require.Equal(t, 5, ret.Decks[0].Count)
		require.Equal(t, 5, ret.Decks[0].Wins)
		require.Len(t, ret.Decks[0].PokemonSprites, 1)
		require.Equal(t, "0006", ret.Decks[0].PokemonSprites[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 自分デッキもスプライト未設定なら decks.name からの推測で救済される(代表2体)
	t.Run("正常系_自分のデッキ名からスプライトを推測して集計する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		deckId := "01HD7Y3K8D6FDHMHTZ2GT41TD1"

		// 5マッチとも同じデッキで全勝。相手情報なし。デッキのスプライトは未設定。
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 5; i++ {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, deckId, true, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(sqlmock.NewRows(matchPokemonSpriteColumns))
		mock.ExpectQuery(`SELECT \* FROM "deck_pokemon_sprites" WHERE deck_id IN`).
			WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns))
		mock.ExpectQuery(`SELECT id, name FROM "decks" WHERE id IN`).
			WithArgs(deckId).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(deckId, "ロスバレ"))
		expectMatcherQueries(mock,
			sqlmock.NewRows(deckNameAliasColumns).
				AddRow("ロスバレ", 1, "0487_origin").
				AddRow("ロスバレ", 2, "0225"),
			sqlmock.NewRows(pokemonSpriteColumns),
		)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)

		require.NoError(t, err)
		require.Equal(t, 5, ret.TotalVotes)
		require.Len(t, ret.Decks, 1)
		require.Equal(t, 5, ret.Decks[0].Count)
		require.Equal(t, 5, ret.Decks[0].Wins)
		require.Len(t, ret.Decks[0].PokemonSprites, 2)
		require.Equal(t, "0487_origin", ret.Decks[0].PokemonSprites[0].ID)
		require.Equal(t, "0225", ret.Decks[0].PokemonSprites[1].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 辞書にヒットしない名前は従来どおり除外される(その他にも入らない)
	t.Run("正常系_デッキ名が辞書にヒットしなければ除外する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		expectWeeklyMatchQuery(mock).WillReturnRows(
			sqlmock.NewRows(weeklyMatchRowColumns).AddRow("match-1", "user-1", "", true, "謎のデッキ"),
		)
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(sqlmock.NewRows(matchPokemonSpriteColumns))
		expectMatcherQueries(mock,
			sqlmock.NewRows(deckNameAliasColumns).AddRow("リザ", 1, "0006"),
			sqlmock.NewRows(pokemonSpriteColumns),
		)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)

		require.NoError(t, err)
		require.Zero(t, ret.TotalVotes)
		require.Empty(t, ret.Decks)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 推測(1体)と実スプライト(2体)は指紋が異なるため別変種として集計される
	t.Run("正常系_推測1体と実スプライト2体は別変種になる", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// 前半5マッチは実スプライト(0006+0018)、後半5マッチは未設定+名前「リザードン」
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 5; i++ {
			rows = rows.AddRow("real-"+string(rune('1'+i)), uid, "", false, "")
		}
		for i := 0; i < 5; i++ {
			rows = rows.AddRow("guess-"+string(rune('1'+i)), uid, "", false, "リザードン")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		spriteRows := sqlmock.NewRows(matchPokemonSpriteColumns)
		for i := 0; i < 5; i++ {
			spriteRows = spriteRows.AddRow("real-"+string(rune('1'+i)), 1, "0006")
			spriteRows = spriteRows.AddRow("real-"+string(rune('1'+i)), 2, "0018")
		}
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(spriteRows)
		// エイリアス辞書は空でも、正式名(pokemon_sprites.name)が突合対象になる
		expectMatcherQueries(mock,
			sqlmock.NewRows(deckNameAliasColumns),
			sqlmock.NewRows(pokemonSpriteColumns).AddRow("0006", "リザードン"),
		)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)

		require.NoError(t, err)
		require.Equal(t, 10, ret.TotalVotes)
		require.Len(t, ret.Decks, 2)
		require.NotEqual(t, ret.Decks[0].Fingerprint, ret.Decks[1].Fingerprint)
		require.Len(t, ret.Decks[0].PokemonSprites, 2)
		require.Len(t, ret.Decks[1].PokemonSprites, 1)
		require.Equal(t, "0006", ret.Decks[1].PokemonSprites[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_辞書取得のエラーをそのまま返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		expectWeeklyMatchQuery(mock).WillReturnRows(
			sqlmock.NewRows(weeklyMatchRowColumns).AddRow("match-1", "user-1", "", true, "リザ"),
		)
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(sqlmock.NewRows(matchPokemonSpriteColumns))
		mock.ExpectQuery(`SELECT \* FROM "deck_name_aliases"`).WillReturnError(sql.ErrConnDone)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)

		require.Error(t, err)
		require.Nil(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 表示スロットは position 1/2 の2枠のみ。3体目以降は画面に現れないのに指紋だけを
	// 分けて「見た目が同じ行」が並ぶ原因になるため、指紋にも表示にも含めない。
	t.Run("正常系_3体目以降のスプライトは指紋に含めず2体の変種に合流する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// 5マッチ(デッキ未登録)。前半3マッチは2体登録、後半2マッチは同じ2体+3体目。
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 5; i++ {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, "", false, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		spriteRows := sqlmock.NewRows(matchPokemonSpriteColumns)
		for i := 0; i < 3; i++ {
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 1, "0006")
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 2, "0018")
		}
		for i := 3; i < 5; i++ {
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 1, "0006")
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 2, "0018")
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 3, "0400")
		}
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(spriteRows)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)

		require.NoError(t, err)
		require.Equal(t, 5, ret.TotalVotes)
		// 3体目(0400)は指紋に含まれず、1つの変種(0006,0018)に合流する
		require.Len(t, ret.Decks, 1)
		require.Equal(t, 5, ret.Decks[0].Count)
		require.Len(t, ret.Decks[0].PokemonSprites, 2)
		require.Equal(t, "0006", ret.Decks[0].PokemonSprites[0].ID)
		require.Equal(t, "0018", ret.Decks[0].PokemonSprites[1].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_マッチ取得のエラーをそのまま返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		expectWeeklyMatchQuery(mock).WillReturnError(sql.ErrConnDone)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)

		require.Error(t, err)
		require.Nil(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
