package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
)

var playerRankingColumns = []string{
	"ranking_date", "league_id", "player_id", "nickname", "current_ranking",
	"prefecture_name", "champion_ship_point", "public_flg", "champion_flg", "avatar_image",
}

func TestPlayerRankingInfrastructure(t *testing.T) {
	playerId := "1234567890123456"

	t.Run("正常系_最新のランキング日の行を返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewPlayerRanking(db)

		rankingDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)

		mock.ExpectQuery(`SELECT \* FROM "player_rankings" WHERE player_id = \$1 ORDER BY ranking_date DESC`).
			WithArgs(playerId, 1).
			WillReturnRows(sqlmock.NewRows(playerRankingColumns).AddRow(
				rankingDate, 1, playerId, "ニックネーム", 100, "福岡県", 250, true, false, "https://example.com/avatar.png",
			))

		ret, err := r.FindLatestByPlayerId(context.Background(), playerId)

		require.NoError(t, err)
		require.Equal(t, playerId, ret.PlayerId)
		require.Equal(t, rankingDate, ret.RankingDate)
		require.Equal(t, 250, ret.ChampionShipPoint)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_ランキング履歴が無ければErrRecordNotFoundへ変換する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewPlayerRanking(db)

		mock.ExpectQuery(`SELECT \* FROM "player_rankings"`).
			WithArgs(playerId, 1).WillReturnRows(sqlmock.NewRows(playerRankingColumns))

		ret, err := r.FindLatestByPlayerId(context.Background(), playerId)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Nil(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
