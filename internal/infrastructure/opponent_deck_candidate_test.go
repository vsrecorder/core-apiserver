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

func setup4OpponentDeckCandidateInfrastructure() (repository.OpponentDeckCandidateInterface, sqlmock.Sqlmock, error) {
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

	return NewOpponentDeckCandidate(db), mock, nil
}

// 期待するSQL。不戦勝・不戦敗と表記が空の対戦を除き、論理削除された記録・対戦と
// 集計対象外(ignore_stats_flg)の記録の対戦を数えず、記録の private_flg は条件に含めない。
const opponentDeckCandidateSelect = `SELECT matches.opponents_deck_info AS opponents_deck_info, COALESCE(s1.pokemon_sprite_id, '') AS sprite1, COALESCE(s2.pokemon_sprite_id, '') AS sprite2, COUNT(*) AS count, MAX(matches.created_at) AS last_used_at FROM "matches" JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL AND records.ignore_stats_flg = false LEFT JOIN match_pokemon_sprites s1 ON s1.match_id = matches.id AND s1.position = 1 LEFT JOIN match_pokemon_sprites s2 ON s2.match_id = matches.id AND s2.position = 2 WHERE (matches.deleted_at IS NULL AND matches.default_victory_flg = false AND matches.default_defeat_flg = false AND matches.opponents_deck_info <> '') AND matches.created_at >= $1`

const opponentDeckCandidateGroupBy = ` GROUP BY matches.opponents_deck_info, sprite1, sprite2 ORDER BY count DESC, last_used_at DESC, opponents_deck_info ASC LIMIT `

// 全ユーザーを対象にする場合(UserId 指定なし)
const opponentDeckCandidateAllUsersQuery = opponentDeckCandidateSelect + opponentDeckCandidateGroupBy + `$2`

// 自身の履歴を対象にする場合(UserId 指定あり)
const opponentDeckCandidateOwnQuery = opponentDeckCandidateSelect + ` AND matches.user_id = $2` + opponentDeckCandidateGroupBy + `$3`

func TestOpponentDeckCandidateInfrastructure_FindOpponentDeckCandidates(t *testing.T) {
	since := time.Date(2026, 6, 19, 0, 0, 0, 0, time.Local)
	lastUsedAt := time.Date(2026, 9, 16, 12, 0, 0, 0, time.Local)

	candidateColumns := []string{"opponents_deck_info", "sprite1", "sprite2", "count", "last_used_at"}

	t.Run("正常系_全ユーザーを対象に出現回数順の候補をスプライト付きで返す", func(t *testing.T) {
		i, mock, err := setup4OpponentDeckCandidateInfrastructure()
		require.NoError(t, err)

		rows := sqlmock.NewRows(candidateColumns).
			AddRow("ロストバレット", "0887", "0006", 12, lastUsedAt).
			AddRow("サーナイトex", "0282", "", 7, lastUsedAt).
			AddRow("謎のデッキ", "", "", 1, lastUsedAt)

		mock.ExpectQuery(regexp.QuoteMeta(opponentDeckCandidateAllUsersQuery)).
			WithArgs(since, 100).
			WillReturnRows(rows)

		ret, err := i.FindOpponentDeckCandidates(context.Background(), &repository.OpponentDeckCandidateFilter{
			Since: since,
			Limit: 100,
		})

		require.NoError(t, err)
		require.Len(t, ret, 3)

		require.Equal(t, "ロストバレット", ret[0].OpponentsDeckInfo)
		require.Equal(t, 12, ret[0].Count)
		require.Len(t, ret[0].PokemonSprites, 2)
		require.Equal(t, "0887", ret[0].PokemonSprites[0].ID)
		require.Equal(t, uint(1), ret[0].PokemonSprites[0].Position)
		require.Equal(t, "0006", ret[0].PokemonSprites[1].ID)
		require.Equal(t, uint(2), ret[0].PokemonSprites[1].Position)

		// 2体目が無いときは1体目だけ
		require.Len(t, ret[1].PokemonSprites, 1)
		require.Equal(t, uint(1), ret[1].PokemonSprites[0].Position)

		// スプライト未設定は空
		require.Empty(t, ret[2].PokemonSprites)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 自身の履歴からの候補。他人の対戦が混ざらないよう matches.user_id で絞る
	t.Run("正常系_UserId指定時はそのユーザーの対戦だけを集計する", func(t *testing.T) {
		i, mock, err := setup4OpponentDeckCandidateInfrastructure()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		mock.ExpectQuery(regexp.QuoteMeta(opponentDeckCandidateOwnQuery)).
			WithArgs(since, uid, 50).
			WillReturnRows(sqlmock.NewRows(candidateColumns).AddRow("ロストバレット", "0887", "", 3, lastUsedAt))

		ret, err := i.FindOpponentDeckCandidates(context.Background(), &repository.OpponentDeckCandidateFilter{
			UserId: uid,
			Since:  since,
			Limit:  50,
		})

		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, "ロストバレット", ret[0].OpponentsDeckInfo)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_該当なしは空スライスを返す", func(t *testing.T) {
		i, mock, err := setup4OpponentDeckCandidateInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(regexp.QuoteMeta(opponentDeckCandidateAllUsersQuery)).
			WithArgs(since, 10).
			WillReturnRows(sqlmock.NewRows(candidateColumns))

		ret, err := i.FindOpponentDeckCandidates(context.Background(), &repository.OpponentDeckCandidateFilter{
			Since: since,
			Limit: 10,
		})

		require.NoError(t, err)
		require.Empty(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_クエリ失敗はエラーを返す", func(t *testing.T) {
		i, mock, err := setup4OpponentDeckCandidateInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(regexp.QuoteMeta(opponentDeckCandidateAllUsersQuery)).
			WithArgs(since, 10).
			WillReturnError(sqlmock.ErrCancelled)

		ret, err := i.FindOpponentDeckCandidates(context.Background(), &repository.OpponentDeckCandidateFilter{
			Since: since,
			Limit: 10,
		})

		require.Error(t, err)
		require.Nil(t, ret)
	})
}
