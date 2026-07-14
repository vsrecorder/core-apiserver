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

func setup4BadgeStatsInfrastructure() (repository.BadgeStatsInterface, sqlmock.Sqlmock, error) {
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

	return NewBadgeStats(db), mock, nil
}

// exactQuery は発行SQLとの完全一致を要求する正規表現を返す。sqlmock の既定マッチャは
// 部分一致のため、アンカー無しだと期待SQLの後ろに条件が足されても(例: ignore_stats_flg =
// false の復活)マッチしてしまい、回帰を検知できない。
func exactQuery(query string) string {
	return `^` + regexp.QuoteMeta(query) + `$`
}

// バッジ(はじめの一歩・マイルストーン)と週次ストリークは、集計対象外(ignore_stats_flg=true)の
// 記録も活動量として数える。分析系の集計と違い、これらのクエリに ignore_stats_flg の条件が
// 混入していないことを、発行SQLの完全一致で固定する。
const badgeStatsCountRecordsQuery = `SELECT count(*) FROM "records" WHERE user_id = $1 AND deleted_at IS NULL`

const badgeStatsCountMatchesQuery = `SELECT count(*) FROM "matches" JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL WHERE matches.user_id = $1 AND matches.deleted_at IS NULL`

const badgeStatsFindRecordDatesQuery = `SELECT event_date, created_at FROM "records" WHERE user_id = $1 AND deleted_at IS NULL`

const badgeStatsFindMatchDatesQuery = `SELECT "matches"."created_at" FROM "matches" JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL WHERE matches.user_id = $1 AND matches.deleted_at IS NULL ORDER BY matches.created_at ASC`

func TestBadgeStatsInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"CountRecordsIncludesIgnoredRecords":    test_BadgeStatsInfrastructure_CountRecordsIncludesIgnoredRecords,
		"CountMatchesIncludesIgnoredRecords":    test_BadgeStatsInfrastructure_CountMatchesIncludesIgnoredRecords,
		"FindRecordDatesIncludesIgnoredRecords": test_BadgeStatsInfrastructure_FindRecordDatesIncludesIgnoredRecords,
		"FindMatchDatesIncludesIgnoredRecords":  test_BadgeStatsInfrastructure_FindMatchDatesIncludesIgnoredRecords,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_BadgeStatsInfrastructure_CountRecordsIncludesIgnoredRecords(t *testing.T) {
	i, mock, err := setup4BadgeStatsInfrastructure()
	require.NoError(t, err)

	userId := "user-01"

	mock.ExpectQuery(exactQuery(badgeStatsCountRecordsQuery)).
		WithArgs(userId).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := i.CountRecordsByUserId(context.Background(), userId, time.Time{}, time.Time{})

	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func test_BadgeStatsInfrastructure_CountMatchesIncludesIgnoredRecords(t *testing.T) {
	i, mock, err := setup4BadgeStatsInfrastructure()
	require.NoError(t, err)

	userId := "user-01"

	mock.ExpectQuery(exactQuery(badgeStatsCountMatchesQuery)).
		WithArgs(userId).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := i.CountMatchesByUserId(context.Background(), userId, time.Time{}, time.Time{})

	require.NoError(t, err)
	require.Equal(t, 5, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func test_BadgeStatsInfrastructure_FindRecordDatesIncludesIgnoredRecords(t *testing.T) {
	i, mock, err := setup4BadgeStatsInfrastructure()
	require.NoError(t, err)

	userId := "user-01"
	eventDate := time.Date(2026, 7, 4, 0, 0, 0, 0, time.Local)
	createdAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.Local)

	// 1件目は event_date 入力あり、2件目は未入力(created_atが基準日になる)
	rows := sqlmock.NewRows([]string{"event_date", "created_at"}).
		AddRow(eventDate, createdAt).
		AddRow(time.Time{}, createdAt)

	mock.ExpectQuery(exactQuery(badgeStatsFindRecordDatesQuery)).
		WithArgs(userId).
		WillReturnRows(rows)

	dates, err := i.FindRecordDatesByUserId(context.Background(), userId, time.Time{}, time.Time{})

	require.NoError(t, err)
	require.Len(t, dates, 2)
	require.Equal(t, eventDate, dates[0])
	require.Equal(t, createdAt, dates[1])
	require.NoError(t, mock.ExpectationsWereMet())
}

func test_BadgeStatsInfrastructure_FindMatchDatesIncludesIgnoredRecords(t *testing.T) {
	i, mock, err := setup4BadgeStatsInfrastructure()
	require.NoError(t, err)

	userId := "user-01"
	createdAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.Local)

	mock.ExpectQuery(exactQuery(badgeStatsFindMatchDatesQuery)).
		WithArgs(userId).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(createdAt))

	dates, err := i.FindMatchDatesByUserId(context.Background(), userId, time.Time{}, time.Time{})

	require.NoError(t, err)
	require.Len(t, dates, 1)
	require.Equal(t, createdAt, dates[0])
	require.NoError(t, mock.ExpectationsWereMet())
}
