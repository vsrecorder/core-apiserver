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

func setup4OpponentDeckUsageStatInfrastructure() (repository.OpponentDeckUsageStatInterface, sqlmock.Sqlmock, error) {
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

	return NewOpponentDeckUsageStat(db), mock, nil
}

func TestOpponentDeckUsageStatInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"SameDeckInfoDifferentSpritesAreTreatedAsDifferentDecks": test_OpponentDeckUsageStatInfrastructure_SameDeckInfoDifferentSpritesAreTreatedAsDifferentDecks,
		"SameDeckInfoSameSpritesAreAggregated":                   test_OpponentDeckUsageStatInfrastructure_SameDeckInfoSameSpritesAreAggregated,
		"NoMatches":                                              test_OpponentDeckUsageStatInfrastructure_NoMatches,
		"FilterByDeckIdUsesRecordsDeckId":                        test_OpponentDeckUsageStatInfrastructure_FilterByDeckIdUsesRecordsDeckId,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_OpponentDeckUsageStatInfrastructure_SameDeckInfoDifferentSpritesAreTreatedAsDifferentDecks(t *testing.T) {
	i, mock, err := setup4OpponentDeckUsageStatInfrastructure()
	require.NoError(t, err)

	userId := "user-01"

	matchRows := sqlmock.NewRows([]string{"match_id", "deck_info", "victory_flg"}).
		AddRow("match-01", "リザードンex", true).
		AddRow("match-02", "リザードンex", false)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT matches.id AS match_id, matches.opponents_deck_info AS deck_info, matches.victory_flg AS victory_flg FROM "matches" JOIN records ON matches.record_id = records.id WHERE records.user_id = $1 AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND matches.opponents_deck_info != '' ORDER BY records.event_date ASC`,
	)).WithArgs(userId).WillReturnRows(matchRows)

	spriteRows := sqlmock.NewRows([]string{"match_id", "position", "pokemon_sprite_id"}).
		AddRow("match-01", 1, "0006").
		AddRow("match-02", 1, "0025")

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "match_pokemon_sprites" WHERE match_id IN ($1,$2) ORDER BY position ASC`,
	)).WithArgs("match-01", "match-02").WillReturnRows(spriteRows)

	stat, err := i.FindOpponentDeckUsageStat(context.Background(), userId, time.Time{}, time.Time{}, "")

	require.NoError(t, err)
	require.Equal(t, 2, stat.TotalMatches)
	// デッキ名は同じだがスプライトが異なるため、2件の別デッキとして集計される
	require.Len(t, stat.Decks, 2)
	for _, d := range stat.Decks {
		require.Equal(t, "リザードンex", d.DeckInfo)
		require.Equal(t, 1, d.Count)
	}
}

func test_OpponentDeckUsageStatInfrastructure_SameDeckInfoSameSpritesAreAggregated(t *testing.T) {
	i, mock, err := setup4OpponentDeckUsageStatInfrastructure()
	require.NoError(t, err)

	userId := "user-02"

	matchRows := sqlmock.NewRows([]string{"match_id", "deck_info", "victory_flg"}).
		AddRow("match-01", "リザードンex", true).
		AddRow("match-02", "リザードンex", false)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT matches.id AS match_id, matches.opponents_deck_info AS deck_info, matches.victory_flg AS victory_flg FROM "matches" JOIN records ON matches.record_id = records.id WHERE records.user_id = $1 AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND matches.opponents_deck_info != '' ORDER BY records.event_date ASC`,
	)).WithArgs(userId).WillReturnRows(matchRows)

	spriteRows := sqlmock.NewRows([]string{"match_id", "position", "pokemon_sprite_id"}).
		AddRow("match-01", 1, "0006").
		AddRow("match-02", 1, "0006")

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "match_pokemon_sprites" WHERE match_id IN ($1,$2) ORDER BY position ASC`,
	)).WithArgs("match-01", "match-02").WillReturnRows(spriteRows)

	stat, err := i.FindOpponentDeckUsageStat(context.Background(), userId, time.Time{}, time.Time{}, "")

	require.NoError(t, err)
	require.Equal(t, 2, stat.TotalMatches)
	// デッキ名・スプライト構成が同じなので1件に集計される
	require.Len(t, stat.Decks, 1)
	require.Equal(t, "リザードンex", stat.Decks[0].DeckInfo)
	require.Equal(t, 2, stat.Decks[0].Count)
	require.Equal(t, 1, stat.Decks[0].Wins)
	require.Equal(t, 1, stat.Decks[0].Losses)
	require.InDelta(t, 0.5, stat.Decks[0].WinRate, 0.0001)
}

func test_OpponentDeckUsageStatInfrastructure_NoMatches(t *testing.T) {
	i, mock, err := setup4OpponentDeckUsageStatInfrastructure()
	require.NoError(t, err)

	userId := "user-03"

	matchRows := sqlmock.NewRows([]string{"match_id", "deck_info", "victory_flg"})

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT matches.id AS match_id, matches.opponents_deck_info AS deck_info, matches.victory_flg AS victory_flg FROM "matches" JOIN records ON matches.record_id = records.id WHERE records.user_id = $1 AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND matches.opponents_deck_info != '' ORDER BY records.event_date ASC`,
	)).WithArgs(userId).WillReturnRows(matchRows)

	stat, err := i.FindOpponentDeckUsageStat(context.Background(), userId, time.Time{}, time.Time{}, "")

	require.NoError(t, err)
	require.Equal(t, 0, stat.TotalMatches)
	require.Empty(t, stat.Decks)
}

// 「自分のデッキ」セレクタは records.deck_id を基準に選択肢を作っているため（deck_usage_stat.go参照）、
// deck_id 絞り込みも records.deck_id を使う必要がある。
// matches.deck_id はマッチ作成時点の値のまま更新されないため、記録編集後のデッキ変更やアーカイブ済みデッキ選択時に
// records.deck_id とズレて対戦相手デッキが見つからなくなる不具合の再発防止テスト。
func test_OpponentDeckUsageStatInfrastructure_FilterByDeckIdUsesRecordsDeckId(t *testing.T) {
	i, mock, err := setup4OpponentDeckUsageStatInfrastructure()
	require.NoError(t, err)

	userId := "user-04"
	deckId := "deck-01"

	matchRows := sqlmock.NewRows([]string{"match_id", "deck_info", "victory_flg"}).
		AddRow("match-01", "リザードンex", true)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT matches.id AS match_id, matches.opponents_deck_info AS deck_info, matches.victory_flg AS victory_flg FROM "matches" JOIN records ON matches.record_id = records.id WHERE (records.user_id = $1 AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND matches.opponents_deck_info != '') AND records.deck_id = $2 ORDER BY records.event_date ASC`,
	)).WithArgs(userId, deckId).WillReturnRows(matchRows)

	spriteRows := sqlmock.NewRows([]string{"match_id", "position", "pokemon_sprite_id"}).
		AddRow("match-01", 1, "0006")

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "match_pokemon_sprites" WHERE match_id IN ($1) ORDER BY position ASC`,
	)).WithArgs("match-01").WillReturnRows(spriteRows)

	stat, err := i.FindOpponentDeckUsageStat(context.Background(), userId, time.Time{}, time.Time{}, deckId)

	require.NoError(t, err)
	require.Equal(t, 1, stat.TotalMatches)
	require.Len(t, stat.Decks, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}
