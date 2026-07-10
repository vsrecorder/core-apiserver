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

func setup4OldestRecordInfrastructure() (repository.OldestRecordInterface, sqlmock.Sqlmock, error) {
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

	return NewOldestRecord(db), mock, nil
}

func TestOldestRecordInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"ReturnsOldestEventDate":         test_OldestRecordInfrastructure_ReturnsOldestEventDate,
		"NoMatchingRecordsReturnsNil":    test_OldestRecordInfrastructure_NoMatchingRecordsReturnsNil,
		"FilterByDeckIdIsAppliedToWhere": test_OldestRecordInfrastructure_FilterByDeckIdIsAppliedToWhere,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_OldestRecordInfrastructure_ReturnsOldestEventDate(t *testing.T) {
	i, mock, err := setup4OldestRecordInfrastructure()
	require.NoError(t, err)

	userId := "user-01"
	expected := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"event_date"}).AddRow(expected)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT MIN(event_date) AS event_date FROM "records" WHERE user_id = $1 AND deleted_at IS NULL AND event_date IS NOT NULL`,
	)).WithArgs(userId).WillReturnRows(rows)

	record, err := i.FindOldestRecord(context.Background(), userId, "")

	require.NoError(t, err)
	require.NotNil(t, record.EventDate)
	require.True(t, expected.Equal(*record.EventDate))
}

func test_OldestRecordInfrastructure_NoMatchingRecordsReturnsNil(t *testing.T) {
	i, mock, err := setup4OldestRecordInfrastructure()
	require.NoError(t, err)

	userId := "user-02"

	// MIN()は集約関数のため、該当行が無くてもNULL値を持つ1行が返る
	rows := sqlmock.NewRows([]string{"event_date"}).AddRow(nil)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT MIN(event_date) AS event_date FROM "records" WHERE user_id = $1 AND deleted_at IS NULL AND event_date IS NOT NULL`,
	)).WithArgs(userId).WillReturnRows(rows)

	record, err := i.FindOldestRecord(context.Background(), userId, "")

	require.NoError(t, err)
	require.Nil(t, record.EventDate)
}

func test_OldestRecordInfrastructure_FilterByDeckIdIsAppliedToWhere(t *testing.T) {
	i, mock, err := setup4OldestRecordInfrastructure()
	require.NoError(t, err)

	userId := "user-03"
	deckId := "deck-01"
	expected := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"event_date"}).AddRow(expected)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT MIN(event_date) AS event_date FROM "records" WHERE (user_id = $1 AND deleted_at IS NULL AND event_date IS NOT NULL) AND deck_id = $2`,
	)).WithArgs(userId, deckId).WillReturnRows(rows)

	record, err := i.FindOldestRecord(context.Background(), userId, deckId)

	require.NoError(t, err)
	require.NotNil(t, record.EventDate)
	require.True(t, expected.Equal(*record.EventDate))
}
