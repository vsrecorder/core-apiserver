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

func setup4UserStatRecentInfrastructure() (repository.UserStatRecentInterface, sqlmock.Sqlmock, error) {
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

	return NewUserStatRecent(db), mock, nil
}

func TestUserStatRecentInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"FindRecentMatches":                test_UserStatRecentInfrastructure_FindRecentMatches,
		"FindRecentMatchesWithDeckId":      test_UserStatRecentInfrastructure_FindRecentMatchesWithDeckId,
		"FindRecentMatchesOrdersAscending": test_UserStatRecentInfrastructure_FindRecentMatchesOrdersAscending,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_UserStatRecentInfrastructure_FindRecentMatches(t *testing.T) {
	i, mock, err := setup4UserStatRecentInfrastructure()
	require.NoError(t, err)

	userId := "user-01"
	count := 10

	rows := sqlmock.NewRows([]string{
		"match_id",
		"event_date",
		"deck_id",
		"opponents_deck_info",
		"victory_flg",
	})

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT matches.id AS match_id, records.event_date AS event_date, matches.deck_id AS deck_id, matches.opponents_deck_info AS opponents_deck_info, matches.victory_flg AS victory_flg FROM "matches" JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL WHERE matches.user_id = $1 AND matches.deleted_at IS NULL ORDER BY records.event_date DESC, matches.created_at DESC LIMIT $2`,
	)).WithArgs(
		userId,
		count,
	).WillReturnRows(rows)

	matches, err := i.FindRecentMatches(context.Background(), userId, count, "")

	require.NoError(t, err)
	require.Empty(t, matches)
}

func test_UserStatRecentInfrastructure_FindRecentMatchesWithDeckId(t *testing.T) {
	i, mock, err := setup4UserStatRecentInfrastructure()
	require.NoError(t, err)

	userId := "user-01"
	count := 10
	deckId := "deck-01"

	rows := sqlmock.NewRows([]string{
		"match_id",
		"event_date",
		"deck_id",
		"opponents_deck_info",
		"victory_flg",
	})

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT matches.id AS match_id, records.event_date AS event_date, matches.deck_id AS deck_id, matches.opponents_deck_info AS opponents_deck_info, matches.victory_flg AS victory_flg FROM "matches" JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL WHERE (matches.user_id = $1 AND matches.deleted_at IS NULL) AND matches.deck_id = $2 ORDER BY records.event_date DESC, matches.created_at DESC LIMIT $3`,
	)).WithArgs(
		userId,
		deckId,
		count,
	).WillReturnRows(rows)

	matches, err := i.FindRecentMatches(context.Background(), userId, count, deckId)

	require.NoError(t, err)
	require.Empty(t, matches)
}

func test_UserStatRecentInfrastructure_FindRecentMatchesOrdersAscending(t *testing.T) {
	i, mock, err := setup4UserStatRecentInfrastructure()
	require.NoError(t, err)

	userId := "user-01"
	count := 2

	newest := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	oldest := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// DBはevent_date DESCで返す（新しい試合が先頭）
	rows := sqlmock.NewRows([]string{
		"match_id",
		"event_date",
		"deck_id",
		"opponents_deck_info",
		"victory_flg",
	}).AddRow(
		"match-02", newest, "deck-01", "対戦相手デッキA", true,
	).AddRow(
		"match-01", oldest, "deck-01", "対戦相手デッキB", false,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT matches.id AS match_id, records.event_date AS event_date, matches.deck_id AS deck_id, matches.opponents_deck_info AS opponents_deck_info, matches.victory_flg AS victory_flg FROM "matches" JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL WHERE matches.user_id = $1 AND matches.deleted_at IS NULL ORDER BY records.event_date DESC, matches.created_at DESC LIMIT $2`,
	)).WithArgs(
		userId,
		count,
	).WillReturnRows(rows)

	spriteRows := sqlmock.NewRows([]string{"match_id", "position", "pokemon_sprite_id"}).
		AddRow("match-01", 1, "0006").
		AddRow("match-02", 1, "0025")

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "match_pokemon_sprites" WHERE match_id IN ($1,$2) ORDER BY position ASC`,
	)).WithArgs("match-02", "match-01").WillReturnRows(spriteRows)

	matches, err := i.FindRecentMatches(context.Background(), userId, count, "")

	require.NoError(t, err)
	require.Len(t, matches, 2)
	// 返り値は対戦日の古い順に並び替えられている
	require.True(t, matches[0].EventDate.Equal(oldest))
	require.True(t, matches[1].EventDate.Equal(newest))
	// スプライトも対応する試合に紐づいている
	require.Len(t, matches[0].PokemonSprites, 1)
	require.Equal(t, "0006", matches[0].PokemonSprites[0].ID)
	require.Len(t, matches[1].PokemonSprites, 1)
	require.Equal(t, "0025", matches[1].PokemonSprites[0].ID)
}
