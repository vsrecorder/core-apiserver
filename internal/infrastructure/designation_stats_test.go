package infrastructure

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDesignationStatsInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	playerId := "1234567890123456"
	fromDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.Local)
	toDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)

	userCountColumns := []string{"user_id", "count"}

	t.Run("CountRecordsByUserId", func(t *testing.T) {
		t.Run("正常系_対戦とデッキを持つ記録のみを数えて返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewDesignationStats(db)

			// 対戦(matches)とデッキ登録の存在をEXISTSで要求する
			mock.ExpectQuery(`SELECT count\(\*\) FROM "records" WHERE \(user_id = \$1 AND deleted_at IS NULL AND ignore_stats_flg = false\) AND \(EXISTS \(SELECT 1 FROM matches`).
				WithArgs(uid, fromDate, toDate).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

			count, err := r.CountRecordsByUserId(context.Background(), uid, fromDate, toDate)

			require.NoError(t, err)
			require.Equal(t, 3, count)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_集計エラーをそのまま返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewDesignationStats(db)

			mock.ExpectQuery(`SELECT count\(\*\) FROM "records"`).WillReturnError(sql.ErrConnDone)

			count, err := r.CountRecordsByUserId(context.Background(), uid, fromDate, toDate)

			require.Error(t, err)
			require.Zero(t, count)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("CountRecordsAsOfByUserId", func(t *testing.T) {
		t.Run("正常系_asOf時点で成立していた記録のみを数えて返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewDesignationStats(db)

			asOf := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)

			// asOf以前に対戦・デッキが揃っていたことをmatches.created_atと
			// decks/deck_codesのcreated_atで検証する条件が含まれる
			mock.ExpectQuery(`SELECT count\(\*\) FROM "records".*matches\.created_at < \$2.*COALESCE\(deck_registered_at, created_at\) < \$3.*decks\.created_at < \$4.*deck_codes\.created_at < \$5.*event_date < \$6`).
				WithArgs(uid, asOf, asOf, asOf, asOf, asOf, fromDate).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

			count, err := r.CountRecordsAsOfByUserId(context.Background(), uid, fromDate, asOf)

			require.NoError(t, err)
			require.Equal(t, 2, count)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("CountCityLeagueRecordsByUserId", func(t *testing.T) {
		t.Run("正常系_公式イベント種別がシティリーグの記録を数えて返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewDesignationStats(db)

			mock.ExpectQuery(`SELECT count\(\*\) FROM "records" JOIN official_events ON official_events\.id = records\.official_event_id`).
				WithArgs(uid, 2, fromDate, toDate).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

			count, err := r.CountCityLeagueRecordsByUserId(context.Background(), uid, fromDate, toDate)

			require.NoError(t, err)
			require.Equal(t, 2, count)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("CountLeagueRecordsByUserId", func(t *testing.T) {
		t.Run("正常系_シティリーグとトレーナーズリーグの記録を数えて返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewDesignationStats(db)

			mock.ExpectQuery(`SELECT count\(\*\) FROM "records" JOIN official_events ON official_events\.id = records\.official_event_id.*type_id IN \(\$2, \$3\)`).
				WithArgs(uid, 2, 3, fromDate, toDate).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

			count, err := r.CountLeagueRecordsByUserId(context.Background(), uid, fromDate, toDate)

			require.NoError(t, err)
			require.Equal(t, 5, count)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("CountRecordsGroupByUserId", func(t *testing.T) {
		t.Run("正常系_ユーザごとの記録数をマップで返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewDesignationStats(db)

			mock.ExpectQuery(`SELECT user_id AS user_id, COUNT\(\*\) AS count FROM "records".*GROUP BY "user_id"`).
				WithArgs(fromDate, toDate).
				WillReturnRows(sqlmock.NewRows(userCountColumns).
					AddRow("user-1", 3).
					AddRow("user-2", 1),
				)

			counts, err := r.CountRecordsGroupByUserId(context.Background(), fromDate, toDate)

			require.NoError(t, err)
			require.Equal(t, map[string]int{"user-1": 3, "user-2": 1}, counts)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("CountCityLeagueRecordsGroupByUserId", func(t *testing.T) {
		t.Run("正常系_ユーザごとのシティリーグ記録数をマップで返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewDesignationStats(db)

			mock.ExpectQuery(`SELECT records\.user_id AS user_id, COUNT\(\*\) AS count FROM "records" JOIN official_events`).
				WithArgs(2, fromDate, toDate).
				WillReturnRows(sqlmock.NewRows(userCountColumns).AddRow("user-1", 2))

			counts, err := r.CountCityLeagueRecordsGroupByUserId(context.Background(), fromDate, toDate)

			require.NoError(t, err)
			require.Equal(t, map[string]int{"user-1": 2}, counts)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("ExistsCityLeagueResultByPlayerId", func(t *testing.T) {
		t.Run("正常系_一致する結果と自身の記録があればtrueを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewDesignationStats(db)

			// cityleague_resultsと同じ公式イベントの自身の記録の存在をEXISTSで要求する
			mock.ExpectQuery(`SELECT count\(\*\) FROM "cityleague_results" WHERE player_id = \$1 AND \(EXISTS \(SELECT 1 FROM records`).
				WithArgs(playerId, uid, fromDate, toDate, 1).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			exists, err := r.ExistsCityLeagueResultByPlayerId(context.Background(), uid, playerId, fromDate, toDate)

			require.NoError(t, err)
			require.True(t, exists)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_該当がなければfalseを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewDesignationStats(db)

			mock.ExpectQuery(`SELECT count\(\*\) FROM "cityleague_results"`).
				WithArgs(playerId, uid, fromDate, toDate, 1).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			exists, err := r.ExistsCityLeagueResultByPlayerId(context.Background(), uid, playerId, fromDate, toDate)

			require.NoError(t, err)
			require.False(t, exists)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("ExistsCityLeagueResultGroupByUserId", func(t *testing.T) {
		t.Run("正常系_連携済みユーザごとの達成有無をマップで返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewDesignationStats(db)

			mock.ExpectQuery(`SELECT DISTINCT users_players\.user_id AS user_id, 1 AS count FROM "cityleague_results" JOIN users_players`).
				WithArgs(fromDate, toDate).
				WillReturnRows(sqlmock.NewRows(userCountColumns).AddRow("user-1", 1))

			counts, err := r.ExistsCityLeagueResultGroupByUserId(context.Background(), fromDate, toDate)

			require.NoError(t, err)
			require.Equal(t, map[string]int{"user-1": 1}, counts)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
