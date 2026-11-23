package infrastructure

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func setup4DeckUsageStatInfrastructure() (repository.DeckUsageStatInterface, sqlmock.Sqlmock, error) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}

	db, err := gorm.Open(
		postgres.New(postgres.Config{
			Conn: mockDB,
		}),
		&gorm.Config{},
	)
	if err != nil {
		return nil, nil, err
	}

	return NewDeckUsageStat(db), mock, nil
}

const deckUsageStatQuery = `SELECT records.deck_id AS deck_id, COALESCE(decks.name, '') AS name, COUNT(DISTINCT matches.id) AS count, COUNT(DISTINCT CASE WHEN matches.victory_flg THEN matches.id END) AS wins, COUNT(DISTINCT CASE WHEN matches.draw_flg THEN matches.id END) AS draws, COUNT(games.id) AS game_count, SUM(CASE WHEN games.go_first THEN 1 ELSE 0 END) AS go_first_count, SUM(CASE WHEN games.go_first AND games.winning_flg THEN 1 ELSE 0 END) AS go_first_wins, SUM(CASE WHEN games.go_first = false AND games.winning_flg THEN 1 ELSE 0 END) AS go_second_wins FROM "matches" JOIN records ON matches.record_id = records.id LEFT JOIN decks ON records.deck_id = decks.id LEFT JOIN games ON games.match_id = matches.id AND games.deleted_at IS NULL WHERE records.user_id = $1 AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND records.deck_id != '' GROUP BY records.deck_id, decks.name ORDER BY count DESC`

const deckUsageIgnoredQuery = `SELECT records.deck_id AS deck_id, COALESCE(decks.name, '') AS name, COUNT(DISTINCT records.id) AS ignored_count FROM "records" LEFT JOIN decks ON records.deck_id = decks.id WHERE records.user_id = $1 AND records.deleted_at IS NULL AND records.ignore_stats_flg = true AND records.deck_id != '' GROUP BY records.deck_id, decks.name ORDER BY ignored_count DESC`

var deckUsageIgnoredColumns = []string{"deck_id", "name", "ignored_count"}

// 除外した不戦勝/不戦敗の件数をデッキごとに数えるクエリ(excludeDefaultMatches のときだけ実行される)
const deckUsageDefaultMatchQuery = `SELECT records.deck_id AS deck_id, COALESCE(decks.name, '') AS name, COUNT(DISTINCT matches.id) AS default_match_count FROM "matches" JOIN records ON matches.record_id = records.id LEFT JOIN decks ON records.deck_id = decks.id WHERE records.user_id = $1 AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND records.deck_id != '' AND (matches.default_victory_flg = true OR matches.default_defeat_flg = true) GROUP BY records.deck_id, decks.name ORDER BY default_match_count DESC`

var deckUsageDefaultMatchColumns = []string{"deck_id", "name", "default_match_count"}

// 行のアイコンを選ぶための、デッキ×バージョンごとの対戦数のクエリ(集計対象の対戦があるときだけ実行される)
const deckUsageVersionCountQueryPattern = `SELECT records\.deck_id AS deck_id, records\.deck_code_id AS deck_code_id, COUNT\(DISTINCT matches\.id\) AS count FROM "matches" JOIN records`

var deckUsageVersionCountColumns = []string{"deck_id", "deck_code_id", "count"}

// バージョンが紐付いた対戦が無い(行のアイコンはデッキのスプライトのまま)ときの期待
func expectNoDeckVersionCounts(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(deckUsageVersionCountQueryPattern).
		WillReturnRows(sqlmock.NewRows(deckUsageVersionCountColumns))
}

func TestDeckUsageStatInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"AggregatesWinsAndGoFirstCountsPerDeck": test_DeckUsageStatInfrastructure_AggregatesWinsAndGoFirstCountsPerDeck,
		"NoMatches":                             test_DeckUsageStatInfrastructure_NoMatches,
		"IncludesIgnoredCounts":                 test_DeckUsageStatInfrastructure_IncludesIgnoredCounts,
		"DrawsAreNotCountedAsLosses":            test_DeckUsageStatInfrastructure_DrawsAreNotCountedAsLosses,
		"ExcludesDefaultMatches":                test_DeckUsageStatInfrastructure_ExcludesDefaultMatches,
		"DefaultOnlyDeckRemains":                test_DeckUsageStatInfrastructure_DefaultOnlyDeckRemains,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_DeckUsageStatInfrastructure_AggregatesWinsAndGoFirstCountsPerDeck(t *testing.T) {
	i, mock, err := setup4DeckUsageStatInfrastructure()
	require.NoError(t, err)

	userId := "user-01"

	deckRows := sqlmock.NewRows(
		[]string{"deck_id", "name", "count", "wins", "game_count", "go_first_count", "go_first_wins", "go_second_wins"},
	).AddRow("deck-01", "リザードンex", 3, 2, 5, 3, 2, 0)

	mock.ExpectQuery(regexp.QuoteMeta(deckUsageStatQuery)).
		WithArgs(userId).
		WillReturnRows(deckRows)

	// 集計対象外の記録は無い想定（空を返す）
	mock.ExpectQuery(regexp.QuoteMeta(deckUsageIgnoredQuery)).
		WithArgs(userId).
		WillReturnRows(sqlmock.NewRows(deckUsageIgnoredColumns))

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1) ORDER BY position ASC`,
	)).WithArgs("deck-01").WillReturnRows(sqlmock.NewRows([]string{"deck_id", "position", "pokemon_sprite_id"}))
	expectNoDeckVersionCounts(mock)

	stat, err := i.FindDeckUsageStat(context.Background(), userId, repository.StatPeriod{}, 0, false)

	require.NoError(t, err)
	require.Equal(t, 3, stat.TotalRecords)
	require.Len(t, stat.Decks, 1)

	deck := stat.Decks[0]
	require.Equal(t, "deck-01", deck.DeckId)
	require.Equal(t, 3, deck.Count)
	require.Equal(t, 0, deck.IgnoredCount)
	require.Equal(t, 2, deck.Wins)
	require.Equal(t, 1, deck.Losses)
	require.InDelta(t, 2.0/3.0, deck.WinRate, 0.0001)
	require.Equal(t, 5, deck.GameCount)
	require.Equal(t, 3, deck.GoFirstCount)
	require.Equal(t, 2, deck.GoSecondCount)
	require.InDelta(t, 3.0/5.0, deck.GoFirstRate, 0.0001)
	require.Equal(t, 2, deck.GoFirstWins)
	require.InDelta(t, 2.0/3.0, deck.GoFirstWinRate, 0.0001)
	require.Equal(t, 0, deck.GoSecondWins)
	require.InDelta(t, 0.0, deck.GoSecondWinRate, 0.0001)
}

// 引き分けは負けに数えず、勝率の分母からも除外されることを検証する。
// count=4, wins=2, draws=1 → losses=1, winRate=2/(2+1)。
func test_DeckUsageStatInfrastructure_DrawsAreNotCountedAsLosses(t *testing.T) {
	i, mock, err := setup4DeckUsageStatInfrastructure()
	require.NoError(t, err)

	userId := "user-01"

	deckRows := sqlmock.NewRows(
		[]string{"deck_id", "name", "count", "wins", "draws", "game_count", "go_first_count", "go_first_wins", "go_second_wins"},
	).AddRow("deck-01", "リザードンex", 4, 2, 1, 6, 3, 2, 0)

	mock.ExpectQuery(regexp.QuoteMeta(deckUsageStatQuery)).
		WithArgs(userId).
		WillReturnRows(deckRows)

	mock.ExpectQuery(regexp.QuoteMeta(deckUsageIgnoredQuery)).
		WithArgs(userId).
		WillReturnRows(sqlmock.NewRows(deckUsageIgnoredColumns))

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1) ORDER BY position ASC`,
	)).WithArgs("deck-01").WillReturnRows(sqlmock.NewRows([]string{"deck_id", "position", "pokemon_sprite_id"}))
	expectNoDeckVersionCounts(mock)

	stat, err := i.FindDeckUsageStat(context.Background(), userId, repository.StatPeriod{}, 0, false)

	require.NoError(t, err)
	require.Len(t, stat.Decks, 1)

	deck := stat.Decks[0]
	require.Equal(t, 4, deck.Count)
	require.Equal(t, 2, deck.Wins)
	require.Equal(t, 1, deck.Losses, "引き分け1件が負けに数えられていないこと(4-2-1=1)")
	require.InDelta(t, 2.0/3.0, deck.WinRate, 0.0001, "勝率は引き分けを分母から除外する(2/(2+1))")
}

func test_DeckUsageStatInfrastructure_NoMatches(t *testing.T) {
	i, mock, err := setup4DeckUsageStatInfrastructure()
	require.NoError(t, err)

	userId := "user-01"

	mock.ExpectQuery(regexp.QuoteMeta(deckUsageStatQuery)).
		WithArgs(userId).
		WillReturnRows(sqlmock.NewRows(
			[]string{"deck_id", "name", "count", "wins", "game_count", "go_first_count", "go_first_wins", "go_second_wins"},
		))

	mock.ExpectQuery(regexp.QuoteMeta(deckUsageIgnoredQuery)).
		WithArgs(userId).
		WillReturnRows(sqlmock.NewRows(deckUsageIgnoredColumns))

	stat, err := i.FindDeckUsageStat(context.Background(), userId, repository.StatPeriod{}, 0, false)

	require.NoError(t, err)
	require.Equal(t, 0, stat.TotalRecords)
	require.Len(t, stat.Decks, 0)
}

// 集計対象と集計対象外が混在するケース。
// - deck-01: 集計対象の対戦あり + 集計対象外の記録2件 → IgnoredCount=2 が付与される
// - deck-02: 集計対象外の記録のみ1件 → count=0 のデッキとして追加される（all_time時）
func test_DeckUsageStatInfrastructure_IncludesIgnoredCounts(t *testing.T) {
	i, mock, err := setup4DeckUsageStatInfrastructure()
	require.NoError(t, err)

	userId := "user-01"

	deckRows := sqlmock.NewRows(
		[]string{"deck_id", "name", "count", "wins", "game_count", "go_first_count", "go_first_wins", "go_second_wins"},
	).AddRow("deck-01", "リザードンex", 3, 2, 5, 3, 2, 0)

	mock.ExpectQuery(regexp.QuoteMeta(deckUsageStatQuery)).
		WithArgs(userId).
		WillReturnRows(deckRows)

	// deck-01 は集計対象外2件、deck-02 は集計対象外のみ1件（ignored_count DESC 順）
	ignoredRows := sqlmock.NewRows(deckUsageIgnoredColumns).
		AddRow("deck-01", "リザードンex", 2).
		AddRow("deck-02", "サーナイトex", 1)

	mock.ExpectQuery(regexp.QuoteMeta(deckUsageIgnoredQuery)).
		WithArgs(userId).
		WillReturnRows(ignoredRows)

	// スプライトはデッキごとに引かず1クエリでまとめて取得する。
	// deck-01 は集計対象・集計対象外の両方に現れるが、IDは重複を除いて渡される。
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1,$2) ORDER BY position ASC`,
	)).WithArgs("deck-01", "deck-02").WillReturnRows(sqlmock.NewRows([]string{"deck_id", "position", "pokemon_sprite_id"}))
	expectNoDeckVersionCounts(mock)

	stat, err := i.FindDeckUsageStat(context.Background(), userId, repository.StatPeriod{}, 0, false)

	require.NoError(t, err)
	require.Equal(t, 3, stat.TotalRecords)
	require.Len(t, stat.Decks, 2)

	deck1 := stat.Decks[0]
	require.Equal(t, "deck-01", deck1.DeckId)
	require.Equal(t, 3, deck1.Count)
	require.Equal(t, 2, deck1.IgnoredCount)

	deck2 := stat.Decks[1]
	require.Equal(t, "deck-02", deck2.DeckId)
	require.Equal(t, "サーナイトex", deck2.Name)
	require.Equal(t, 0, deck2.Count)
	require.Equal(t, 1, deck2.IgnoredCount)
}

// 不戦勝/不戦敗の除外を指定すると、対戦(matches)の集計にだけ条件が付くことを検証する。
// 集計対象外(ignore_stats_flg=true)の件数は記録単位の数え上げなので条件は付かない。
func test_DeckUsageStatInfrastructure_ExcludesDefaultMatches(t *testing.T) {
	i, mock, err := setup4DeckUsageStatInfrastructure()
	require.NoError(t, err)

	userId := "user-01"

	deckRows := sqlmock.NewRows(
		[]string{"deck_id", "name", "count", "wins", "draws", "game_count", "go_first_count", "go_first_wins", "go_second_wins"},
	).AddRow("deck-01", "リザードンex", 2, 1, 0, 3, 2, 1, 0)

	mock.ExpectQuery(`SELECT records\.deck_id AS deck_id.*FROM "matches".*matches\.default_victory_flg = false AND matches\.default_defeat_flg = false`).
		WithArgs(userId).
		WillReturnRows(deckRows)

	// 集計対象外の件数を数えるクエリには除外条件が付かない(付いていれば末尾の $ に一致せず失敗する)
	mock.ExpectQuery(regexp.QuoteMeta(deckUsageIgnoredQuery) + `$`).
		WithArgs(userId).
		WillReturnRows(sqlmock.NewRows(deckUsageIgnoredColumns))

	// 外した不戦の件数を数えるクエリ
	mock.ExpectQuery(regexp.QuoteMeta(deckUsageDefaultMatchQuery)).
		WithArgs(userId).
		WillReturnRows(
			sqlmock.NewRows(deckUsageDefaultMatchColumns).AddRow("deck-01", "リザードンex", 3),
		)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1) ORDER BY position ASC`,
	)).WithArgs("deck-01").WillReturnRows(sqlmock.NewRows([]string{"deck_id", "position", "pokemon_sprite_id"}))
	expectNoDeckVersionCounts(mock)

	stat, err := i.FindDeckUsageStat(context.Background(), userId, repository.StatPeriod{}, 0, true)

	require.NoError(t, err)
	require.Equal(t, 2, stat.TotalRecords)
	require.Len(t, stat.Decks, 1)

	deck := stat.Decks[0]
	require.Equal(t, 2, deck.Count)
	require.Equal(t, 1, deck.Wins)
	require.Equal(t, 1, deck.Losses)
	require.InDelta(t, 0.5, deck.WinRate, 0.0001)
	// 外した不戦の件数は、勝敗とは別に返す(画面が「勝率に含めていない」と示せるように)
	require.Equal(t, 3, deck.DefaultMatchCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

// 不戦しか記録が無いデッキが、対戦数0でも一覧に残ることを検証する。
// ここで消えると、画面には「まだ対戦記録がありません」と出てしまう。
func test_DeckUsageStatInfrastructure_DefaultOnlyDeckRemains(t *testing.T) {
	i, mock, err := setup4DeckUsageStatInfrastructure()
	require.NoError(t, err)

	userId := "user-01"

	// 集計対象の対戦は0件(不戦しか無いので除外条件で全て外れる)
	mock.ExpectQuery(`SELECT records\.deck_id AS deck_id.*FROM "matches".*matches\.default_victory_flg = false AND matches\.default_defeat_flg = false`).
		WithArgs(userId).
		WillReturnRows(sqlmock.NewRows([]string{"deck_id", "name", "count", "wins", "draws"}))

	mock.ExpectQuery(regexp.QuoteMeta(deckUsageIgnoredQuery) + `$`).
		WithArgs(userId).
		WillReturnRows(sqlmock.NewRows(deckUsageIgnoredColumns))

	mock.ExpectQuery(regexp.QuoteMeta(deckUsageDefaultMatchQuery)).
		WithArgs(userId).
		WillReturnRows(
			sqlmock.NewRows(deckUsageDefaultMatchColumns).AddRow("deck-01", "リザードンex", 1),
		)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1) ORDER BY position ASC`,
	)).WithArgs("deck-01").WillReturnRows(sqlmock.NewRows([]string{"deck_id", "position", "pokemon_sprite_id"}))
	// 集計対象の対戦が無いので、バージョンごとの対戦数は引かない(アイコンはデッキのもの)

	stat, err := i.FindDeckUsageStat(context.Background(), userId, repository.StatPeriod{}, 0, true)

	require.NoError(t, err)
	require.Equal(t, 0, stat.TotalRecords)
	require.Len(t, stat.Decks, 1)

	deck := stat.Decks[0]
	require.Equal(t, "deck-01", deck.DeckId)
	require.Equal(t, "リザードンex", deck.Name)
	require.Equal(t, 0, deck.Count)
	require.Equal(t, 1, deck.DefaultMatchCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

// 行のアイコンは、その期間の対戦でいちばん多く使ったスプライトの組み合わせにする。
// デッキのスプライトは最新バージョンに合わせて変わるので、デッキのものを出すと
// 過去の期間の集計にも「いまの相棒」が出てしまう。
func TestDeckUsageStatInfrastructure_RepresentativeSprites(t *testing.T) {
	userId := "user-01"
	deckId := "deck-01"

	type versionCount struct {
		id    string
		count int
	}

	// run は「デッキの対戦数 total・デッキのスプライト・バージョンごとの対戦数とスプライト」から
	// 行に出るスプライト(枠:ID)を返す。
	run := func(
		t *testing.T,
		total int,
		deckSprites [][2]any,
		versions []versionCount,
		versionSprites [][3]any,
	) []string {
		i, mock, err := setup4DeckUsageStatInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(regexp.QuoteMeta(deckUsageStatQuery)).
			WithArgs(userId).
			WillReturnRows(sqlmock.NewRows(
				[]string{"deck_id", "name", "count", "wins", "draws"},
			).AddRow(deckId, "ドラパルト", total, 0, 0))
		mock.ExpectQuery(regexp.QuoteMeta(deckUsageIgnoredQuery)).
			WithArgs(userId).
			WillReturnRows(sqlmock.NewRows(deckUsageIgnoredColumns))

		deckSpriteRows := sqlmock.NewRows([]string{"deck_id", "position", "pokemon_sprite_id"})
		for _, s := range deckSprites {
			deckSpriteRows.AddRow(deckId, s[0], s[1])
		}
		mock.ExpectQuery(`SELECT \* FROM "deck_pokemon_sprites" WHERE deck_id IN`).WillReturnRows(deckSpriteRows)

		countRows := sqlmock.NewRows(deckUsageVersionCountColumns)
		for _, v := range versions {
			countRows.AddRow(deckId, v.id, v.count)
		}
		mock.ExpectQuery(deckUsageVersionCountQueryPattern).WithArgs(userId).WillReturnRows(countRows)

		if len(versions) > 0 {
			versionSpriteRows := sqlmock.NewRows([]string{"deck_code_id", "position", "pokemon_sprite_id"})
			for _, s := range versionSprites {
				versionSpriteRows.AddRow(s[0], s[1], s[2])
			}
			mock.ExpectQuery(`SELECT \* FROM "deck_code_pokemon_sprites" WHERE deck_code_id IN`).
				WillReturnRows(versionSpriteRows)
		}

		stat, err := i.FindDeckUsageStat(context.Background(), userId, repository.StatPeriod{}, 0, false)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
		require.Len(t, stat.Decks, 1)

		// デッキ自体のスプライトは、行のアイコンがどれになっても、いまのデッキのもののまま返す
		// (デッキの選択肢のアイコンなど「どのデッキか」を示す用途に使う)
		require.Len(t, stat.Decks[0].DeckPokemonSprites, len(deckSprites))
		for index, sprite := range stat.Decks[0].DeckPokemonSprites {
			require.Equal(t, deckSprites[index][1], sprite.ID)
		}

		layout := []string{}
		for _, sprite := range stat.Decks[0].PokemonSprites {
			layout = append(layout, fmt.Sprintf("%d:%s", sprite.Position, sprite.ID))
		}
		return layout
	}

	// いまのデッキ(＝最新バージョン)は ドラパルト / リザードン
	charizard := [][2]any{{1, "0887"}, {2, "0006"}}

	t.Run("正常系_期間中にいちばん多く使ったバージョンのスプライトを出す", func(t *testing.T) {
		layout := run(t, 6, charizard,
			[]versionCount{{"v-old", 4}, {"v-new", 2}},
			[][3]any{
				{"v-old", 1, "0887"}, {"v-old", 2, "0477"},
				{"v-new", 1, "0887"}, {"v-new", 2, "0006"},
			},
		)

		// 相棒がヨノワールだった頃のバージョンで4戦、いまのバージョンで2戦
		require.Equal(t, []string{"1:0887", "2:0477"}, layout)
	})

	t.Run("正常系_同じスプライトのバージョンは合算して比べる", func(t *testing.T) {
		layout := run(t, 7, charizard,
			[]versionCount{{"v-old", 3}, {"v-new-1", 2}, {"v-new-2", 2}},
			[][3]any{
				{"v-old", 1, "0887"}, {"v-old", 2, "0477"},
				{"v-new-1", 1, "0887"}, {"v-new-1", 2, "0006"},
				{"v-new-2", 1, "0887"}, {"v-new-2", 2, "0006"},
			},
		)

		// リザードンの2バージョンを合わせると4戦で、ヨノワールの3戦を上回る
		require.Equal(t, []string{"1:0887", "2:0006"}, layout)
	})

	t.Run("正常系_未設定のバージョンとバージョンなしの対戦はデッキのスプライトとして数える", func(t *testing.T) {
		// 全7戦のうち、ヨノワールのバージョンが3戦、未設定のバージョンが2戦、バージョンなしが2戦
		layout := run(t, 7, charizard,
			[]versionCount{{"v-old", 3}, {"v-unset", 2}},
			[][3]any{{"v-old", 1, "0887"}, {"v-old", 2, "0477"}},
		)

		// デッキのスプライト扱いが 2+2=4戦でヨノワールの3戦を上回る
		require.Equal(t, []string{"1:0887", "2:0006"}, layout)
	})

	t.Run("正常系_同数ならいまのデッキのスプライトを優先する", func(t *testing.T) {
		layout := run(t, 4, charizard,
			[]versionCount{{"v-old", 2}, {"v-new", 2}},
			[][3]any{
				{"v-old", 1, "0887"}, {"v-old", 2, "0477"},
				{"v-new", 1, "0887"}, {"v-new", 2, "0006"},
			},
		)

		require.Equal(t, []string{"1:0887", "2:0006"}, layout)
	})

	t.Run("正常系_バージョンが紐付いた対戦が無ければデッキのスプライトを出す", func(t *testing.T) {
		layout := run(t, 3, charizard, nil, nil)

		require.Equal(t, []string{"1:0887", "2:0006"}, layout)
	})

	t.Run("正常系_デッキにスプライトが無くてもバージョンにあればそれを出す", func(t *testing.T) {
		layout := run(t, 3, nil,
			[]versionCount{{"v-old", 1}},
			[][3]any{{"v-old", 1, "0887"}},
		)

		require.Equal(t, []string{"1:0887"}, layout)
	})
}

// スプライトの組み合わせごとに行を分ける集計(FindDeckUsageStatBySprites)。
// 同じデッキ名のまま相棒を入れ替えた前後の成績を、別の行として見られるようにする。
func TestDeckUsageStatInfrastructure_SplitBySprites(t *testing.T) {
	userId := "user-01"

	// バージョン単位で集計する(あとで同じ組み合わせを足し合わせる)
	const splitQueryPattern = `SELECT records\.deck_id AS deck_id, COALESCE\(records\.deck_code_id, ''\) AS deck_code_id, COALESCE\(decks\.name, ''\) AS name, COUNT\(DISTINCT matches\.id\) AS count.*GROUP BY records\.deck_id, decks\.name, COALESCE\(records\.deck_code_id, ''\) ORDER BY count DESC`

	usageColumns := []string{"deck_id", "deck_code_id", "name", "count", "wins", "draws", "game_count", "go_first_count", "go_first_wins", "go_second_wins"}

	t.Run("正常系_同じデッキでもスプライトの組み合わせごとに行を分け同じ組み合わせは足し合わせる", func(t *testing.T) {
		i, mock, err := setup4DeckUsageStatInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(splitQueryPattern).WithArgs(userId).WillReturnRows(
			sqlmock.NewRows(usageColumns).
				// ドラパルト: ヨノワールのバージョンで4戦3勝
				AddRow("deck-01", "v-old", "ドラパルト", 4, 3, 0, 4, 2, 2, 1).
				// ドラパルト: リザードンのバージョンが2つ(メモ違いで作り直した)。合わせて5戦2勝1分
				AddRow("deck-01", "v-new-1", "ドラパルト", 3, 1, 1, 3, 1, 1, 0).
				AddRow("deck-01", "v-new-2", "ドラパルト", 2, 1, 0, 2, 1, 0, 1).
				// ドラパルト: バージョンなしの記録が1戦1勝 → デッキのスプライト(リザードン)へ合算
				AddRow("deck-01", "", "ドラパルト", 1, 1, 0, 0, 0, 0, 0).
				// 別のデッキ: スプライト未設定のバージョン → デッキのスプライト
				AddRow("deck-02", "v-x", "サーナイト", 2, 0, 0, 0, 0, 0, 0),
		)
		// 集計対象外の記録がドラパルトに2件(デッキ単位の数)
		mock.ExpectQuery(regexp.QuoteMeta(deckUsageIgnoredQuery)).WithArgs(userId).WillReturnRows(
			sqlmock.NewRows(deckUsageIgnoredColumns).AddRow("deck-01", "ドラパルト", 2),
		)
		mock.ExpectQuery(`SELECT \* FROM "deck_pokemon_sprites" WHERE deck_id IN`).WillReturnRows(
			sqlmock.NewRows([]string{"deck_id", "position", "pokemon_sprite_id"}).
				AddRow("deck-01", 1, "0887").AddRow("deck-01", 2, "0006").
				AddRow("deck-02", 1, "0282"),
		)
		mock.ExpectQuery(`SELECT \* FROM "deck_code_pokemon_sprites" WHERE deck_code_id IN`).WillReturnRows(
			sqlmock.NewRows([]string{"deck_code_id", "position", "pokemon_sprite_id"}).
				AddRow("v-old", 1, "0887").AddRow("v-old", 2, "0477").
				AddRow("v-new-1", 1, "0887").AddRow("v-new-1", 2, "0006").
				AddRow("v-new-2", 1, "0887").AddRow("v-new-2", 2, "0006"),
		)

		stat, err := i.FindDeckUsageStatBySprites(context.Background(), userId, repository.StatPeriod{}, 0, false)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
		require.Equal(t, 12, stat.TotalRecords)
		require.Len(t, stat.Decks, 3)

		// 対戦数の多い順: リザードン(3+2+1=6) → ヨノワール(4) → サーナイト(2)
		charizard, dusknoir, gardevoir := stat.Decks[0], stat.Decks[1], stat.Decks[2]

		require.Equal(t, "deck-01", charizard.DeckId)
		require.Equal(t, "0006", charizard.PokemonSprites[1].ID)
		require.Equal(t, 6, charizard.Count)
		require.Equal(t, 3, charizard.Wins)
		require.Equal(t, 2, charizard.Losses, "引き分け1は負けに数えない")
		require.InDelta(t, 6.0/12.0, charizard.UsageRate, 0.0001)
		require.Equal(t, 5, charizard.GameCount)
		require.Equal(t, 2, charizard.GoFirstCount)

		require.Equal(t, "deck-01", dusknoir.DeckId)
		require.Equal(t, "0477", dusknoir.PokemonSprites[1].ID)
		// 行を分けても、デッキ自体のスプライトはどの行もいまのデッキのもの(リザードン)
		require.Equal(t, "0006", dusknoir.DeckPokemonSprites[1].ID)
		require.Equal(t, 4, dusknoir.Count)
		require.Equal(t, 3, dusknoir.Wins)

		require.Equal(t, "deck-02", gardevoir.DeckId)
		require.Equal(t, "0282", gardevoir.PokemonSprites[0].ID)

		// デッキ単位の件数は先頭の行にだけ付ける(両方に付けると合計が4件に見える)
		require.Equal(t, 2, charizard.IgnoredCount)
		require.Equal(t, 0, dusknoir.IgnoredCount)
	})

	t.Run("正常系_分けない集計は従来どおりバージョンを取得しないクエリで1デッキ1行", func(t *testing.T) {
		i, mock, err := setup4DeckUsageStatInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(regexp.QuoteMeta(deckUsageStatQuery)).WithArgs(userId).WillReturnRows(
			sqlmock.NewRows([]string{"deck_id", "name", "count", "wins", "draws"}).AddRow("deck-01", "ドラパルト", 3, 1, 0),
		)
		mock.ExpectQuery(regexp.QuoteMeta(deckUsageIgnoredQuery)).WithArgs(userId).
			WillReturnRows(sqlmock.NewRows(deckUsageIgnoredColumns))
		mock.ExpectQuery(`SELECT \* FROM "deck_pokemon_sprites" WHERE deck_id IN`).
			WillReturnRows(sqlmock.NewRows([]string{"deck_id", "position", "pokemon_sprite_id"}))
		expectNoDeckVersionCounts(mock)

		stat, err := i.FindDeckUsageStat(context.Background(), userId, repository.StatPeriod{}, 0, false)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
		require.Len(t, stat.Decks, 1)
	})
}

const deckCodeUsageStatQuery = `SELECT COALESCE(records.deck_code_id, '') AS deck_code_id, COUNT(matches.id) AS count, SUM(CASE WHEN matches.victory_flg THEN 1 ELSE 0 END) AS wins, SUM(CASE WHEN matches.draw_flg THEN 1 ELSE 0 END) AS draws FROM "matches" JOIN records ON matches.record_id = records.id WHERE records.user_id = $1 AND records.deck_id = $2 AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL`

var deckCodeUsageColumns = []string{"deck_code_id", "count", "wins", "draws"}

func TestDeckUsageStatInfrastructure_FindDeckCodeUsageStat(t *testing.T) {
	userId := "user-01"
	deckId := "deck-01"

	t.Run("正常系_バージョンごとの勝敗を集計し未指定の対戦は件数だけ返す", func(t *testing.T) {
		i, mock, err := setup4DeckUsageStatInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(regexp.QuoteMeta(deckCodeUsageStatQuery+` GROUP BY COALESCE(records.deck_code_id, '')`)).
			WithArgs(userId, deckId).
			WillReturnRows(sqlmock.NewRows(deckCodeUsageColumns).
				AddRow("deckcode-01", 12, 5, 0).
				AddRow("deckcode-02", 5, 3, 1).
				AddRow("", 4, 2, 0))

		stat, err := i.FindDeckCodeUsageStat(context.Background(), userId, deckId, false)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
		require.Equal(t, 4, stat.UnassignedCount)
		require.Len(t, stat.DeckCodes, 2)

		require.Equal(t, "deckcode-01", stat.DeckCodes[0].DeckCodeId)
		require.Equal(t, 7, stat.DeckCodes[0].Losses)
		require.InDelta(t, 5.0/12.0, stat.DeckCodes[0].WinRate, 1e-9)

		// 引き分けは負けにも勝率の分母にも数えない
		require.Equal(t, 1, stat.DeckCodes[1].Losses)
		require.Equal(t, 1, stat.DeckCodes[1].Draws)
		require.InDelta(t, 0.75, stat.DeckCodes[1].WinRate, 1e-9)
	})

	t.Run("正常系_不戦勝・不戦敗を外す指定なら条件を足す", func(t *testing.T) {
		i, mock, err := setup4DeckUsageStatInfrastructure()
		require.NoError(t, err)

		// Where を重ねると GORM が各条件を括弧で包む
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(records.deck_code_id, '') AS deck_code_id, COUNT(matches.id) AS count, SUM(CASE WHEN matches.victory_flg THEN 1 ELSE 0 END) AS wins, SUM(CASE WHEN matches.draw_flg THEN 1 ELSE 0 END) AS draws FROM "matches" JOIN records ON matches.record_id = records.id WHERE (records.user_id = $1 AND records.deck_id = $2 AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL) AND (matches.default_victory_flg = false AND matches.default_defeat_flg = false) GROUP BY COALESCE(records.deck_code_id, '')`)).
			WithArgs(userId, deckId).
			WillReturnRows(sqlmock.NewRows(deckCodeUsageColumns))

		stat, err := i.FindDeckCodeUsageStat(context.Background(), userId, deckId, true)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
		require.Empty(t, stat.DeckCodes)
		require.Equal(t, 0, stat.UnassignedCount)
	})
}
