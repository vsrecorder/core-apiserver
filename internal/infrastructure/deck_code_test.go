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
		"FindIdsByUserId": test_DeckCodeInfrastructure_FindIdsByUserId,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_DeckCodeInfrastructure_FindIdsByUserId(t *testing.T) {
	r, mock, err := setup4DeckCodeInfrastructure()
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"id",
	}).AddRow(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT "id" FROM "deck_codes" WHERE user_id = $1 AND "deck_codes"."deleted_at" IS NULL`,
	)).WithArgs(
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
	).WillReturnRows(rows)

	ids, err := r.FindIdsByUserId(context.Background(), "CeQ0Oa9g9uRThL11lj4l45VAg8p1")

	require.NoError(t, err)
	require.Equal(t, []string{"01HD7Y3K8D6FDHMHTZ2GT41TN2"}, ids)
}
