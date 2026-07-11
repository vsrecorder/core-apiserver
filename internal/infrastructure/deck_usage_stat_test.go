package infrastructure

import (
	"context"
	"regexp"
	"testing"
	"time"

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

const deckUsageStatQuery = `SELECT records.deck_id AS deck_id, COALESCE(decks.name, '') AS name, COUNT(DISTINCT matches.id) AS count, COUNT(DISTINCT CASE WHEN matches.victory_flg THEN matches.id END) AS wins, COUNT(games.id) AS game_count, SUM(CASE WHEN games.go_first THEN 1 ELSE 0 END) AS go_first_count, SUM(CASE WHEN games.go_first AND games.winning_flg THEN 1 ELSE 0 END) AS go_first_wins, SUM(CASE WHEN games.go_first = false AND games.winning_flg THEN 1 ELSE 0 END) AS go_second_wins FROM "matches" JOIN records ON matches.record_id = records.id LEFT JOIN decks ON records.deck_id = decks.id LEFT JOIN games ON games.match_id = matches.id AND games.deleted_at IS NULL WHERE records.user_id = $1 AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND records.deck_id != '' GROUP BY records.deck_id, decks.name ORDER BY count DESC`

const deckUsageIgnoredQuery = `SELECT records.deck_id AS deck_id, COALESCE(decks.name, '') AS name, COUNT(DISTINCT records.id) AS ignored_count FROM "records" LEFT JOIN decks ON records.deck_id = decks.id WHERE records.user_id = $1 AND records.deleted_at IS NULL AND records.ignore_stats_flg = true AND records.deck_id != '' GROUP BY records.deck_id, decks.name ORDER BY ignored_count DESC`

var deckUsageIgnoredColumns = []string{"deck_id", "name", "ignored_count"}

func TestDeckUsageStatInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"AggregatesWinsAndGoFirstCountsPerDeck": test_DeckUsageStatInfrastructure_AggregatesWinsAndGoFirstCountsPerDeck,
		"NoMatches":                             test_DeckUsageStatInfrastructure_NoMatches,
		"IncludesIgnoredCounts":                 test_DeckUsageStatInfrastructure_IncludesIgnoredCounts,
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
		`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id = $1 ORDER BY position ASC`,
	)).WithArgs("deck-01").WillReturnRows(sqlmock.NewRows([]string{"deck_id", "position", "pokemon_sprite_id"}))

	stat, err := i.FindDeckUsageStat(context.Background(), userId, time.Time{}, time.Time{})

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

	stat, err := i.FindDeckUsageStat(context.Background(), userId, time.Time{}, time.Time{})

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

	// 集計対象デッキ(deck-01)のスプライト取得
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id = $1 ORDER BY position ASC`,
	)).WithArgs("deck-01").WillReturnRows(sqlmock.NewRows([]string{"deck_id", "position", "pokemon_sprite_id"}))

	// 集計対象外のみのデッキ(deck-02)のスプライト取得
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id = $1 ORDER BY position ASC`,
	)).WithArgs("deck-02").WillReturnRows(sqlmock.NewRows([]string{"deck_id", "position", "pokemon_sprite_id"}))

	stat, err := i.FindDeckUsageStat(context.Background(), userId, time.Time{}, time.Time{})

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
