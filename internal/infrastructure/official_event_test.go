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

func setupMock4OfficialEventInfrastructure() (*gorm.DB, sqlmock.Sqlmock, error) {
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

func setup4OfficialEventInfrastructure() (repository.OfficialEventInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock4OfficialEventInfrastructure()

	if err != nil {
		return nil, nil, err
	}

	r := NewOfficialEvent(db)

	return r, mock, err
}

// officialEventColumns は official_events に shops / prefectures / environments /
// standard_regulations を JOIN した結果のカラム(model.OfficialEvent に対応)。
var officialEventColumns = []string{
	"id",
	"title",
	"address",
	"venue",
	"date",
	"started_at",
	"ended_at",
	"type_id",
	"type_name",
	"league_title",
	"regulation_title",
	"csp_flg",
	"capacity",
	"shop_id",
	"shop_name",
	"prefecture_id",
	"prefecture_name",
	"environment_id",
	"environment_title",
	"standard_regulation_id",
	"standard_regulation_marks",
}

// officialEventQuery は公式イベントを引くクエリにマッチする正規表現を組み立てる。
// SELECT句・JOIN句はGoのソース上の改行やインデントをそのままSQLに含むため、
// 検証したいWHERE以降(絞り込み条件・並び順)だけを完全一致で見る。
func officialEventQuery(tail string) string {
	return `(?s)SELECT.*FROM "official_events".*LEFT JOIN shops ON shops\.id = official_events\.shop_id.*` +
		`LEFT JOIN prefectures.*LEFT JOIN environments.*LEFT JOIN standard_regulations.*` + regexp.QuoteMeta(tail)
}

func TestOfficialEventInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"Find":     test_OfficialEventInfrastructure_Find,
		"FindById": test_OfficialEventInfrastructure_FindById,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

// https://players.pokemon-card.com/event_detail_search?event_holding_id=606466 を模したデータ
func officialEventRows(date, startedAt, endedAt time.Time) *sqlmock.Rows {
	return sqlmock.NewRows(officialEventColumns).AddRow(
		uint(606466),
		"チャンピオンズリーグ2025 福岡 マスターリーグ",
		"福岡県福岡市博多区沖浜町２−１",
		"マリンメッセ福岡　A館・B館",
		date,
		startedAt,
		endedAt,
		uint(1),
		"大型大会",
		"マスター",
		"スタンダード",
		true,
		uint(5000),
		uint(0),
		"",
		uint(40),
		"福岡県",
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		"レギュレーションG",
		"01HD7Y3K8D6FDHMHTZ2GT41TN3",
		"G,H,I",
	)
}

func test_OfficialEventInfrastructure_Find(t *testing.T) {
	date := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)
	startedAt := time.Date(2025, 2, 15, 7, 30, 0, 0, time.UTC)
	endedAt := time.Date(2025, 2, 15, 20, 50, 0, 0, time.UTC)

	// 大会の種類の指定なし・リーグの指定あり(マスター)
	t.Run("クエリパターン_#01", func(t *testing.T) {
		r, mock, err := setup4OfficialEventInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(officialEventQuery(
			`WHERE league_title = $1 AND date BETWEEN $2 AND $3 ORDER BY started_at ASC`,
		)).WithArgs(
			"マスター",
			date,
			date,
		).WillReturnRows(officialEventRows(date, startedAt, endedAt))

		events, err := r.Find(context.Background(), uint(0), uint(4), date, date)

		require.NoError(t, err)
		require.Equal(t, 1, len(events))
		require.Equal(t, uint(606466), events[0].ID)
		require.Equal(t, "マスター", events[0].LeagueTitle)
		require.Equal(t, date, events[0].Date)
		require.Equal(t, "福岡県", events[0].PrefectureName)
		require.Equal(t, "レギュレーションG", events[0].EnvironmentTitle)
		require.Equal(t, "G,H,I", events[0].StandardRegulationMarks)
		require.True(t, events[0].CSPFlg)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 大会の種類・リーグともに指定なし
	t.Run("クエリパターン_#02", func(t *testing.T) {
		r, mock, err := setup4OfficialEventInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(officialEventQuery(
			`WHERE date BETWEEN $1 AND $2 ORDER BY started_at ASC`,
		)).WithArgs(
			date,
			date,
		).WillReturnRows(officialEventRows(date, startedAt, endedAt))

		events, err := r.Find(context.Background(), uint(0), uint(0), date, date)

		require.NoError(t, err)
		require.Equal(t, 1, len(events))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 大会の種類の指定あり(大型大会)・リーグの指定あり(マスター)
	t.Run("クエリパターン_#03", func(t *testing.T) {
		r, mock, err := setup4OfficialEventInfrastructure()
		require.NoError(t, err)

		typeId := uint(1)

		mock.ExpectQuery(officialEventQuery(
			`WHERE type_id = $1 AND league_title = $2 AND date BETWEEN $3 AND $4 ORDER BY started_at ASC`,
		)).WithArgs(
			typeId,
			"マスター",
			date,
			date,
		).WillReturnRows(officialEventRows(date, startedAt, endedAt))

		events, err := r.Find(context.Background(), typeId, uint(4), date, date)

		require.NoError(t, err)
		require.Equal(t, 1, len(events))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 大会の種類の指定あり(大型大会)・リーグの指定なし
	t.Run("クエリパターン_#04", func(t *testing.T) {
		r, mock, err := setup4OfficialEventInfrastructure()
		require.NoError(t, err)

		typeId := uint(1)

		mock.ExpectQuery(officialEventQuery(
			`WHERE type_id = $1 AND date BETWEEN $2 AND $3 ORDER BY started_at ASC`,
		)).WithArgs(
			typeId,
			date,
			date,
		).WillReturnRows(officialEventRows(date, startedAt, endedAt))

		events, err := r.Find(context.Background(), typeId, uint(0), date, date)

		require.NoError(t, err)
		require.Equal(t, 1, len(events))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// リーグの指定は1〜4のみ有効で、範囲外は指定なしとして扱われる
	t.Run("クエリパターン_#05", func(t *testing.T) {
		r, mock, err := setup4OfficialEventInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(officialEventQuery(
			`WHERE date BETWEEN $1 AND $2 ORDER BY started_at ASC`,
		)).WithArgs(
			date,
			date,
		).WillReturnRows(officialEventRows(date, startedAt, endedAt))

		events, err := r.Find(context.Background(), uint(0), uint(5), date, date)

		require.NoError(t, err)
		require.Equal(t, 1, len(events))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// リーグの指定値ごとに、対応するリーグ名で絞り込む
	t.Run("正常系_#01", func(t *testing.T) {
		for leagueType, leagueTitle := range map[uint]string{
			1: "オープン",
			2: "ジュニア",
			3: "シニア",
			4: "マスター",
		} {
			r, mock, err := setup4OfficialEventInfrastructure()
			require.NoError(t, err)

			mock.ExpectQuery(officialEventQuery(
				`WHERE league_title = $1 AND date BETWEEN $2 AND $3 ORDER BY started_at ASC`,
			)).WithArgs(
				leagueTitle,
				date,
				date,
			).WillReturnRows(sqlmock.NewRows(officialEventColumns))

			_, err = r.Find(context.Background(), uint(0), leagueType, date, date)

			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		}
	})

	// 該当イベントが無い場合
	t.Run("正常系_#02", func(t *testing.T) {
		r, mock, err := setup4OfficialEventInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(officialEventQuery(
			`WHERE date BETWEEN $1 AND $2 ORDER BY started_at ASC`,
		)).WithArgs(
			date,
			date,
		).WillReturnRows(sqlmock.NewRows(officialEventColumns))

		events, err := r.Find(context.Background(), uint(0), uint(0), date, date)

		require.NoError(t, err)
		require.Empty(t, events)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_OfficialEventInfrastructure_FindById(t *testing.T) {
	date := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)
	startedAt := time.Date(2025, 2, 15, 7, 30, 0, 0, time.UTC)
	endedAt := time.Date(2025, 2, 15, 20, 50, 0, 0, time.UTC)

	t.Run("正常系_#01", func(t *testing.T) {
		r, mock, err := setup4OfficialEventInfrastructure()
		require.NoError(t, err)

		id := uint(606466)

		mock.ExpectQuery(officialEventQuery(
			`WHERE official_events.id = $1`,
		)).WithArgs(
			id,
		).WillReturnRows(officialEventRows(date, startedAt, endedAt))

		officialEvent, err := r.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, officialEvent.ID)
		require.Equal(t, "チャンピオンズリーグ2025 福岡 マスターリーグ", officialEvent.Title)
		require.Equal(t, date, officialEvent.Date)
		require.Equal(t, startedAt, officialEvent.StartedAt)
		require.Equal(t, endedAt, officialEvent.EndedAt)
		require.Equal(t, uint(1), officialEvent.TypeId)
		require.Equal(t, uint(5000), officialEvent.Capacity)
		require.Equal(t, uint(40), officialEvent.PrefectureId)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
