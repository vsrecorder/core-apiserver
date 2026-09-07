package infrastructure

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// matchSummaryColumns は FindSummariesByRecordIds のクエリが返すカラム。
var matchSummaryColumns = []string{
	"record_id",
	"total",
	"wins",
	"draws",
	"group_match_count",
	"bo3_count",
}

// matchSummaryQuery は FindSummariesByRecordIds が発行するSQL。
// IN句のプレースホルダ数は記録IDの数で変わるため、件数から組み立てる。
func matchSummaryQuery(recordIdCount int) string {
	placeholders := make([]string, 0, recordIdCount)
	for i := 1; i <= recordIdCount; i++ {
		placeholders = append(placeholders, "$"+strconv.Itoa(i))
	}

	return `SELECT records.id AS record_id, ` +
		`COUNT(matches.id) AS total, ` +
		`COUNT(CASE WHEN matches.victory_flg THEN 1 END) AS wins, ` +
		`COUNT(CASE WHEN matches.draw_flg THEN 1 END) AS draws, ` +
		`COUNT(CASE WHEN matches.group_match_flg THEN 1 END) AS group_match_count, ` +
		`COUNT(CASE WHEN matches.bo3_flg THEN 1 END) AS bo3_count ` +
		`FROM "records" ` +
		`LEFT JOIN matches ON records.id = matches.record_id AND matches.deleted_at IS NULL ` +
		`WHERE records.id IN (` + strings.Join(placeholders, ",") + `) ` +
		`AND records.user_id = $` + strconv.Itoa(recordIdCount+1) + ` ` +
		`AND records.deleted_at IS NULL ` +
		`GROUP BY "records"."id"`
}

func test_MatchInfrastructure_FindSummariesByRecordIds(t *testing.T) {
	uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"
	recordId1 := "01HD7Y3K8D6FDHMHTZ2GT41TR1"
	recordId2 := "01HD7Y3K8D6FDHMHTZ2GT41TR2"
	recordId3 := "01HD7Y3K8D6FDHMHTZ2GT41TR3"

	t.Run("正常系_勝敗数とチーム戦BO3の有無を記録ごとに集計する", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		rows := sqlmock.NewRows(matchSummaryColumns).
			// 5戦3勝1分け → 負けは 5-3-1=1。BO3が含まれチーム戦は無い
			AddRow(recordId1, 5, 3, 1, 0, 2).
			// 対戦が1件も無い記録も total=0 として返る
			AddRow(recordId2, 0, 0, 0, 0, 0)

		mock.ExpectQuery(regexp.QuoteMeta(matchSummaryQuery(2))).
			WithArgs(recordId1, recordId2, uid).
			WillReturnRows(rows)

		summaries, err := r.FindSummariesByRecordIds(context.Background(), uid, []string{recordId1, recordId2})

		require.NoError(t, err)
		require.Len(t, summaries, 2)

		require.Equal(t, recordId1, summaries[0].RecordId)
		require.Equal(t, 5, summaries[0].Total)
		require.Equal(t, 3, summaries[0].Wins)
		require.Equal(t, 1, summaries[0].Draws)
		require.Equal(t, 1, summaries[0].Losses)
		require.False(t, summaries[0].HasGroupMatch)
		require.True(t, summaries[0].HasBo3)

		require.Equal(t, recordId2, summaries[1].RecordId)
		require.Equal(t, 0, summaries[1].Total)
		require.Equal(t, 0, summaries[1].Losses)
		require.False(t, summaries[1].HasGroupMatch)
		require.False(t, summaries[1].HasBo3)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 認可はクエリの records.user_id で行うため、他人の記録・存在しない記録は
	// そもそも行として返らず、結果から落ちる(エラーにはしない)。
	t.Run("正常系_他人の記録や存在しない記録は結果から除外される", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		// 3件指定したうち、自分が所有しているのは recordId2 だけという想定
		rows := sqlmock.NewRows(matchSummaryColumns).AddRow(recordId2, 2, 2, 0, 2, 0)

		mock.ExpectQuery(regexp.QuoteMeta(matchSummaryQuery(3))).
			WithArgs(recordId1, recordId2, recordId3, uid).
			WillReturnRows(rows)

		summaries, err := r.FindSummariesByRecordIds(
			context.Background(), uid, []string{recordId1, recordId2, recordId3},
		)

		require.NoError(t, err)
		require.Len(t, summaries, 1)
		require.Equal(t, recordId2, summaries[0].RecordId)
		require.Equal(t, 2, summaries[0].Wins)
		require.True(t, summaries[0].HasGroupMatch)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// GROUP BY の結果順はDB任せのため、要求した record_ids の順に並べ直す。
	t.Run("正常系_要求した記録IDの順に並べ替えて返す", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		rows := sqlmock.NewRows(matchSummaryColumns).
			AddRow(recordId3, 1, 1, 0, 0, 0).
			AddRow(recordId1, 1, 0, 0, 0, 0).
			AddRow(recordId2, 1, 0, 1, 0, 1)

		mock.ExpectQuery(regexp.QuoteMeta(matchSummaryQuery(3))).
			WithArgs(recordId1, recordId2, recordId3, uid).
			WillReturnRows(rows)

		summaries, err := r.FindSummariesByRecordIds(
			context.Background(), uid, []string{recordId1, recordId2, recordId3},
		)

		require.NoError(t, err)
		require.Len(t, summaries, 3)
		require.Equal(t, recordId1, summaries[0].RecordId)
		require.Equal(t, recordId2, summaries[1].RecordId)
		require.Equal(t, recordId3, summaries[2].RecordId)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_記録IDが空ならクエリを投げずに空を返す", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		summaries, err := r.FindSummariesByRecordIds(context.Background(), uid, nil)

		require.NoError(t, err)
		require.Empty(t, summaries)
		// クエリが1本も発行されていないこと
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_クエリが失敗したらエラーを返す", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(regexp.QuoteMeta(matchSummaryQuery(1))).
			WithArgs(recordId1, uid).
			WillReturnError(errors.New("query failed"))

		summaries, err := r.FindSummariesByRecordIds(context.Background(), uid, []string{recordId1})

		require.Error(t, err)
		require.Nil(t, summaries)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
