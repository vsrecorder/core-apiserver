package infrastructure

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func setup4CityleagueResultInfrastructure() (repository.CityleagueResultInterface, sqlmock.Sqlmock, error) {
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

	return NewCityleagueResult(db), mock, nil
}

func TestCityleagueResultInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"FindEvents":                  test_CityleagueResultInfrastructure_FindEvents,
		"FindEventsWithLeagueType":    test_CityleagueResultInfrastructure_FindEventsWithLeagueType,
		"FindEventsWithTerm":          test_CityleagueResultInfrastructure_FindEventsWithTerm,
		"FindEventsReturnsEmptySlice": test_CityleagueResultInfrastructure_FindEventsReturnsEmptySlice,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

// leagueType が 0 かつ期間の指定が無い場合、絞り込み条件を付けずに全イベントを返す。
func test_CityleagueResultInfrastructure_FindEvents(t *testing.T) {
	r, mock, err := setup4CityleagueResultInfrastructure()
	require.NoError(t, err)

	eventDate, err := time.Parse(time.RFC3339, "2026-04-30T00:00:00Z")
	require.NoError(t, err)

	values := [][]driver.Value{
		{uint(952749), uint(1), eventDate},
		{uint(952750), uint(2), eventDate},
	}

	rows := sqlmock.NewRows([]string{"official_event_id", "league_type", "event_date"})
	for _, value := range values {
		rows.AddRow(value...)
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT DISTINCT "official_event_id","league_type","event_date" FROM "cityleague_results" ORDER BY event_date DESC, league_type ASC, official_event_id ASC`,
	)).WillReturnRows(rows)

	ret, err := r.FindEvents(context.Background(), 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, ret, 2)
	require.Equal(t, uint(952749), ret[0].OfficialEventId)
	require.Equal(t, uint(1), ret[0].LeagueType)
	require.Equal(t, eventDate, ret[0].EventDate)
	require.Equal(t, uint(952750), ret[1].OfficialEventId)
	require.Equal(t, uint(2), ret[1].LeagueType)
}

// leagueType が指定された場合、league_type で絞り込む。
func test_CityleagueResultInfrastructure_FindEventsWithLeagueType(t *testing.T) {
	r, mock, err := setup4CityleagueResultInfrastructure()
	require.NoError(t, err)

	eventDate, err := time.Parse(time.RFC3339, "2026-04-30T00:00:00Z")
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"official_event_id", "league_type", "event_date"}).
		AddRow(uint(952749), uint(1), eventDate)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT DISTINCT "official_event_id","league_type","event_date" FROM "cityleague_results" WHERE league_type = $1 ORDER BY event_date DESC, league_type ASC, official_event_id ASC`,
	)).WithArgs(uint(1)).WillReturnRows(rows)

	ret, err := r.FindEvents(context.Background(), 1, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, ret, 1)
	require.Equal(t, uint(952749), ret[0].OfficialEventId)
}

// fromDate と toDate が指定された場合、event_date で絞り込む。
func test_CityleagueResultInfrastructure_FindEventsWithTerm(t *testing.T) {
	r, mock, err := setup4CityleagueResultInfrastructure()
	require.NoError(t, err)

	fromDate, err := time.Parse(time.RFC3339, "2026-04-01T00:00:00Z")
	require.NoError(t, err)
	toDate, err := time.Parse(time.RFC3339, "2026-04-30T00:00:00Z")
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"official_event_id", "league_type", "event_date"}).
		AddRow(uint(952749), uint(1), toDate)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT DISTINCT "official_event_id","league_type","event_date" FROM "cityleague_results" WHERE event_date >= $1 AND event_date <= $2 ORDER BY event_date DESC, league_type ASC, official_event_id ASC`,
	)).WithArgs(fromDate, toDate).WillReturnRows(rows)

	ret, err := r.FindEvents(context.Background(), 0, fromDate, toDate)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, ret, 1)
	require.Equal(t, uint(952749), ret[0].OfficialEventId)
}

// 該当が無い場合は、エラーではなく空のスライスを返す。
func test_CityleagueResultInfrastructure_FindEventsReturnsEmptySlice(t *testing.T) {
	r, mock, err := setup4CityleagueResultInfrastructure()
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"official_event_id", "league_type", "event_date"})

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT DISTINCT "official_event_id","league_type","event_date" FROM "cityleague_results"`,
	)).WillReturnRows(rows)

	ret, err := r.FindEvents(context.Background(), 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.NotNil(t, ret)
	require.Empty(t, ret)
}
