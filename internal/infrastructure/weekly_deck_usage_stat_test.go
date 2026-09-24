package infrastructure

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

var weeklyMatchRowColumns = []string{"match_id", "user_id", "deck_id", "victory_flg", "opponents_deck_info"}

func TestWeeklyDeckUsageStatInfrastructure(t *testing.T) {
	fromDate := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)
	toDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local)

	const weeklyMatchQueryPattern = `SELECT matches\.id AS match_id, records\.user_id AS user_id, records\.deck_id AS deck_id, matches\.victory_flg AS victory_flg, matches\.draw_flg AS draw_flg, matches\.opponents_deck_info AS opponents_deck_info FROM "matches" JOIN records`

	// スタンダード(regulation_id)の記録だけを集計するため、期間の前に
	// レギュレーションが引数として渡る。
	expectWeeklyMatchQuery := func(mock sqlmock.Sqlmock) *sqlmock.ExpectedQuery {
		return mock.ExpectQuery(weeklyMatchQueryPattern).
			WithArgs(int(entity.RegulationIdStandard), fromDate, toDate)
	}

	// 変種が1件でもあると前週 [from-7d, from) の比較集計が走る。
	prevFromDate := fromDate.AddDate(0, 0, -7)
	expectPrevWeekQuery := func(mock sqlmock.Sqlmock) *sqlmock.ExpectedQuery {
		return mock.ExpectQuery(weeklyMatchQueryPattern).
			WithArgs(int(entity.RegulationIdStandard), prevFromDate, fromDate)
	}
	// 前週にデータが無いケースの共通形(比較値はすべて nil になる)。
	expectPrevWeekEmpty := func(mock sqlmock.Sqlmock) {
		expectPrevWeekQuery(mock).WillReturnRows(sqlmock.NewRows(weeklyMatchRowColumns))
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

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

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
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

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
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

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

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

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
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

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
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

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

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

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
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

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

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

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
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

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

	// 前週を同じ規則で集計し、指紋(スプライトの組み合わせ)で突き合わせて前週の
	// 順位・使用率・勝率を付与する。前週「その他」の内訳にいた変種も NEW にしない。
	t.Run("正常系_前週の順位と使用率勝率を変種に付与する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// 今週: A(0006)3票全勝、B(0025)3票全敗、E(0150)3票全敗 → A/B/E の順
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 3; i++ {
			rows = rows.AddRow("cur-a-"+string(rune('1'+i)), uid, "", false, "")
		}
		for i := 0; i < 3; i++ {
			rows = rows.AddRow("cur-b-"+string(rune('1'+i)), uid, "", true, "")
		}
		for i := 0; i < 3; i++ {
			rows = rows.AddRow("cur-e-"+string(rune('1'+i)), uid, "", true, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		curSprites := sqlmock.NewRows(matchPokemonSpriteColumns)
		for i := 0; i < 3; i++ {
			curSprites = curSprites.AddRow("cur-a-"+string(rune('1'+i)), 1, "0006")
		}
		for i := 0; i < 3; i++ {
			curSprites = curSprites.AddRow("cur-b-"+string(rune('1'+i)), 1, "0025")
		}
		for i := 0; i < 3; i++ {
			curSprites = curSprites.AddRow("cur-e-"+string(rune('1'+i)), 1, "0150")
		}
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(curSprites)

		// 前週: C(0018)4票で1位、A(0006)3票全勝で2位、B(0025)2票は閾値未満で
		// 「その他」の内訳(3番目の連番)に表示されていた。E は前週に存在しない。
		prevRows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 4; i++ {
			prevRows = prevRows.AddRow("prev-c-"+string(rune('1'+i)), uid, "", true, "")
		}
		for i := 0; i < 3; i++ {
			prevRows = prevRows.AddRow("prev-a-"+string(rune('1'+i)), uid, "", false, "")
		}
		for i := 0; i < 2; i++ {
			prevRows = prevRows.AddRow("prev-b-"+string(rune('1'+i)), uid, "", true, "")
		}
		expectPrevWeekQuery(mock).WillReturnRows(prevRows)

		prevSprites := sqlmock.NewRows(matchPokemonSpriteColumns)
		for i := 0; i < 4; i++ {
			prevSprites = prevSprites.AddRow("prev-c-"+string(rune('1'+i)), 1, "0018")
		}
		for i := 0; i < 3; i++ {
			prevSprites = prevSprites.AddRow("prev-a-"+string(rune('1'+i)), 1, "0006")
		}
		for i := 0; i < 2; i++ {
			prevSprites = prevSprites.AddRow("prev-b-"+string(rune('1'+i)), 1, "0025")
		}
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(prevSprites)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

		require.NoError(t, err)
		require.Len(t, ret.Decks, 3)

		// 変種A: 前週2位(4票のCに次ぐ) → 今週1位。使用率 3/9、勝率 1.0 が前週値として付く。
		a := ret.Decks[0]
		require.Equal(t, "0006", a.Fingerprint)
		require.NotNil(t, a.PreviousRank)
		require.Equal(t, 2, *a.PreviousRank)
		require.NotNil(t, a.PreviousUsageRate)
		require.InDelta(t, float64(3)/9, *a.PreviousUsageRate, 1e-9)
		require.NotNil(t, a.PreviousWinRate)
		require.InDelta(t, 1.0, *a.PreviousWinRate, 1e-9)

		// 変種B: 前週は「その他」の内訳にいた → NEW ではなく内訳の連番(3位)を引き継ぐ
		b := ret.Decks[1]
		require.Equal(t, "0025", b.Fingerprint)
		require.NotNil(t, b.PreviousRank)
		require.Equal(t, 3, *b.PreviousRank)
		require.NotNil(t, b.PreviousUsageRate)
		require.InDelta(t, float64(2)/9, *b.PreviousUsageRate, 1e-9)
		require.NotNil(t, b.PreviousWinRate)
		require.InDelta(t, 0.0, *b.PreviousWinRate, 1e-9)

		// 変種E: 前週に一度も現れていないので比較値は付かない(NEW)
		e := ret.Decks[2]
		require.Equal(t, "0150", e.Fingerprint)
		require.Nil(t, e.PreviousRank)
		require.Nil(t, e.PreviousUsageRate)
		require.Nil(t, e.PreviousWinRate)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 不戦勝/不戦敗（default_victory_flg / default_defeat_flg）は対戦が行われていないため
	// SQL の時点で除外する。相手側の票は指紋が空で元から落ちるが、自分側の票
	// （records.deck_id の指紋）はこの条件が無いと使用数・勝敗へ混入する。
	t.Run("正常系_不戦勝と不戦敗はクエリの時点で除外する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		mock.ExpectQuery(`matches\.default_victory_flg = false AND matches\.default_defeat_flg = false`).
			WithArgs(int(entity.RegulationIdStandard), fromDate, toDate).
			WillReturnRows(sqlmock.NewRows(weeklyMatchRowColumns))

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

		require.NoError(t, err)
		require.Zero(t, ret.TotalVotes)
		require.Empty(t, ret.Decks)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 1体目でまとめる集計(DeckUsageGroupingFirstSprite)。2体目が違うだけの派生を
	// 1行に束ね、指紋も表示も1体目のスプライトだけになる。
	t.Run("正常系_1体目でまとめる集計は2体目が違う変種を同じデッキとして数える", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// 4マッチ(デッキ未登録)。相手は全て1体目が 0006 で、2体目だけが分かれる。
		// 組み合わせ単位なら 0006+0018 が3票・0006+0157 が1票（後者は閾値未満で「その他」）。
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 4; i++ {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, "", false, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		spriteRows := sqlmock.NewRows(matchPokemonSpriteColumns)
		for i := 0; i < 3; i++ {
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 1, "0006")
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 2, "0018")
		}
		spriteRows = spriteRows.AddRow("match-4", 1, "0006")
		spriteRows = spriteRows.AddRow("match-4", 2, "0157")
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(spriteRows)
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(
			context.Background(), fromDate, toDate, entity.DeckUsageGroupingFirstSprite,
		)

		require.NoError(t, err)
		require.Equal(t, entity.DeckUsageGroupingFirstSprite, ret.Grouping)
		require.Equal(t, 4, ret.TotalVotes)

		// 4票がすべて「1体目が 0006」の1行にまとまる(「その他」へ落ちる変種は無い)。
		require.Len(t, ret.Decks, 1)
		require.Equal(t, "0006", ret.Decks[0].Fingerprint)
		require.Equal(t, 4, ret.Decks[0].Count)
		// 記録者が4敗 = 相手側の4勝
		require.Equal(t, 4, ret.Decks[0].Wins)

		// 表示も1体目だけ。2体目(0018/0157)は行の代表として出さない。
		require.Len(t, ret.Decks[0].PokemonSprites, 1)
		require.Equal(t, "0006", ret.Decks[0].PokemonSprites[0].ID)
		require.Equal(t, uint(1), ret.Decks[0].PokemonSprites[0].Position)

		// 束ねる前の組み合わせは内訳(Members)として残り、UI のアコーディオンで
		// 一覧表示できる。並びは件数の降順で、2体目も含めて表示できる。
		require.Len(t, ret.Decks[0].Members, 2)

		require.Equal(t, 3, ret.Decks[0].Members[0].Count)
		require.Equal(t, 0.75, ret.Decks[0].Members[0].UsageRate)
		require.Len(t, ret.Decks[0].Members[0].PokemonSprites, 2)
		require.Equal(t, "0006", ret.Decks[0].Members[0].PokemonSprites[0].ID)
		require.Equal(t, "0018", ret.Decks[0].Members[0].PokemonSprites[1].ID)
		require.Equal(t, uint(2), ret.Decks[0].Members[0].PokemonSprites[1].Position)

		require.Equal(t, 1, ret.Decks[0].Members[1].Count)
		require.Equal(t, "0157", ret.Decks[0].Members[1].PokemonSprites[1].ID)

		// 内訳の件数の合計は行の件数に一致する(取りこぼしも二重計上も無い)。
		require.Equal(t, ret.Decks[0].Count, ret.Decks[0].Members[0].Count+ret.Decks[0].Members[1].Count)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 組み合わせ一致の集計では、行そのものが組み合わせ単位のため内訳を持たない
	// (「その他」行だけが集約した変種の内訳を持つ)。
	t.Run("正常系_組み合わせ一致の集計では個別表示の行は内訳を持たない", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 3; i++ {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, "", false, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		spriteRows := sqlmock.NewRows(matchPokemonSpriteColumns)
		for i := 0; i < 3; i++ {
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 1, "0006")
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 2, "0018")
		}
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(spriteRows)
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(
			context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact,
		)

		require.NoError(t, err)
		require.Len(t, ret.Decks, 1)
		require.Empty(t, ret.Decks[0].Members)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 1体目でまとめた変種が「その他」へ落ちる場合、その他の内訳(集約された変種)は
	// さらにその内訳(組み合わせ単位)を持たない。アコーディオンを二段に畳まないため。
	t.Run("正常系_その他へ集約された内訳も組み合わせの内訳を持つ", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// 1体目が 0006 の票が3つ(個別表示)、1体目が 0025 の票が1つ(「その他」へ集約)。
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 4; i++ {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, "", false, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		spriteRows := sqlmock.NewRows(matchPokemonSpriteColumns)
		for i := 0; i < 3; i++ {
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 1, "0006")
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 2, "0018")
		}
		spriteRows = spriteRows.AddRow("match-4", 1, "0025")
		spriteRows = spriteRows.AddRow("match-4", 2, "0157")
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(spriteRows)
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(
			context.Background(), fromDate, toDate, entity.DeckUsageGroupingFirstSprite,
		)

		require.NoError(t, err)
		require.Len(t, ret.Decks, 2)

		// 個別表示された行は組み合わせの内訳を持つ。
		require.Equal(t, "0006", ret.Decks[0].Fingerprint)
		require.Len(t, ret.Decks[0].Members, 1)

		// 「その他」の内訳は1体目でまとめた変種のまま。ここで消えると、その行が何と
		// 組んだデッキだったのかを追う手段が無くなるため、組み合わせの内訳も残す。
		require.Equal(t, "", ret.Decks[1].Fingerprint)
		require.Len(t, ret.Decks[1].Members, 1)
		require.Equal(t, "0025", ret.Decks[1].Members[0].Fingerprint)

		require.Len(t, ret.Decks[1].Members[0].Members, 1)
		combination := ret.Decks[1].Members[0].Members[0]
		require.Equal(t, "0025,0157", combination.Fingerprint)
		require.Equal(t, 1, combination.Count)
		require.Len(t, combination.PokemonSprites, 2)
		require.Equal(t, "0025", combination.PokemonSprites[0].ID)
		require.Equal(t, "0157", combination.PokemonSprites[1].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 1枠目が欠けて2枠目にだけスプライトが入っている票(旧データ)も、先頭のスプライトを
	// 1体目として扱って集計する。position==1 で抜き出すと指紋を作れず丸ごと落ちてしまう。
	t.Run("正常系_1体目でまとめる集計は2枠目だけの票も先頭のスプライトで数える", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 3; i++ {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, "", true, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		spriteRows := sqlmock.NewRows(matchPokemonSpriteColumns)
		for i := 0; i < 3; i++ {
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 2, "0006")
		}
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(spriteRows)
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(
			context.Background(), fromDate, toDate, entity.DeckUsageGroupingFirstSprite,
		)

		require.NoError(t, err)
		require.Len(t, ret.Decks, 1)
		require.Equal(t, "0006", ret.Decks[0].Fingerprint)
		require.Equal(t, 3, ret.Decks[0].Count)
		// 表示スロットは1枠目へ寄せる(この集計単位では「1体目が○○」の行のため)。
		require.Len(t, ret.Decks[0].PokemonSprites, 1)
		require.Equal(t, uint(1), ret.Decks[0].PokemonSprites[0].Position)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 未知の集計単位でレポートごと落とさず、既定(組み合わせ一致)へ寄せる。
	t.Run("正常系_未知の集計単位は組み合わせ一致として扱う", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		expectWeeklyMatchQuery(mock).WillReturnRows(sqlmock.NewRows(weeklyMatchRowColumns))

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, "unknown")

		require.NoError(t, err)
		require.Equal(t, entity.DeckUsageGroupingExact, ret.Grouping)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 同じ組み合わせでも、1体目と2体目を入れ替えて登録した票が混ざる。
	// 行に出すアイコンの並びは「最初に来た票」ではなく、多数派の並びに合わせる。
	t.Run("正常系_組み合わせが同じで並びが違う票は多数派の並びで表示する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// 4マッチ(デッキ未登録)。先頭の1票だけ 0018 が1体目で、残り3票は 0006 が1体目。
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 4; i++ {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, "", false, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		spriteRows := sqlmock.NewRows(matchPokemonSpriteColumns)
		spriteRows = spriteRows.AddRow("match-1", 1, "0018")
		spriteRows = spriteRows.AddRow("match-1", 2, "0006")
		for i := 1; i < 4; i++ {
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 1, "0006")
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 2, "0018")
		}
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(spriteRows)
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

		require.NoError(t, err)
		// 指紋は元から順序非依存なので、並びが違っても1行に集まる。
		require.Len(t, ret.Decks, 1)
		require.Equal(t, 4, ret.Decks[0].Count)

		// 表示は多数派(3票)の並び。少数派が先に集計されても引きずられない。
		require.Len(t, ret.Decks[0].PokemonSprites, 2)
		require.Equal(t, "0006", ret.Decks[0].PokemonSprites[0].ID)
		require.Equal(t, uint(1), ret.Decks[0].PokemonSprites[0].Position)
		require.Equal(t, "0018", ret.Decks[0].PokemonSprites[1].ID)
		require.Equal(t, uint(2), ret.Decks[0].PokemonSprites[1].Position)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 1体目でまとめる集計では、並びの違いが行そのものを割ってしまう。
	// 少数派の並びの票も、その組み合わせで多数派の1体目の行へまとめる。
	t.Run("正常系_1体目でまとめる集計は並びが逆の票も多数派の行へまとめる", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// 5マッチ(デッキ未登録)。0006+0018 が4票で、うち1票だけ 0018 が1体目。
		// 残り1票は 0006+0157(0006 が1体目)。
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 5; i++ {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, "", false, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		spriteRows := sqlmock.NewRows(matchPokemonSpriteColumns)
		spriteRows = spriteRows.AddRow("match-1", 1, "0018")
		spriteRows = spriteRows.AddRow("match-1", 2, "0006")
		for i := 1; i < 4; i++ {
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 1, "0006")
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 2, "0018")
		}
		spriteRows = spriteRows.AddRow("match-5", 1, "0006")
		spriteRows = spriteRows.AddRow("match-5", 2, "0157")
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(spriteRows)
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(
			context.Background(), fromDate, toDate, entity.DeckUsageGroupingFirstSprite,
		)

		require.NoError(t, err)
		require.Equal(t, 5, ret.TotalVotes)

		// 0018 を1体目にした票も 0006 の行に入る(0018 の行はできない)。
		require.Len(t, ret.Decks, 1)
		require.Equal(t, "0006", ret.Decks[0].Fingerprint)
		require.Equal(t, 5, ret.Decks[0].Count)

		// 内訳も並びを揃えた組み合わせ単位。0006+0018 は逆順の票を含めて4票。
		require.Len(t, ret.Decks[0].Members, 2)
		require.Equal(t, "0006,0018", ret.Decks[0].Members[0].Fingerprint)
		require.Equal(t, 4, ret.Decks[0].Members[0].Count)
		require.Equal(t, "0006", ret.Decks[0].Members[0].PokemonSprites[0].ID)
		require.Equal(t, "0018", ret.Decks[0].Members[0].PokemonSprites[1].ID)
		require.Equal(t, 1, ret.Decks[0].Members[1].Count)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 並びが半々のときに票の到着順で代表が決まると、同じ週を集計し直すたびに
	// 表示が変わりかねない。票数が同じならスプライトIDの順で決める。
	t.Run("正常系_並びの票数が同じときは決まった順序で代表を選ぶ", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// 4マッチ(デッキ未登録)。0018 が1体目の票と 0006 が1体目の票が2票ずつ。
		rows := sqlmock.NewRows(weeklyMatchRowColumns)
		for i := 0; i < 4; i++ {
			rows = rows.AddRow("match-"+string(rune('1'+i)), uid, "", false, "")
		}
		expectWeeklyMatchQuery(mock).WillReturnRows(rows)

		spriteRows := sqlmock.NewRows(matchPokemonSpriteColumns)
		for i := 0; i < 2; i++ {
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 1, "0018")
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 2, "0006")
		}
		for i := 2; i < 4; i++ {
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 1, "0006")
			spriteRows = spriteRows.AddRow("match-"+string(rune('1'+i)), 2, "0018")
		}
		mock.ExpectQuery(`SELECT \* FROM "match_pokemon_sprites" WHERE match_id IN`).
			WillReturnRows(spriteRows)
		expectPrevWeekEmpty(mock)

		ret, err := r.FindWeeklyDeckUsageStat(
			context.Background(), fromDate, toDate, entity.DeckUsageGroupingFirstSprite,
		)

		require.NoError(t, err)
		require.Len(t, ret.Decks, 1)
		require.Equal(t, "0006", ret.Decks[0].Fingerprint)
		require.Equal(t, 4, ret.Decks[0].Count)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_マッチ取得のエラーをそのまま返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewWeeklyDeckUsageStat(db)

		expectWeeklyMatchQuery(mock).WillReturnError(sql.ErrConnDone)

		ret, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact)

		require.Error(t, err)
		require.Nil(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
