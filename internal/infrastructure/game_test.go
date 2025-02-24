package infrastructure

import (
	"context"
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

func setupMock4GameInfrastructure() (*gorm.DB, sqlmock.Sqlmock, error) {
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

func setup4GameInfrastructure() (repository.GameInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock4GameInfrastructure()

	if err != nil {
		return nil, nil, err
	}

	r := NewGame(db)

	return r, mock, err
}

func TestGameInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"FindById":      test_GameInfrastructure_FindById,
		"FindByMatchId": test_GameInfrastructure_FindByMatchId,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_GameInfrastructure_FindById(t *testing.T) {
	r, mock, err := setup4GameInfrastructure()
	require.NoError(t, err)

	id := "01JJGBG4EGM1EZVWK02TEVQ5F0"
	matchId := "01JJGBG44X1CZ6FZY39N1RQN9Z"
	userId := "d4385mX98abtmLny3qxlmBlBLIu1"
	datetime := time.Now().Truncate(0)

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
	}).AddRow(
		id,
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
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "games" WHERE id = $1 AND "games"."deleted_at" IS NULL ORDER BY "games"."id" LIMIT $2`,
	)).WithArgs(
		id,
		1,
	).WillReturnRows(rows)

	game, err := r.FindById(context.Background(), id)

	require.NoError(t, err)
	require.IsType(t, &entity.Game{}, game)
	require.Equal(t, id, game.ID)
	require.Equal(t, matchId, game.MatchId)
	require.Equal(t, userId, game.UserId)
}

func test_GameInfrastructure_FindByMatchId(t *testing.T) {
	r, mock, err := setup4GameInfrastructure()
	require.NoError(t, err)

	id := "01JJGBG4EGM1EZVWK02TEVQ5F0"
	matchId := "01JJGBG44X1CZ6FZY39N1RQN9Z"
	userId := "d4385mX98abtmLny3qxlmBlBLIu1"
	datetime := time.Now().Truncate(0)

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
	}).AddRow(
		id,
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
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "games" WHERE match_id = $1 AND "games"."deleted_at" IS NULL ORDER BY created_at ASC`,
	)).WithArgs(
		id,
	).WillReturnRows(rows)

	game, err := r.FindByMatchId(context.Background(), id)

	require.NoError(t, err)
	require.IsType(t, []*entity.Game{}, game)
	require.Equal(t, id, game[0].ID)
	require.Equal(t, matchId, game[0].MatchId)
	require.Equal(t, userId, game[0].UserId)
}
