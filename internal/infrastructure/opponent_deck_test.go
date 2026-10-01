package infrastructure

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// 期待するSQL。自分の(論理削除されていない記録の)対戦を、表記 × 1体目 × 2体目で束ねる。
// 期間・集計対象外(ignore_stats_flg)・不戦勝では絞らず、表記もスプライトも無い対戦だけを除く。
const opponentDeckFindByUserIdQuery = `SELECT matches.opponents_deck_info AS opponents_deck_info, COALESCE(s1.pokemon_sprite_id, '') AS sprite1, COALESCE(s2.pokemon_sprite_id, '') AS sprite2, COUNT(*) AS count, TO_CHAR(MAX(records.event_date), 'YYYY-MM-DD') AS last_event_date FROM "matches" JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL LEFT JOIN match_pokemon_sprites s1 ON s1.match_id = matches.id AND s1.position = 1 LEFT JOIN match_pokemon_sprites s2 ON s2.match_id = matches.id AND s2.position = 2 WHERE (matches.deleted_at IS NULL AND matches.user_id = $1) AND (matches.opponents_deck_info <> '' OR s1.pokemon_sprite_id IS NOT NULL OR s2.pokemon_sprite_id IS NOT NULL) GROUP BY matches.opponents_deck_info, sprite1, sprite2 ORDER BY count DESC, last_event_date DESC, opponents_deck_info ASC LIMIT $2`

func TestOpponentDeckInfrastructure_FindByUserId(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	columns := []string{"opponents_deck_info", "sprite1", "sprite2", "count", "last_event_date"}

	t.Run("正常系_組み合わせごとの対戦の数をスプライト付きで返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		i := NewOpponentDeck(db)

		rows := sqlmock.NewRows(columns).
			AddRow("ドラパルトex", "0887", "0006", 12, "2026-09-28").
			AddRow("サナ", "", "0282", 3, "2026-09-01").
			AddRow("謎のデッキ", "", "", 1, "2026-08-01")

		mock.ExpectQuery(regexp.QuoteMeta(opponentDeckFindByUserIdQuery)).
			WithArgs(uid, maxOpponentDecksPerUser).
			WillReturnRows(rows)

		ret, err := i.FindByUserId(context.Background(), uid)

		require.NoError(t, err)
		require.Len(t, ret, 3)
		require.Equal(t, 12, ret[0].Count)
		require.Equal(t, "2026-09-28", ret[0].LastEventDate)
		require.Equal(t, "0887", ret[0].SpriteIdAt(1))
		require.Equal(t, "0006", ret[0].SpriteIdAt(2))
		// 2体目だけのスプライトは position 2 のまま
		require.Len(t, ret[1].PokemonSprites, 1)
		require.Equal(t, "0282", ret[1].SpriteIdAt(2))
		require.Empty(t, ret[2].PokemonSprites)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_クエリのエラーを返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		i := NewOpponentDeck(db)

		mock.ExpectQuery(regexp.QuoteMeta(opponentDeckFindByUserIdQuery)).
			WithArgs(uid, maxOpponentDecksPerUser).
			WillReturnError(errors.New("db error"))

		_, err := i.FindByUserId(context.Background(), uid)

		require.Error(t, err)
	})
}
