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
		"FindEvents":                          test_CityleagueResultInfrastructure_FindEvents,
		"FindEventsWithLeagueType":            test_CityleagueResultInfrastructure_FindEventsWithLeagueType,
		"FindEventsWithTerm":                  test_CityleagueResultInfrastructure_FindEventsWithTerm,
		"FindEventsReturnsEmptySlice":         test_CityleagueResultInfrastructure_FindEventsReturnsEmptySlice,
		"FindByPlayerId":                      test_CityleagueResultInfrastructure_FindByPlayerId,
		"FindByPlayerIdWithoutTerm":           test_CityleagueResultInfrastructure_FindByPlayerIdWithoutTerm,
		"FindByPlayerIdReturnsEmptySlice":     test_CityleagueResultInfrastructure_FindByPlayerIdReturnsEmptySlice,
		"FindByPlayerIdWithNullOfficialEvent": test_CityleagueResultInfrastructure_FindByPlayerIdWithNullOfficialEvent,
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

// FindByPlayerId が生成するSELECT句(official_events・shops・prefectures の結合込み)。
// テストごとに書き下すと差分が読みにくくなるため定数にまとめる。
const findByPlayerIdSelect = `SELECT cityleague_results.cityleague_schedule_id AS cityleague_schedule_id,` +
	`cityleague_results.official_event_id AS official_event_id,` +
	`cityleague_results.league_type AS league_type,` +
	`cityleague_results.event_date AS event_date,` +
	`cityleague_results.rank AS rank,` +
	`cityleague_results.point AS point,` +
	`cityleague_results.deck_code AS deck_code,` +
	`official_events.title AS event_title,` +
	`official_events.shop_name AS shop_name,` +
	`prefectures.name AS prefecture_name,` +
	`environments.title AS environment_title ` +
	`FROM "cityleague_results" ` +
	`LEFT JOIN official_events ON official_events.id = cityleague_results.official_event_id ` +
	`LEFT JOIN shops ON shops.id = official_events.shop_id ` +
	`LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id ` +
	`LEFT JOIN environments ON environments.from_date <= cityleague_results.event_date AND environments.to_date >= cityleague_results.event_date `

const findByPlayerIdOrder = ` ORDER BY cityleague_results.event_date DESC, cityleague_results.rank ASC, cityleague_results.official_event_id ASC`

var findByPlayerIdColumns = []string{
	"cityleague_schedule_id",
	"official_event_id",
	"league_type",
	"event_date",
	"rank",
	"point",
	"deck_code",
	"event_title",
	"shop_name",
	"prefecture_name",
	"environment_title",
}

// 期間が指定された場合、event_date の半開区間 [fromDate, toDate) で絞り込む。
func test_CityleagueResultInfrastructure_FindByPlayerId(t *testing.T) {
	r, mock, err := setup4CityleagueResultInfrastructure()
	require.NoError(t, err)

	fromDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local)
	toDate := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
	eventDate := time.Date(2026, 6, 7, 0, 0, 0, 0, time.Local)

	values := [][]driver.Value{
		{"250607", uint(952749), uint(1), eventDate, uint(1), uint(15), "gnnHHn-Vg3aWc-LHNnHH", "シティリーグ2026 シーズン4", "ポケモンカードステーション・渋谷", "東京都", "ニンジャスピナー"},
		{"250607", uint(952750), uint(1), eventDate, uint(3), uint(12), "xxxYYY-ZZZzzz-AAAbbb", "シティリーグ2026 シーズン4", "ポケモンカードジム・札幌", "北海道", "ニンジャスピナー"},
	}

	rows := sqlmock.NewRows(findByPlayerIdColumns)
	for _, value := range values {
		rows.AddRow(value...)
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		findByPlayerIdSelect+
			`WHERE cityleague_results.player_id = $1 AND cityleague_results.event_date >= $2 AND cityleague_results.event_date < $3`+
			findByPlayerIdOrder,
	)).WithArgs("1234567890", fromDate, toDate).WillReturnRows(rows)

	ret, err := r.FindByPlayerId(context.Background(), "1234567890", fromDate, toDate)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, ret, 2)
	require.Equal(t, "250607", ret[0].CityleagueScheduleId)
	require.Equal(t, uint(952749), ret[0].OfficialEventId)
	require.Equal(t, uint(1), ret[0].LeagueType)
	require.Equal(t, eventDate, ret[0].EventDate)
	require.Equal(t, uint(1), ret[0].Rank)
	require.Equal(t, uint(15), ret[0].Point)
	require.Equal(t, "gnnHHn-Vg3aWc-LHNnHH", ret[0].DeckCode)
	require.Equal(t, "シティリーグ2026 シーズン4", ret[0].EventTitle)
	require.Equal(t, "ポケモンカードステーション・渋谷", ret[0].ShopName)
	require.Equal(t, "東京都", ret[0].PrefectureName)
	require.Equal(t, "ニンジャスピナー", ret[0].EnvironmentTitle)
	require.Equal(t, uint(3), ret[1].Rank)
}

// 期間がゼロ値の場合、event_date の条件を付けず全期間を対象にする。
func test_CityleagueResultInfrastructure_FindByPlayerIdWithoutTerm(t *testing.T) {
	r, mock, err := setup4CityleagueResultInfrastructure()
	require.NoError(t, err)

	eventDate := time.Date(2026, 6, 7, 0, 0, 0, 0, time.Local)

	rows := sqlmock.NewRows(findByPlayerIdColumns).
		AddRow("250607", uint(952749), uint(1), eventDate, uint(1), uint(15), "gnnHHn-Vg3aWc-LHNnHH", "シティリーグ2026 シーズン4", "ポケモンカードステーション・渋谷", "東京都", "ニンジャスピナー")

	mock.ExpectQuery(regexp.QuoteMeta(
		findByPlayerIdSelect + `WHERE cityleague_results.player_id = $1` + findByPlayerIdOrder,
	)).WithArgs("1234567890").WillReturnRows(rows)

	ret, err := r.FindByPlayerId(context.Background(), "1234567890", time.Time{}, time.Time{})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, ret, 1)
}

// 入賞が無い場合は、エラーではなく空のスライスを返す
// (連携済みでまだ入賞していないユーザは正常系のため)。
func test_CityleagueResultInfrastructure_FindByPlayerIdReturnsEmptySlice(t *testing.T) {
	r, mock, err := setup4CityleagueResultInfrastructure()
	require.NoError(t, err)

	rows := sqlmock.NewRows(findByPlayerIdColumns)

	mock.ExpectQuery(regexp.QuoteMeta(findByPlayerIdSelect)).WillReturnRows(rows)

	ret, err := r.FindByPlayerId(context.Background(), "1234567890", time.Time{}, time.Time{})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.NotNil(t, ret)
	require.Empty(t, ret)
}

// official_events 側が引けない入賞でも、入賞そのものは落とさずに返す(LEFT JOIN)。
func test_CityleagueResultInfrastructure_FindByPlayerIdWithNullOfficialEvent(t *testing.T) {
	r, mock, err := setup4CityleagueResultInfrastructure()
	require.NoError(t, err)

	eventDate := time.Date(2026, 6, 7, 0, 0, 0, 0, time.Local)

	rows := sqlmock.NewRows(findByPlayerIdColumns).
		AddRow("250607", uint(952749), uint(1), eventDate, uint(1), uint(15), "gnnHHn-Vg3aWc-LHNnHH", nil, nil, nil, nil)

	mock.ExpectQuery(regexp.QuoteMeta(findByPlayerIdSelect)).WillReturnRows(rows)

	ret, err := r.FindByPlayerId(context.Background(), "1234567890", time.Time{}, time.Time{})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, ret, 1)
	require.Equal(t, uint(952749), ret[0].OfficialEventId)
	require.Empty(t, ret[0].EventTitle)
	require.Empty(t, ret[0].ShopName)
	require.Empty(t, ret[0].PrefectureName)
	require.Empty(t, ret[0].EnvironmentTitle)
}
