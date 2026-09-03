package infrastructure

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func setup4ChampionsleagueResultInfrastructure(t *testing.T) (
	repository.ChampionsleagueResultInterface,
	sqlmock.Sqlmock,
) {
	t.Helper()

	db, mock := setupSqlmockDB(t)

	return NewChampionsleagueResult(db), mock
}

func TestChampionsleagueResultInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"FindEvents":                                test_ChampionsleagueResultInfrastructure_FindEvents,
		"FindEventsReturnsEmptySlice":               test_ChampionsleagueResultInfrastructure_FindEventsReturnsEmptySlice,
		"FindByChampionsleagueScheduleId":           test_ChampionsleagueResultInfrastructure_FindByChampionsleagueScheduleId,
		"FindByChampionsleagueScheduleIdLeagueType": test_ChampionsleagueResultInfrastructure_FindByChampionsleagueScheduleIdLeagueType,
		"FindByChampionsleagueScheduleIdNotFound":   test_ChampionsleagueResultInfrastructure_FindByChampionsleagueScheduleIdNotFound,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_ChampionsleagueResultInfrastructure_FindEvents(t *testing.T) {
	r, mock := setup4ChampionsleagueResultInfrastructure(t)

	eventDate, err := time.Parse(time.RFC3339, "2026-06-07T00:00:00Z")
	require.NoError(t, err)

	values := [][]driver.Value{
		{"pjcs2026", uint(1032135), uint(4), eventDate},
		{"pjcs2026", uint(1032136), uint(3), eventDate},
	}

	rows := sqlmock.NewRows([]string{
		"championsleague_schedule_id", "official_event_id", "league_type", "event_date",
	})
	for _, value := range values {
		rows.AddRow(value...)
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT DISTINCT "championsleague_schedule_id","official_event_id","league_type","event_date" FROM "championsleague_results" ORDER BY event_date DESC, league_type DESC, official_event_id ASC`,
	)).WillReturnRows(rows)

	ret, err := r.FindEvents(context.Background())
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, ret, 2)
	require.Equal(t, "pjcs2026", ret[0].ChampionsleagueScheduleId)
	require.Equal(t, uint(1032135), ret[0].OfficialEventId)
	require.Equal(t, uint(4), ret[0].LeagueType)
	require.Equal(t, eventDate, ret[0].EventDate)
	require.Equal(t, uint(1032136), ret[1].OfficialEventId)
}

// 1件も無い場合は nil ではなく空スライスを返す(呼び出し側で len だけ見れば済むようにする)。
func test_ChampionsleagueResultInfrastructure_FindEventsReturnsEmptySlice(t *testing.T) {
	r, mock := setup4ChampionsleagueResultInfrastructure(t)

	rows := sqlmock.NewRows([]string{
		"championsleague_schedule_id", "official_event_id", "league_type", "event_date",
	})

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT DISTINCT`)).WillReturnRows(rows)

	ret, err := r.FindEvents(context.Background())
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.NotNil(t, ret)
	require.Empty(t, ret)
}

// 入賞者1人=1レコードで返る行を、official_event_id ごとにまとめ直す。
func test_ChampionsleagueResultInfrastructure_FindByChampionsleagueScheduleId(t *testing.T) {
	r, mock := setup4ChampionsleagueResultInfrastructure(t)

	eventDate, err := time.Parse(time.RFC3339, "2026-06-07T00:00:00Z")
	require.NoError(t, err)

	values := [][]driver.Value{
		{"pjcs2026", uint(1032135), uint(4), eventDate, "0000000001", "テスト太郎", uint(1), "aaaaaa-bbbbbb-cccccc"},
		{"pjcs2026", uint(1032135), uint(4), eventDate, "0000000002", "テスト次郎", uint(2), "dddddd-eeeeee-ffffff"},
		{"pjcs2026", uint(1032136), uint(3), eventDate, "0000000003", "テスト三郎", uint(1), "gggggg-hhhhhh-iiiiii"},
	}

	rows := sqlmock.NewRows([]string{
		"championsleague_schedule_id", "official_event_id", "league_type", "event_date",
		"player_id", "player_name", "rank", "deck_code",
	})
	for _, value := range values {
		rows.AddRow(value...)
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "championsleague_results" WHERE championsleague_schedule_id = $1 ORDER BY event_date DESC, league_type DESC, official_event_id ASC, rank ASC, player_id ASC`,
	)).WithArgs("pjcs2026").WillReturnRows(rows)

	ret, err := r.FindByChampionsleagueScheduleId(context.Background(), 0, "pjcs2026")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, ret, 2)

	require.Equal(t, uint(1032135), ret[0].OfficialEventId)
	require.Equal(t, uint(4), ret[0].LeagueType)
	require.Len(t, ret[0].EventResults, 2)
	require.Equal(t, "テスト太郎", ret[0].EventResults[0].PlayerName)
	require.Equal(t, uint(1), ret[0].EventResults[0].Rank)
	require.Equal(t, "dddddd-eeeeee-ffffff", ret[0].EventResults[1].DeckCode)

	require.Equal(t, uint(1032136), ret[1].OfficialEventId)
	require.Len(t, ret[1].EventResults, 1)
	require.Equal(t, "テスト三郎", ret[1].EventResults[0].PlayerName)
}

// 結果が未登録の大会は「存在しない」として扱う(webapp 側は 404 で notFound に流す)。
func test_ChampionsleagueResultInfrastructure_FindByChampionsleagueScheduleIdNotFound(t *testing.T) {
	r, mock := setup4ChampionsleagueResultInfrastructure(t)

	rows := sqlmock.NewRows([]string{
		"championsleague_schedule_id", "official_event_id", "league_type", "event_date",
		"player_id", "player_name", "rank", "deck_code",
	})

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "championsleague_results"`)).
		WithArgs("cl2027_yokohama").
		WillReturnRows(rows)

	ret, err := r.FindByChampionsleagueScheduleId(context.Background(), 0, "cl2027_yokohama")
	require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	require.Nil(t, ret)
	require.NoError(t, mock.ExpectationsWereMet())
}

// leagueType が指定された場合は league_type でも絞り込む。
func test_ChampionsleagueResultInfrastructure_FindByChampionsleagueScheduleIdLeagueType(t *testing.T) {
	r, mock := setup4ChampionsleagueResultInfrastructure(t)

	eventDate, err := time.Parse(time.RFC3339, "2026-06-07T00:00:00Z")
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"championsleague_schedule_id", "official_event_id", "league_type", "event_date",
		"player_id", "player_name", "rank", "deck_code",
	}).AddRow(
		"pjcs2026", uint(1032135), uint(4), eventDate,
		"0000000001", "テスト太郎", uint(1), "aaaaaa-aaaaaa-aaaaaa",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "championsleague_results" WHERE championsleague_schedule_id = $1 AND league_type = $2 ORDER BY event_date DESC, league_type DESC, official_event_id ASC, rank ASC, player_id ASC`,
	)).WithArgs("pjcs2026", uint(4)).WillReturnRows(rows)

	ret, err := r.FindByChampionsleagueScheduleId(context.Background(), 4, "pjcs2026")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, ret, 1)
	require.Equal(t, uint(4), ret[0].LeagueType)
	require.Len(t, ret[0].EventResults, 1)
}
