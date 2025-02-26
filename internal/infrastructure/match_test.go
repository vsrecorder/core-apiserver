package infrastructure

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMock4MatchInfrastructure() (*gorm.DB, sqlmock.Sqlmock, error) {
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

	return db, mock, err
}

func setup4MatchInfrastructure() (repository.MatchInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock4MatchInfrastructure()

	if err != nil {
		return nil, nil, err
	}

	r := NewMatch(db)

	return r, mock, err
}

func TestMatchInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"FindById":       test_MatchInfrastructure_FindById,
		"FindByRecordId": test_MatchInfrastructure_FindByRecordId,
		"Create":         test_MatchInfrastructure_Create,
		"Update":         test_MatchInfrastructure_Update,
		"Delete":         test_MatchInfrastructure_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_MatchInfrastructure_FindById(t *testing.T) {
	r, mock, err := setup4MatchInfrastructure()
	require.NoError(t, err)

	{
		matchId := "01JMPKHAXQHQYJZ6VVASF5CATW"

		createdAt := time.Now().Truncate(0)
		updatedAt := time.Now().Truncate(0)

		values := [][]driver.Value{
			{
				matchId,
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				"01JMPK4VF04QX714CG4PHYJ88K",
				"01JMKRNBW5TVN902YAE8GYZ367",
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				"KBp7roRDZobZg1t0OPzFR1kvLeO2",
				true,
				false,
				false,
				false,
				false,
				false,
				"Test1",
				"memo",
				"01JMPKHBXCJ32JZYNMDMY9SZ3B",
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				matchId,
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				false,
				false,
				0,
				0,
				"memo1",
			},
			{
				matchId,
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				"01JMPK4VF04QX714CG4PHYJ88K",
				"01JMKRNBW5TVN902YAE8GYZ367",
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				"KBp7roRDZobZg1t0OPzFR1kvLeO2",
				true,
				false,
				false,
				false,
				false,
				false,
				"Test1",
				"memo",
				"01JMPMPY964J0XBR7F5FTSGCDC",
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				matchId,
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				true,
				true,
				0,
				0,
				"memo2",
			},
			{
				matchId,
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				"01JMPK4VF04QX714CG4PHYJ88K",
				"01JMKRNBW5TVN902YAE8GYZ367",
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				"KBp7roRDZobZg1t0OPzFR1kvLeO2",
				true,
				false,
				false,
				false,
				false,
				false,
				"Test1",
				"memo",
				"01JMPMSN6RVW69EME7F1SGW5MD",
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				matchId,
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				false,
				true,
				0,
				0,
				"memo3",
			},
		}
		rows := sqlmock.NewRows([]string{
			"match_id",
			"match_created_at",
			"match_updated_at",
			"match_deleted_at",
			"match_record_id",
			"match_deck_id",
			"match_user_id",
			"match_opponents_user_id",
			"match_bo3_flg",
			"match_qualifying_round_flg",
			"match_final_tournament_flg",
			"match_default_victory_flg",
			"match_default_defeat_flg",
			"match_victory_flg",
			"match_opponents_deck_info",
			"match_memo",
			"game_id",
			"game_created_at",
			"game_updated_at",
			"game_deleted_at",
			"game_match_id",
			"game_user_id",
			"game_go_first",
			"game_winning_flg",
			"game_your_prize_cards",
			"game_opponents_prize_cards",
			"game_memo",
		}).AddRows(values...)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT matches.id AS match_id,matches.created_at AS match_created_at,matches.updated_at AS match_updated_at,matches.deleted_at AS match_deleted_at,matches.record_id AS match_record_id,matches.deck_id AS match_deck_id,matches.user_id AS match_user_id,matches.opponents_user_id AS match_opponents_user_id,matches.bo3_flg AS match_bo3_flg,matches.qualifying_round_flg AS match_qualifying_round_flg,matches.final_tournament_flg AS match_final_tournament_flg,matches.default_victory_flg AS match_default_victory_flg,matches.default_defeat_flg AS match_default_defeat_flg,matches.victory_flg AS match_victory_flg,matches.opponents_deck_info AS match_opponents_deck_info,matches.memo AS match_memo,games.id AS game_id,games.created_at AS game_created_at,games.updated_at AS game_updated_at,games.deleted_at AS game_deleted_at,games.match_id AS game_match_id, games.user_id AS game_user_id, games.go_first AS game_go_first, games.winning_flg AS game_winning_flg,games.your_prize_cards AS game_your_prize_cards,games.opponents_prize_cards AS game_opponents_prize_cards,games.memo AS game_memo FROM "matches" INNER JOIN games on matches.id = games.match_id WHERE matches.id = $1 AND matches.deleted_at IS NULL ORDER BY games.created_at ASC`,
		)).WithArgs(
			matchId,
		).WillReturnRows(rows)

		matches, err := r.FindById(context.Background(), matchId)
		require.NoError(t, err)

		require.Equal(t, matchId, matches.ID)
		require.Equal(t, createdAt, matches.CreatedAt)
		require.Equal(t, "01JMPK4VF04QX714CG4PHYJ88K", matches.RecordId)
		require.Equal(t, "01JMKRNBW5TVN902YAE8GYZ367", matches.DeckId)
		require.Equal(t, "zor5SLfEfwfZ90yRVXzlxBEFARy2", matches.UserId)
		require.Equal(t, "KBp7roRDZobZg1t0OPzFR1kvLeO2", matches.OpponentsUserId)
		require.Equal(t, "memo", matches.Memo)
		require.Equal(t, len(values), len(matches.Games))

		require.Equal(t, "01JMPKHBXCJ32JZYNMDMY9SZ3B", matches.Games[0].ID)
		require.Equal(t, matchId, matches.Games[0].MatchId)
		require.Equal(t, "zor5SLfEfwfZ90yRVXzlxBEFARy2", matches.Games[0].UserId)
		require.Equal(t, false, matches.Games[0].GoFirst)
		require.Equal(t, false, matches.Games[0].WinningFlg)
		require.Equal(t, "memo1", matches.Games[0].Memo)

		require.Equal(t, "01JMPMPY964J0XBR7F5FTSGCDC", matches.Games[1].ID)
		require.Equal(t, matchId, matches.Games[1].MatchId)
		require.Equal(t, true, matches.Games[1].GoFirst)
		require.Equal(t, "zor5SLfEfwfZ90yRVXzlxBEFARy2", matches.Games[1].UserId)
		require.Equal(t, true, matches.Games[1].WinningFlg)
		require.Equal(t, "memo2", matches.Games[1].Memo)

		require.Equal(t, "01JMPMSN6RVW69EME7F1SGW5MD", matches.Games[2].ID)
		require.Equal(t, matchId, matches.Games[2].MatchId)
		require.Equal(t, "zor5SLfEfwfZ90yRVXzlxBEFARy2", matches.Games[2].UserId)
		require.Equal(t, false, matches.Games[2].GoFirst)
		require.Equal(t, true, matches.Games[2].WinningFlg)
		require.Equal(t, "memo3", matches.Games[2].Memo)
	}

	{
		matchId := "01JMPKHM2CAECZ9F6V67ZY57N2"

		createdAt := time.Now().Truncate(0)
		updatedAt := time.Now().Truncate(0)

		values := [][]driver.Value{
			{
				matchId,
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				"01JMPK4VF04QX714CG4PHYJ88K",
				"",
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				"",
				true,
				false,
				false,
				false,
				false,
				true,
				"Test2",
				"",
				"01JMPKHM7QD0X26JMWV23JY4M9",
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				matchId,
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				true,
				true,
				0,
				0,
				"",
			},
			{
				matchId,
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				"01JMPK4VF04QX714CG4PHYJ88K",
				"",
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				"",
				true,
				false,
				false,
				false,
				false,
				true,
				"Test2",
				"",
				"01JMPM7WRQBDTYKH8BB921XX5K",
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				matchId,
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				false,
				true,
				0,
				0,
				"",
			},
		}
		rows := sqlmock.NewRows([]string{
			"match_id",
			"match_created_at",
			"match_updated_at",
			"match_deleted_at",
			"match_record_id",
			"match_deck_id",
			"match_user_id",
			"match_opponents_user_id",
			"match_bo3_flg",
			"match_qualifying_round_flg",
			"match_final_tournament_flg",
			"match_default_victory_flg",
			"match_default_defeat_flg",
			"match_victory_flg",
			"match_opponents_deck_info",
			"match_memo",
			"game_id",
			"game_created_at",
			"game_updated_at",
			"game_deleted_at",
			"game_match_id",
			"game_user_id",
			"game_go_first",
			"game_winning_flg",
			"game_your_prize_cards",
			"game_opponents_prize_cards",
			"game_memo",
		}).AddRows(values...)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT matches.id AS match_id,matches.created_at AS match_created_at,matches.updated_at AS match_updated_at,matches.deleted_at AS match_deleted_at,matches.record_id AS match_record_id,matches.deck_id AS match_deck_id,matches.user_id AS match_user_id,matches.opponents_user_id AS match_opponents_user_id,matches.bo3_flg AS match_bo3_flg,matches.qualifying_round_flg AS match_qualifying_round_flg,matches.final_tournament_flg AS match_final_tournament_flg,matches.default_victory_flg AS match_default_victory_flg,matches.default_defeat_flg AS match_default_defeat_flg,matches.victory_flg AS match_victory_flg,matches.opponents_deck_info AS match_opponents_deck_info,matches.memo AS match_memo,games.id AS game_id,games.created_at AS game_created_at,games.updated_at AS game_updated_at,games.deleted_at AS game_deleted_at,games.match_id AS game_match_id, games.user_id AS game_user_id, games.go_first AS game_go_first, games.winning_flg AS game_winning_flg,games.your_prize_cards AS game_your_prize_cards,games.opponents_prize_cards AS game_opponents_prize_cards,games.memo AS game_memo FROM "matches" INNER JOIN games on matches.id = games.match_id WHERE matches.id = $1 AND matches.deleted_at IS NULL ORDER BY games.created_at ASC`,
		)).WithArgs(
			matchId,
		).WillReturnRows(rows)

		matches, err := r.FindById(context.Background(), matchId)
		require.NoError(t, err)

		require.Equal(t, matchId, matches.ID)
		require.Equal(t, createdAt, matches.CreatedAt)
		require.Equal(t, len(values), len(matches.Games))

		require.Equal(t, matchId, matches.Games[0].MatchId)
		require.Equal(t, "01JMPKHM7QD0X26JMWV23JY4M9", matches.Games[0].ID)

		require.Equal(t, matchId, matches.Games[1].MatchId)
		require.Equal(t, "01JMPM7WRQBDTYKH8BB921XX5K", matches.Games[1].ID)
	}

	{
		matchId := "01JMPKHM2CAECZ9F6V67ZY57N2"

		createdAt := time.Now().Truncate(0)
		updatedAt := time.Now().Truncate(0)

		values := [][]driver.Value{
			{
				matchId,
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				"01JMPK4VF04QX714CG4PHYJ88K",
				"",
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				"",
				false,
				true,
				true,
				true,
				true,
				true,
				"Test3",
				"",
				"01JMPKHM7QD0X26JMWV23JY4M9",
				createdAt,
				updatedAt,
				gorm.DeletedAt{},
				matchId,
				"zor5SLfEfwfZ90yRVXzlxBEFARy2",
				true,
				true,
				6,
				5,
				"",
			},
		}
		rows := sqlmock.NewRows([]string{
			"match_id",
			"match_created_at",
			"match_updated_at",
			"match_deleted_at",
			"match_record_id",
			"match_deck_id",
			"match_user_id",
			"match_opponents_user_id",
			"match_bo3_flg",
			"match_qualifying_round_flg",
			"match_final_tournament_flg",
			"match_default_victory_flg",
			"match_default_defeat_flg",
			"match_victory_flg",
			"match_opponents_deck_info",
			"match_memo",
			"game_id",
			"game_created_at",
			"game_updated_at",
			"game_deleted_at",
			"game_match_id",
			"game_user_id",
			"game_go_first",
			"game_winning_flg",
			"game_your_prize_cards",
			"game_opponents_prize_cards",
			"game_memo",
		}).AddRows(values...)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT matches.id AS match_id,matches.created_at AS match_created_at,matches.updated_at AS match_updated_at,matches.deleted_at AS match_deleted_at,matches.record_id AS match_record_id,matches.deck_id AS match_deck_id,matches.user_id AS match_user_id,matches.opponents_user_id AS match_opponents_user_id,matches.bo3_flg AS match_bo3_flg,matches.qualifying_round_flg AS match_qualifying_round_flg,matches.final_tournament_flg AS match_final_tournament_flg,matches.default_victory_flg AS match_default_victory_flg,matches.default_defeat_flg AS match_default_defeat_flg,matches.victory_flg AS match_victory_flg,matches.opponents_deck_info AS match_opponents_deck_info,matches.memo AS match_memo,games.id AS game_id,games.created_at AS game_created_at,games.updated_at AS game_updated_at,games.deleted_at AS game_deleted_at,games.match_id AS game_match_id, games.user_id AS game_user_id, games.go_first AS game_go_first, games.winning_flg AS game_winning_flg,games.your_prize_cards AS game_your_prize_cards,games.opponents_prize_cards AS game_opponents_prize_cards,games.memo AS game_memo FROM "matches" INNER JOIN games on matches.id = games.match_id WHERE matches.id = $1 AND matches.deleted_at IS NULL ORDER BY games.created_at ASC`,
		)).WithArgs(
			matchId,
		).WillReturnRows(rows)

		matches, err := r.FindById(context.Background(), matchId)
		require.NoError(t, err)

		require.Equal(t, matchId, matches.ID)
		require.Equal(t, createdAt, matches.CreatedAt)
		require.Equal(t, false, matches.BO3Flg)
		require.Equal(t, true, matches.QualifyingRoundFlg)
		require.Equal(t, true, matches.FinalTournamentFlg)
		require.Equal(t, true, matches.DefaultVictoryFlg)
		require.Equal(t, true, matches.DefaultDefeatFlg)
		require.Equal(t, true, matches.VictoryFlg)
		require.Equal(t, len(values), len(matches.Games))

		require.Equal(t, "01JMPKHM7QD0X26JMWV23JY4M9", matches.Games[0].ID)
		require.Equal(t, matchId, matches.Games[0].MatchId)
		require.Equal(t, true, matches.Games[0].GoFirst)
		require.Equal(t, true, matches.Games[0].WinningFlg)
		require.Equal(t, uint(6), matches.Games[0].YourPrizeCards)
		require.Equal(t, uint(5), matches.Games[0].OpponentsPrizeCards)
	}
}

func test_MatchInfrastructure_FindByRecordId(t *testing.T) {
	r, mock, err := setup4MatchInfrastructure()
	require.NoError(t, err)

	recordId := "01JMPK4VF04QX714CG4PHYJ88K"

	createdAt := time.Now().Truncate(0)
	updatedAt := time.Now().Truncate(0)

	values := [][]driver.Value{
		{
			"01JMPKHAXQHQYJZ6VVASF5CATW",
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			recordId,
			"01JMKRNBW5TVN902YAE8GYZ367",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			"KBp7roRDZobZg1t0OPzFR1kvLeO2",
			true,
			false,
			false,
			false,
			false,
			false,
			"Test1",
			"memo",
			"01JMPKHBXCJ32JZYNMDMY9SZ3B",
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			"01JMPKHAXQHQYJZ6VVASF5CATW",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			false,
			false,
			0,
			0,
			"memo1",
		},
		{
			"01JMPKHAXQHQYJZ6VVASF5CATW",
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			recordId,
			"01JMKRNBW5TVN902YAE8GYZ367",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			"KBp7roRDZobZg1t0OPzFR1kvLeO2",
			true,
			false,
			false,
			false,
			false,
			false,
			"Test1",
			"memo",
			"01JMPMPY964J0XBR7F5FTSGCDC",
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			"01JMPKHAXQHQYJZ6VVASF5CATW",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			true,
			true,
			0,
			0,
			"memo2",
		},
		{
			"01JMPKHAXQHQYJZ6VVASF5CATW",
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			recordId,
			"01JMKRNBW5TVN902YAE8GYZ367",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			"KBp7roRDZobZg1t0OPzFR1kvLeO2",
			true,
			false,
			false,
			false,
			false,
			false,
			"Test1",
			"memo",
			"01JMPMSN6RVW69EME7F1SGW5MD",
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			"01JMPKHAXQHQYJZ6VVASF5CATW",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			false,
			true,
			0,
			0,
			"memo3",
		},
		{
			"01JMPKHM2CAECZ9F6V67ZY57N2",
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			recordId,
			"01JMKRNBW5TVN902YAE8GYZ367",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			"",
			true,
			false,
			false,
			false,
			false,
			true,
			"Test2",
			"",
			"01JMPKHM7QD0X26JMWV23JY4M9",
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			"01JMPKHM2CAECZ9F6V67ZY57N2",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			true,
			true,
			0,
			0,
			"",
		},
		{
			"01JMPKHM2CAECZ9F6V67ZY57N2",
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			recordId,
			"01JMKRNBW5TVN902YAE8GYZ367",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			"",
			true,
			false,
			false,
			false,
			false,
			true,
			"Test2",
			"",
			"01JMPM7WRQBDTYKH8BB921XX5K",
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			"01JMPKHM2CAECZ9F6V67ZY57N2",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			false,
			true,
			0,
			0,
			"",
		},
	}
	rows := sqlmock.NewRows([]string{
		"match_id",
		"match_created_at",
		"match_updated_at",
		"match_deleted_at",
		"match_record_id",
		"match_deck_id",
		"match_user_id",
		"match_opponents_user_id",
		"match_bo3_flg",
		"match_qualifying_round_flg",
		"match_final_tournament_flg",
		"match_default_victory_flg",
		"match_default_defeat_flg",
		"match_victory_flg",
		"match_opponents_deck_info",
		"match_memo",
		"game_id",
		"game_created_at",
		"game_updated_at",
		"game_deleted_at",
		"game_match_id",
		"game_user_id",
		"game_go_first",
		"game_winning_flg",
		"game_your_prize_cards",
		"game_opponents_prize_cards",
		"game_memo",
	}).AddRows(values...)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT matches.id AS match_id,matches.created_at AS match_created_at,matches.updated_at AS match_updated_at,matches.deleted_at AS match_deleted_at,matches.record_id AS match_record_id,matches.deck_id AS match_deck_id,matches.user_id AS match_user_id,matches.opponents_user_id AS match_opponents_user_id,matches.bo3_flg AS match_bo3_flg,matches.qualifying_round_flg AS match_qualifying_round_flg,matches.final_tournament_flg AS match_final_tournament_flg,matches.default_victory_flg AS match_default_victory_flg,matches.default_defeat_flg AS match_default_defeat_flg,matches.victory_flg AS match_victory_flg,matches.opponents_deck_info AS match_opponents_deck_info,matches.memo AS match_memo,games.id AS game_id,games.created_at AS game_created_at,games.updated_at AS game_updated_at,games.deleted_at AS game_deleted_at,games.match_id AS game_match_id, games.user_id AS game_user_id, games.go_first AS game_go_first, games.winning_flg AS game_winning_flg,games.your_prize_cards AS game_your_prize_cards,games.opponents_prize_cards AS game_opponents_prize_cards,games.memo AS game_memo FROM "records" INNER JOIN matches on records.id = matches.record_id INNER JOIN games on matches.id = games.match_id WHERE records.id = $1 AND records.deleted_at IS NULL AND matches.deleted_at IS NULL ORDER BY matches.created_at, games.created_at ASC`,
	)).WithArgs(
		recordId,
	).WillReturnRows(rows)

	matches, err := r.FindByRecordId(context.Background(), recordId)
	require.NoError(t, err)

	require.Equal(t, 2, len(matches))

	require.Equal(t, createdAt, matches[0].CreatedAt)
	require.Equal(t, "01JMPK4VF04QX714CG4PHYJ88K", matches[0].RecordId)
	require.Equal(t, "01JMKRNBW5TVN902YAE8GYZ367", matches[0].DeckId)
	require.Equal(t, "zor5SLfEfwfZ90yRVXzlxBEFARy2", matches[0].UserId)
	require.Equal(t, "KBp7roRDZobZg1t0OPzFR1kvLeO2", matches[0].OpponentsUserId)
	require.Equal(t, "memo", matches[0].Memo)

	require.Equal(t, "01JMPKHBXCJ32JZYNMDMY9SZ3B", matches[0].Games[0].ID)
	require.Equal(t, "01JMPKHAXQHQYJZ6VVASF5CATW", matches[0].Games[0].MatchId)
	require.Equal(t, "zor5SLfEfwfZ90yRVXzlxBEFARy2", matches[0].Games[0].UserId)
	require.Equal(t, false, matches[0].Games[0].GoFirst)
	require.Equal(t, false, matches[0].Games[0].WinningFlg)
	require.Equal(t, "memo1", matches[0].Games[0].Memo)

	require.Equal(t, "01JMPMPY964J0XBR7F5FTSGCDC", matches[0].Games[1].ID)
	require.Equal(t, "01JMPKHAXQHQYJZ6VVASF5CATW", matches[0].Games[1].MatchId)
	require.Equal(t, true, matches[0].Games[1].GoFirst)
	require.Equal(t, "zor5SLfEfwfZ90yRVXzlxBEFARy2", matches[0].Games[1].UserId)
	require.Equal(t, true, matches[0].Games[1].WinningFlg)
	require.Equal(t, "memo2", matches[0].Games[1].Memo)

	require.Equal(t, "01JMPMSN6RVW69EME7F1SGW5MD", matches[0].Games[2].ID)
	require.Equal(t, "01JMPKHAXQHQYJZ6VVASF5CATW", matches[0].Games[2].MatchId)
	require.Equal(t, "zor5SLfEfwfZ90yRVXzlxBEFARy2", matches[0].Games[2].UserId)
	require.Equal(t, false, matches[0].Games[2].GoFirst)
	require.Equal(t, true, matches[0].Games[2].WinningFlg)
	require.Equal(t, "memo3", matches[0].Games[2].Memo)

	require.Equal(t, createdAt, matches[1].CreatedAt)
	require.Equal(t, "01JMPK4VF04QX714CG4PHYJ88K", matches[1].RecordId)
	require.Equal(t, "01JMKRNBW5TVN902YAE8GYZ367", matches[1].DeckId)
	require.Equal(t, "zor5SLfEfwfZ90yRVXzlxBEFARy2", matches[1].UserId)
	require.Equal(t, "", matches[1].OpponentsUserId)
	require.Equal(t, "", matches[1].Memo)

	require.Equal(t, "01JMPKHM2CAECZ9F6V67ZY57N2", matches[1].Games[0].MatchId)
	require.Equal(t, "01JMPKHM7QD0X26JMWV23JY4M9", matches[1].Games[0].ID)

	require.Equal(t, "01JMPKHM2CAECZ9F6V67ZY57N2", matches[1].Games[1].MatchId)
	require.Equal(t, "01JMPM7WRQBDTYKH8BB921XX5K", matches[1].Games[1].ID)

	require.Equal(t, "01JMPKHAXQHQYJZ6VVASF5CATW", matches[0].ID)
	require.Equal(t, 3, len(matches[0].Games))

	require.Equal(t, "01JMPKHM2CAECZ9F6V67ZY57N2", matches[1].ID)
	require.Equal(t, 2, len(matches[1].Games))
}

func test_MatchInfrastructure_Create(t *testing.T) {
	r, mock, err := setup4MatchInfrastructure()
	require.NoError(t, err)

	datetime := time.Now().Truncate(0)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "matches" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
			`"record_id"=$4,"deck_id"=$5,"user_id"=$6,"opponents_user_id"=$7,`+
			`"bo3_flg"=$8,"qualifying_round_flg"=$9,"final_tournament_flg"=$10,`+
			`"default_victory_flg"=$11,"default_defeat_flg"=$12,"victory_flg"=$13,`+
			`"opponents_deck_info"=$14,"memo"=$15 `+
			`WHERE "matches"."deleted_at" IS NULL AND "id" = $16`,
	)).WithArgs(
		datetime,
		AnyTime{},
		gorm.DeletedAt{},
		"01JMPK4VF04QX714CG4PHYJ88K",
		"01JMKRNBW5TVN902YAE8GYZ367",
		"zor5SLfEfwfZ90yRVXzlxBEFARy2",
		"",
		false,
		false,
		false,
		false,
		false,
		true,
		"",
		"",
		"01JMPKHM2CAECZ9F6V67ZY57N2",
	).WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "games" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
			`"match_id"=$4,"user_id"=$5,"go_first"=$6,"winning_flg"=$7,`+
			`"your_prize_cards"=$8,"opponents_prize_cards"=$9,"memo"=$10 `+
			`WHERE "games"."deleted_at" IS NULL AND "id" = $11`,
	)).WithArgs(
		datetime,
		AnyTime{},
		gorm.DeletedAt{},
		"01JMPKHM2CAECZ9F6V67ZY57N2",
		"zor5SLfEfwfZ90yRVXzlxBEFARy2",
		true,
		true,
		6,
		5,
		"",
		"01JMPKHM7QD0X26JMWV23JY4M9",
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	var games []*entity.Game
	games = append(
		games,
		entity.NewGame(
			"01JMPKHM7QD0X26JMWV23JY4M9",
			datetime,
			"01JMPKHM2CAECZ9F6V67ZY57N2",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			true,
			true,
			6,
			5,
			"",
		),
	)

	match := entity.NewMatch(
		"01JMPKHM2CAECZ9F6V67ZY57N2",
		datetime,
		"01JMPK4VF04QX714CG4PHYJ88K",
		"01JMKRNBW5TVN902YAE8GYZ367",
		"zor5SLfEfwfZ90yRVXzlxBEFARy2",
		"",
		false,
		false,
		false,
		false,
		false,
		true,
		"",
		"",
		games,
	)

	require.NoError(t, r.Create(context.Background(), match))
	require.NoError(t, mock.ExpectationsWereMet())
}

func test_MatchInfrastructure_Update(t *testing.T) {
	r, mock, err := setup4MatchInfrastructure()
	require.NoError(t, err)

	{
		matchId := "01JJGBG44X1CZ6FZY39N1RQN9Z"
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		gameId1, _ := generateId()
		gameId2, _ := generateId()
		gameId3, _ := generateId()
		datetime := time.Now().Truncate(0)
		values := [][]driver.Value{
			{
				gameId1,
				datetime,
				datetime,
				gorm.DeletedAt{},
				matchId,
				userId,
				false,
				false,
				0,
				0,
				"",
			},
			{
				gameId2,
				datetime,
				datetime,
				gorm.DeletedAt{},
				matchId,
				userId,
				true,
				false,
				0,
				0,
				"",
			},
		}

		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"match_id",
			"user_id",
			"go_first",
			"winning_flg",
			"your_prize_cards",
			"opponents_prize_cards",
			"memo",
		}).AddRows(values...)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "games" WHERE match_id = $1 AND "games"."deleted_at" IS NULL ORDER BY created_at ASC`,
		)).WithArgs(
			matchId,
		).WillReturnRows(rows)

		mock.ExpectBegin()

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "matches" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"record_id"=$4,"deck_id"=$5,"user_id"=$6,"opponents_user_id"=$7,`+
				`"bo3_flg"=$8,"qualifying_round_flg"=$9,"final_tournament_flg"=$10,`+
				`"default_victory_flg"=$11,"default_defeat_flg"=$12,"victory_flg"=$13,`+
				`"opponents_deck_info"=$14,"memo"=$15 `+
				`WHERE "matches"."deleted_at" IS NULL AND "id" = $16`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			recordId,
			deckId,
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			true,
			"",
			"",
			matchId,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"match_id"=$4,"user_id"=$5,"go_first"=$6,"winning_flg"=$7,`+
				`"your_prize_cards"=$8,"opponents_prize_cards"=$9,"memo"=$10 `+
				`WHERE "games"."deleted_at" IS NULL AND "id" = $11`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			matchId,
			userId,
			false,
			false,
			0,
			0,
			"",
			gameId1,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"match_id"=$4,"user_id"=$5,"go_first"=$6,"winning_flg"=$7,`+
				`"your_prize_cards"=$8,"opponents_prize_cards"=$9,"memo"=$10 `+
				`WHERE "games"."deleted_at" IS NULL AND "id" = $11`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			matchId,
			userId,
			false,
			true,
			0,
			0,
			"",
			gameId2,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"match_id"=$4,"user_id"=$5,"go_first"=$6,"winning_flg"=$7,`+
				`"your_prize_cards"=$8,"opponents_prize_cards"=$9,"memo"=$10 `+
				`WHERE "games"."deleted_at" IS NULL AND "id" = $11`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			matchId,
			userId,
			true,
			true,
			0,
			0,
			"",
			gameId3,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		var games []*entity.Game
		games = append(
			games,
			entity.NewGame(
				gameId1,
				datetime,
				matchId,
				userId,
				false,
				false,
				0,
				0,
				"",
			),
			entity.NewGame(
				gameId2,
				datetime,
				matchId,
				userId,
				false,
				true,
				0,
				0,
				"",
			),
			entity.NewGame(
				gameId3,
				datetime,
				matchId,
				userId,
				true,
				true,
				0,
				0,
				"",
			),
		)

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			true,
			"",
			"",
			games,
		)

		require.NoError(t, r.Update(context.Background(), match))
		require.NoError(t, mock.ExpectationsWereMet())
	}

	{
		matchId := "01JJGBG44X1CZ6FZY39N1RQN9Z"
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		gameId1, _ := generateId()
		gameId2, _ := generateId()
		gameId3, _ := generateId()
		datetime := time.Now().Truncate(0)
		values := [][]driver.Value{
			{
				gameId1,
				datetime,
				datetime,
				gorm.DeletedAt{},
				matchId,
				userId,
				false,
				false,
				0,
				0,
				"",
			},
			{
				gameId2,
				datetime,
				datetime,
				gorm.DeletedAt{},
				matchId,
				userId,
				true,
				false,
				0,
				0,
				"",
			},
			{
				gameId3,
				datetime,
				datetime,
				gorm.DeletedAt{},
				matchId,
				userId,
				false,
				true,
				0,
				0,
				"",
			},
		}

		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"match_id",
			"user_id",
			"go_first",
			"winning_flg",
			"your_prize_cards",
			"opponents_prize_cards",
			"memo",
		}).AddRows(values...)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "games" WHERE match_id = $1 AND "games"."deleted_at" IS NULL ORDER BY created_at ASC`,
		)).WithArgs(
			matchId,
		).WillReturnRows(rows)

		mock.ExpectBegin()

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "matches" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"record_id"=$4,"deck_id"=$5,"user_id"=$6,"opponents_user_id"=$7,`+
				`"bo3_flg"=$8,"qualifying_round_flg"=$9,"final_tournament_flg"=$10,`+
				`"default_victory_flg"=$11,"default_defeat_flg"=$12,"victory_flg"=$13,`+
				`"opponents_deck_info"=$14,"memo"=$15 `+
				`WHERE "matches"."deleted_at" IS NULL AND "id" = $16`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			recordId,
			deckId,
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			matchId,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"match_id"=$4,"user_id"=$5,"go_first"=$6,"winning_flg"=$7,`+
				`"your_prize_cards"=$8,"opponents_prize_cards"=$9,"memo"=$10 `+
				`WHERE "games"."deleted_at" IS NULL AND "id" = $11`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			matchId,
			userId,
			true,
			false,
			0,
			0,
			"",
			gameId1,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"match_id"=$4,"user_id"=$5,"go_first"=$6,"winning_flg"=$7,`+
				`"your_prize_cards"=$8,"opponents_prize_cards"=$9,"memo"=$10 `+
				`WHERE "games"."deleted_at" IS NULL AND "id" = $11`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			matchId,
			userId,
			true,
			false,
			0,
			0,
			"",
			gameId2,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "deleted_at"=$1 WHERE id = $2 AND "games"."deleted_at" IS NULL`,
		)).WithArgs(
			AnyTime{},
			gameId3,
		).WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectCommit()

		var games []*entity.Game
		games = append(
			games,
			entity.NewGame(
				gameId1,
				datetime,
				matchId,
				userId,
				true,
				false,
				0,
				0,
				"",
			),
			entity.NewGame(
				gameId2,
				datetime,
				matchId,
				userId,
				true,
				false,
				0,
				0,
				"",
			),
		)

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			games,
		)

		require.NoError(t, r.Update(context.Background(), match))
		require.NoError(t, mock.ExpectationsWereMet())
	}

	{
		matchId := "01JJGBG44X1CZ6FZY39N1RQN9Z"
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		gameId1, _ := generateId()
		gameId2, _ := generateId()
		datetime := time.Now().Truncate(0)
		values := [][]driver.Value{
			{
				gameId1,
				datetime,
				datetime,
				gorm.DeletedAt{},
				matchId,
				userId,
				false,
				false,
				0,
				0,
				"",
			},
			{
				gameId2,
				datetime,
				datetime,
				gorm.DeletedAt{},
				matchId,
				userId,
				true,
				false,
				0,
				0,
				"",
			},
		}

		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"match_id",
			"user_id",
			"go_first",
			"winning_flg",
			"your_prize_cards",
			"opponents_prize_cards",
			"memo",
		}).AddRows(values...)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "games" WHERE match_id = $1 AND "games"."deleted_at" IS NULL ORDER BY created_at ASC`,
		)).WithArgs(
			matchId,
		).WillReturnRows(rows)

		mock.ExpectBegin()

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "matches" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"record_id"=$4,"deck_id"=$5,"user_id"=$6,"opponents_user_id"=$7,`+
				`"bo3_flg"=$8,"qualifying_round_flg"=$9,"final_tournament_flg"=$10,`+
				`"default_victory_flg"=$11,"default_defeat_flg"=$12,"victory_flg"=$13,`+
				`"opponents_deck_info"=$14,"memo"=$15 `+
				`WHERE "matches"."deleted_at" IS NULL AND "id" = $16`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			recordId,
			deckId,
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			true,
			"",
			"",
			matchId,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"match_id"=$4,"user_id"=$5,"go_first"=$6,"winning_flg"=$7,`+
				`"your_prize_cards"=$8,"opponents_prize_cards"=$9,"memo"=$10 `+
				`WHERE "games"."deleted_at" IS NULL AND "id" = $11`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			matchId,
			userId,
			false,
			true,
			0,
			0,
			"",
			gameId1,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "deleted_at"=$1 WHERE id = $2 AND "games"."deleted_at" IS NULL`,
		)).WithArgs(
			AnyTime{},
			gameId2,
		).WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectCommit()

		var games []*entity.Game
		games = append(
			games,
			entity.NewGame(
				gameId1,
				datetime,
				matchId,
				userId,
				false,
				true,
				0,
				0,
				"",
			),
		)

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			true,
			"",
			"",
			games,
		)

		require.NoError(t, r.Update(context.Background(), match))
		require.NoError(t, mock.ExpectationsWereMet())
	}

	{
		matchId := "01JJGBG44X1CZ6FZY39N1RQN9Z"
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		gameId1, _ := generateId()
		gameId2, _ := generateId()
		datetime := time.Now().Truncate(0)
		values := [][]driver.Value{
			{
				gameId1,
				datetime,
				datetime,
				gorm.DeletedAt{},
				matchId,
				userId,
				false,
				false,
				0,
				0,
				"",
			},
		}

		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"match_id",
			"user_id",
			"go_first",
			"winning_flg",
			"your_prize_cards",
			"opponents_prize_cards",
			"memo",
		}).AddRows(values...)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "games" WHERE match_id = $1 AND "games"."deleted_at" IS NULL ORDER BY created_at ASC`,
		)).WithArgs(
			matchId,
		).WillReturnRows(rows)

		mock.ExpectBegin()

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "matches" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"record_id"=$4,"deck_id"=$5,"user_id"=$6,"opponents_user_id"=$7,`+
				`"bo3_flg"=$8,"qualifying_round_flg"=$9,"final_tournament_flg"=$10,`+
				`"default_victory_flg"=$11,"default_defeat_flg"=$12,"victory_flg"=$13,`+
				`"opponents_deck_info"=$14,"memo"=$15 `+
				`WHERE "matches"."deleted_at" IS NULL AND "id" = $16`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			recordId,
			deckId,
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			matchId,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"match_id"=$4,"user_id"=$5,"go_first"=$6,"winning_flg"=$7,`+
				`"your_prize_cards"=$8,"opponents_prize_cards"=$9,"memo"=$10 `+
				`WHERE "games"."deleted_at" IS NULL AND "id" = $11`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			matchId,
			userId,
			true,
			false,
			0,
			0,
			"",
			gameId1,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"match_id"=$4,"user_id"=$5,"go_first"=$6,"winning_flg"=$7,`+
				`"your_prize_cards"=$8,"opponents_prize_cards"=$9,"memo"=$10 `+
				`WHERE "games"."deleted_at" IS NULL AND "id" = $11`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			matchId,
			userId,
			false,
			false,
			0,
			0,
			"",
			gameId2,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		var games []*entity.Game
		games = append(
			games,
			entity.NewGame(
				gameId1,
				datetime,
				matchId,
				userId,
				true,
				false,
				0,
				0,
				"",
			),
			entity.NewGame(
				gameId2,
				datetime,
				matchId,
				userId,
				false,
				false,
				0,
				0,
				"",
			),
		)

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			games,
		)

		require.NoError(t, r.Update(context.Background(), match))
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func test_MatchInfrastructure_Delete(t *testing.T) {
	r, mock, err := setup4MatchInfrastructure()
	require.NoError(t, err)

	matchId := "01JMPKHM2CAECZ9F6V67ZY57N2"
	gameId := "01JMPKHM7QD0X26JMWV23JY4M9"

	createdAt := time.Now().Truncate(0)
	updatedAt := time.Now().Truncate(0)

	values := [][]driver.Value{
		{
			matchId,
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			"01JMPK4VF04QX714CG4PHYJ88K",
			"",
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			"",
			false,
			true,
			true,
			true,
			true,
			true,
			"Test3",
			"",
			gameId,
			createdAt,
			updatedAt,
			gorm.DeletedAt{},
			matchId,
			"zor5SLfEfwfZ90yRVXzlxBEFARy2",
			true,
			true,
			6,
			5,
			"",
		},
	}
	rows := sqlmock.NewRows([]string{
		"match_id",
		"match_created_at",
		"match_updated_at",
		"match_deleted_at",
		"match_record_id",
		"match_deck_id",
		"match_user_id",
		"match_opponents_user_id",
		"match_bo3_flg",
		"match_qualifying_round_flg",
		"match_final_tournament_flg",
		"match_default_victory_flg",
		"match_default_defeat_flg",
		"match_victory_flg",
		"match_opponents_deck_info",
		"match_memo",
		"game_id",
		"game_created_at",
		"game_updated_at",
		"game_deleted_at",
		"game_match_id",
		"game_user_id",
		"game_go_first",
		"game_winning_flg",
		"game_your_prize_cards",
		"game_opponents_prize_cards",
		"game_memo",
	}).AddRows(values...)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT matches.id AS match_id,matches.created_at AS match_created_at,matches.updated_at AS match_updated_at,matches.deleted_at AS match_deleted_at,matches.record_id AS match_record_id,matches.deck_id AS match_deck_id,matches.user_id AS match_user_id,matches.opponents_user_id AS match_opponents_user_id,matches.bo3_flg AS match_bo3_flg,matches.qualifying_round_flg AS match_qualifying_round_flg,matches.final_tournament_flg AS match_final_tournament_flg,matches.default_victory_flg AS match_default_victory_flg,matches.default_defeat_flg AS match_default_defeat_flg,matches.victory_flg AS match_victory_flg,matches.opponents_deck_info AS match_opponents_deck_info,matches.memo AS match_memo,games.id AS game_id,games.created_at AS game_created_at,games.updated_at AS game_updated_at,games.deleted_at AS game_deleted_at,games.match_id AS game_match_id, games.user_id AS game_user_id, games.go_first AS game_go_first, games.winning_flg AS game_winning_flg,games.your_prize_cards AS game_your_prize_cards,games.opponents_prize_cards AS game_opponents_prize_cards,games.memo AS game_memo FROM "matches" INNER JOIN games on matches.id = games.match_id WHERE matches.id = $1 AND matches.deleted_at IS NULL ORDER BY games.created_at ASC`,
	)).WithArgs(
		matchId,
	).WillReturnRows(rows)

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "games" SET "deleted_at"=$1 WHERE id = $2 AND "games"."deleted_at" IS NULL`,
	)).WithArgs(
		AnyTime{},
		gameId,
	).WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "matches" SET "deleted_at"=$1 WHERE id = $2 AND "matches"."deleted_at" IS NULL`,
	)).WithArgs(
		AnyTime{},
		matchId,
	).WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectCommit()

	require.NoError(t, r.Delete(context.Background(), matchId))
	require.NoError(t, mock.ExpectationsWereMet())
}
