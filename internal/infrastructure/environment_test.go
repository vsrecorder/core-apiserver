package infrastructure

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMock4EnvironmentInfrastructure() (*gorm.DB, sqlmock.Sqlmock, error) {
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

func setup4EnvironmentInfrastructure() (repository.EnvironmentInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock4EnvironmentInfrastructure()

	if err != nil {
		return nil, nil, err
	}

	r := NewEnvironment(db)

	return r, mock, err
}

func TestEnvironmentInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"Find":       test_EnvironmentInfrastructure_Find,
		"FindById":   test_EnvironmentInfrastructure_FindById,
		"FindByDate": test_EnvironmentInfrastructure_FindByDate,
		"FindByTerm": test_EnvironmentInfrastructure_FindByTerm,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_EnvironmentInfrastructure_Find(t *testing.T) {
	r, mock, err := setup4EnvironmentInfrastructure()
	require.NoError(t, err)

	id := "sv11"
	title := "ブラックボルト/ホワイトフレア"
	fromDate, _ := time.Parse(DateLayout, "2025-06-06")
	toDate, _ := time.Parse(DateLayout, "2025-07-31")

	rows := sqlmock.NewRows([]string{
		"id",
		"title",
		"from_date",
		"to_date",
	}).AddRow(
		id,
		title,
		fromDate,
		toDate,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "environments" ORDER BY from_date DESC`,
	)).WithArgs().WillReturnRows(rows)

	environments, err := r.Find(context.Background())

	require.NoError(t, err)
	require.Equal(t, id, environments[0].ID)
	require.Equal(t, title, environments[0].Title)
	require.Equal(t, fromDate, environments[0].FromDate)
	require.Equal(t, toDate, environments[0].ToDate)
}

func test_EnvironmentInfrastructure_FindById(t *testing.T) {
	r, mock, err := setup4EnvironmentInfrastructure()
	require.NoError(t, err)

	id := "sv11"
	title := "ブラックボルト/ホワイトフレア"
	fromDate, _ := time.Parse(DateLayout, "2025-06-06")
	toDate, _ := time.Parse(DateLayout, "2025-07-31")

	rows := sqlmock.NewRows([]string{
		"id",
		"title",
		"from_date",
		"to_date",
	}).AddRow(
		id,
		title,
		fromDate,
		toDate,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "environments" WHERE id = $1 ORDER BY "environments"."id" LIMIT $2`,
	)).WithArgs(
		id,
		1,
	).WillReturnRows(rows)

	environment, err := r.FindById(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, environment.ID)
	require.Equal(t, title, environment.Title)
	require.Equal(t, fromDate, environment.FromDate)
	require.Equal(t, toDate, environment.ToDate)
}

func test_EnvironmentInfrastructure_FindByDate(t *testing.T) {
	r, mock, err := setup4EnvironmentInfrastructure()
	require.NoError(t, err)

	date, _ := time.Parse(DateLayout, "2025-06-06")

	id := "sv11"
	title := "ブラックボルト/ホワイトフレア"
	fromDate, _ := time.Parse(DateLayout, "2025-06-06")
	toDate, _ := time.Parse(DateLayout, "2025-07-31")

	rows := sqlmock.NewRows([]string{
		"id",
		"title",
		"from_date",
		"to_date",
	}).AddRow(
		id,
		title,
		fromDate,
		toDate,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "environments" WHERE from_date <= $1 ORDER BY from_date DESC,"environments"."id" LIMIT $2`,
	)).WithArgs(
		date,
		1,
	).WillReturnRows(rows)

	environment, err := r.FindByDate(context.Background(), date)

	require.NoError(t, err)
	require.Equal(t, id, environment.ID)
	require.Equal(t, title, environment.Title)
	require.Equal(t, fromDate, environment.FromDate)
	require.Equal(t, toDate, environment.ToDate)
}

func test_EnvironmentInfrastructure_FindByTerm(t *testing.T) {
	r, mock, err := setup4EnvironmentInfrastructure()
	require.NoError(t, err)

	argFromDate, _ := time.Parse(DateLayout, "2025-06-06")
	argToDate, _ := time.Parse(DateLayout, "2025-06-07")

	id := "sv11"
	title := "ブラックボルト/ホワイトフレア"
	fromDate, _ := time.Parse(DateLayout, "2025-06-06")
	toDate, _ := time.Parse(DateLayout, "2025-07-31")

	rows := sqlmock.NewRows([]string{
		"id",
		"title",
		"from_date",
		"to_date",
	}).AddRow(
		id,
		title,
		fromDate,
		toDate,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "environments" WHERE to_date >= $1 AND from_date <= $2 ORDER BY from_date DESC`,
	)).WithArgs(
		argFromDate,
		argToDate,
	).WillReturnRows(rows)

	environments, err := r.FindByTerm(context.Background(), argFromDate, argToDate)

	require.NoError(t, err)
	require.Equal(t, id, environments[0].ID)
	require.Equal(t, title, environments[0].Title)
	require.Equal(t, fromDate, environments[0].FromDate)
	require.Equal(t, toDate, environments[0].ToDate)
}
