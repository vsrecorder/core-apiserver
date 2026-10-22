package infrastructure

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestSiteStatInfrastructure(t *testing.T) {
	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_3つの件数を1回のクエリで返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewSiteStat(db)

			mock.ExpectQuery(regexp.QuoteMeta(siteStatQuery)).WillReturnRows(
				sqlmock.NewRows([]string{"user_count", "match_count", "deck_code_count"}).
					AddRow(int64(812), int64(15234), int64(4021)),
			)

			stat, err := r.Find(context.Background())

			require.NoError(t, err)
			require.Equal(t, int64(812), stat.UserCount)
			require.Equal(t, int64(15234), stat.MatchCount)
			require.Equal(t, int64(4021), stat.DeckCodeCount)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_DBエラーはそのまま返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewSiteStat(db)

			mock.ExpectQuery(regexp.QuoteMeta(siteStatQuery)).WillReturnError(errors.New("connection refused"))

			stat, err := r.Find(context.Background())

			require.Error(t, err)
			require.Nil(t, stat)
		})
	})
}
