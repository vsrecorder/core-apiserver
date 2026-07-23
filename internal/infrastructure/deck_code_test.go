package infrastructure

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func setupMock4DeckCodeInfrastructure() (*gorm.DB, sqlmock.Sqlmock, error) {
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

func setup4DeckCodeInfrastructure() (repository.DeckCodeInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock4DeckCodeInfrastructure()

	if err != nil {
		return nil, nil, err
	}

	r := NewDeckCode(db)

	return r, mock, err
}

func TestDeckCodeInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"DeleteByUserId": test_DeckCodeInfrastructure_DeleteByUserId,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

// 退会時の一括削除。作成者が本人であるデッキコードを、件数によらず1文で削除する。
// (本人のデッキに紐づくものは Deck.DeleteByUserId 側で消える)
func test_DeckCodeInfrastructure_DeleteByUserId(t *testing.T) {
	r, mock, err := setup4DeckCodeInfrastructure()
	require.NoError(t, err)

	uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"

	// GORM は単発の書き込みも既定でトランザクションに包む
	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "deck_codes" SET "deleted_at"=$1 WHERE user_id = $2 AND "deck_codes"."deleted_at" IS NULL`,
	)).WithArgs(
		AnyTime{},
		uid,
	).WillReturnResult(sqlmock.NewResult(0, 2))

	mock.ExpectCommit()

	err = r.DeleteByUserId(context.Background(), uid)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
