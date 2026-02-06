package infrastructure

/*
import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

func test_OfficialEventInfrastructure_Find(t *testing.T) {
	r, mock, err := setup4OfficialEventInfrastructure()
	require.NoError(t, err)

	date, _ := time.Parse(DateLayout, "2025-02-15T00:00:00Z")
	startedAt, _ := time.Parse(DateLayout, "2025-02-15T07:30:00Z")
	endedAt, _ := time.Parse(DateLayout, "2025-02-15T20:50:00Z")

	values := [][]driver.Value{
		{
			uint(606466),
			"チャンピオンズリーグ2025 福岡 マスターリーグ",
			"福岡県福岡市博多区沖浜町２−１",
			"マリンメッセ福岡　A館・B館",
			date,
			startedAt,
			endedAt,
			"",
			1,
			"大型大会",
			false,
			761,
			"マスター",
			1,
			"スタンダード",
			5000,
			3,
			0,
			"",
		},
		{
			uint(606467),
			"チャンピオンズリーグ2025 福岡 マスターリーグ グループA",
			"福岡県福岡市博多区沖浜町２−１",
			"マリンメッセ福岡　A館・B館",
			date,
			startedAt,
			endedAt,
			"",
			1,
			"大型大会",
			false,
			762,
			"マスター",
			1,
			"スタンダード",
			1700,
			3,
			0,
			"",
		},
		{
			uint(606468),
			"チャンピオンズリーグ2025 福岡 マスターリーグ グループB",
			"福岡県福岡市博多区沖浜町２−１",
			"マリンメッセ福岡　A館・B館",
			date,
			startedAt,
			endedAt,
			"",
			1,
			"大型大会",
			false,
			763,
			"マスター",
			1,
			"スタンダード",
			1700,
			3,
			0,
			"",
		},
		{
			uint(606469),
			"チャンピオンズリーグ2025 福岡 マスターリーグ グループC",
			"福岡県福岡市博多区沖浜町２−１",
			"マリンメッセ福岡　A館・B館",
			date,
			startedAt,
			endedAt,
			"",
			1,
			"大型大会",
			false,
			764,
			"マスター",
			1,
			"スタンダード",
			1700,
			3,
			0,
			"",
		},
	}
	rows := sqlmock.NewRows([]string{
		"id",
		"title",
		"address",
		"venue",
		"date",
		"started_at",
		"ended_at",
		"deck_count",
		"type_id",
		"type_name",
		"csp_flg",
		"league_id",
		"league_title",
		"regulation_id",
		"regulation_title",
		"capacity",
		"attr_id",
		"shop_id",
		"shop_name",
	}).AddRows(values...)

	t.Run("クエリパターン_#01", func(t *testing.T) {
		typeId := uint(0)
		leagueType := uint(4)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "official_events" WHERE league_title = $1 AND date BETWEEN $2 AND $3 ORDER BY started_at ASC`,
		)).WithArgs(
			"マスター",
			date,
			date,
		).WillReturnRows(rows)

		_, err := r.Find(context.Background(), typeId, leagueType, date, date)

		require.NoError(t, err)
	})

	t.Run("クエリパターン_#02", func(t *testing.T) {
		typeId := uint(0)
		leagueType := uint(0)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "official_events" WHERE date BETWEEN $1 AND $2 ORDER BY started_at ASC`,
		)).WithArgs(
			date,
			date,
		).WillReturnRows(rows)

		_, err := r.Find(context.Background(), typeId, leagueType, date, date)

		require.NoError(t, err)
	})

	t.Run("クエリパターン_#03", func(t *testing.T) {
		typeId := uint(1)
		leagueType := uint(4)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "official_events" WHERE type_id = $1 AND league_title = $2 AND date BETWEEN $3 AND $4 ORDER BY started_at ASC`,
		)).WithArgs(
			typeId,
			"マスター",
			date,
			date,
		).WillReturnRows(rows)

		_, err := r.Find(context.Background(), typeId, leagueType, date, date)

		require.NoError(t, err)
	})

	t.Run("クエリパターン_#04", func(t *testing.T) {
		typeId := uint(1)
		leagueType := uint(0)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "official_events" WHERE type_id = $1 AND date BETWEEN $2 AND $3 ORDER BY started_at ASC`,
		)).WithArgs(
			typeId,
			date,
			date,
		).WillReturnRows(rows)

		_, err := r.Find(context.Background(), typeId, leagueType, date, date)

		require.NoError(t, err)
	})
}

func test_OfficialEventInfrastructure_FindById(t *testing.T) {
	r, mock, _ := setup4OfficialEventInfrastructure()

	// https://players.pokemon-card.com/event_detail_search?event_holding_id=606466
	id := uint(606466)
	date, _ := time.Parse(DateLayout, "2025-02-15T00:00:00Z")
	startedAt, _ := time.Parse(DateLayout, "2025-02-15T07:30:00Z")
	endedAt, _ := time.Parse(DateLayout, "2025-02-15T20:50:00Z")

	rows := sqlmock.NewRows([]string{
		"id",
		"title",
		"address",
		"venue",
		"date",
		"started_at",
		"ended_at",
		"deck_count",
		"type_id",
		"type_name",
		"csp_flg",
		"league_id",
		"league_title",
		"regulation_id",
		"regulation_title",
		"capacity",
		"attr_id",
		"shop_id",
		"shop_name",
	}).AddRow(
		id,
		"チャンピオンズリーグ2025 福岡 マスターリーグ",
		"福岡県福岡市博多区沖浜町２−１",
		"マリンメッセ福岡　A館・B館",
		date,
		startedAt,
		endedAt,
		"",
		1,
		"大型大会",
		false,
		761,
		"マスター",
		1,
		"スタンダード",
		5000,
		3,
		0,
		"",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "official_events" WHERE id = $1 ORDER BY "official_events"."id" LIMIT $2`,
	)).WithArgs(
		id,
		1,
	).WillReturnRows(rows)

	officialEvent, err := r.FindById(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, officialEvent.ID)
	require.Equal(t, date, officialEvent.Date)
}
*/
